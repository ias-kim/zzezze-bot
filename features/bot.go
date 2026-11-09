package features

import (
	"errors"
	"log"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"

	"zzezze-bot/config"
	"zzezze-bot/interactions"
	"zzezze-bot/session"
	"zzezze-bot/voice"
	"zzezze-bot/yacht"
)

// 시간 값. 근거는 docs/design/six-member.md.
const (
	nudgeCooldown = 30 * time.Minute // 대화 트리거: 트리거별·채널별
	memeCooldown  = 3 * time.Minute  // 밈 자동 반응: 채널별
	chickenJoin   = 60 * time.Second // 음식값 판 참가 모집 시간
)

// custom_id: "nudge:<동작>[:인자]" (interactions 패키지 형식)
const (
	featNudge     = "nudge"
	actChicken    = "chicken" // [🎲 야추로 정하기] → 참가 모집 시작
	actJoin       = "join"    // [🙋 참가]
	actPoll       = "poll"    // [⏰ 시간 투표]
	actMute       = "mute"    // [🔕 오늘 그만], 인자 = NudgeKind
	memeTriggerID = "meme"
)

// Bot 은 디스코드 이벤트를 받아 순수 로직으로 넘기는 연결 계층이다.
type Bot struct {
	Reg    *session.Registry
	Nudges *session.Throttle
	Memes  *session.Throttle
	Now    func() time.Time
	Pick   voice.Picker
	Roll   yacht.Roller
}

// New 는 운영용 Bot 을 만든다 (진짜 시계와 난수).
func New() *Bot {
	return &Bot{
		Reg:    session.New(),
		Nudges: session.NewThrottle(nudgeCooldown),
		Memes:  session.NewThrottle(memeCooldown),
		Now:    time.Now,
		Pick:   rand.IntN,
		Roll:   func() int { return rand.IntN(6) + 1 },
	}
}

// OnMessage 는 MessageCreate 핸들러다.
func (b *Bot) OnMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author == nil || Ignore(m.Content, m.Author.Bot) {
		return
	}
	cfg, ok := config.Lookup(m.ChannelID)
	if !ok {
		return
	}
	now := b.Now()

	if cfg.Has(config.Nudge) {
		if kind, food := DetectNudge(m.Content); kind != NoNudge {
			if b.Nudges.Allow(string(kind), m.ChannelID, now) {
				b.sendNudge(s, m.ChannelID, kind, food, cfg.Tone)
			}
			return // 트리거에 걸린 말에는 밈 반응을 겹쳐 하지 않는다
		}
	}
	if cfg.Has(config.AutoReply) && IsMeme(m.Content) && b.Memes.Allow(memeTriggerID, m.ChannelID, now) {
		if _, err := s.ChannelMessageSend(m.ChannelID, voice.Say(voice.MemeReply, cfg.Tone, b.Pick)); err != nil {
			log.Printf("밈 반응 실패: %v", err)
		}
	}
}

func (b *Bot) sendNudge(s *discordgo.Session, channelID string, kind NudgeKind, food string, tone config.Tone) {
	var msg *discordgo.MessageSend
	switch kind {
	case NudgeFood:
		msg = &discordgo.MessageSend{
			Content: NudgeText(food, tone, b.Pick),
			Components: row(
				// 음식 이름은 버튼 custom_id 인자로 실어 보낸다 (판을 열 때 꺼내 쓴다).
				button(discordgo.PrimaryButton, "🎲", voice.Say(voice.ChickenBtnLabel, tone, b.Pick), actChicken, food),
				button(discordgo.SecondaryButton, "🔕", voice.Say(voice.MuteBtnLabel, tone, b.Pick), actMute, string(kind)),
			),
		}
	case NudgeTime:
		msg = &discordgo.MessageSend{
			Content: voice.Say(voice.NudgeTime, tone, b.Pick),
			Components: row(
				button(discordgo.PrimaryButton, "⏰", voice.Say(voice.TimeBtnLabel, tone, b.Pick), actPoll, ""),
				button(discordgo.SecondaryButton, "🔕", voice.Say(voice.MuteBtnLabel, tone, b.Pick), actMute, string(kind)),
			),
		}
	default:
		return
	}
	if _, err := s.ChannelMessageSendComplex(channelID, msg); err != nil {
		log.Printf("nudge(%s) 전송 실패: %v", kind, err)
	}
}

// OnInteraction 은 InteractionCreate 핸들러다. 버튼만 처리한다.
func (b *Bot) OnInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionMessageComponent || i.Message == nil {
		return
	}
	id, err := interactions.Decode(i.MessageComponentData().CustomID)
	if err != nil || id.Feature != featNudge {
		return
	}
	user := interactionUser(i)
	if user == "" {
		return
	}
	cfg, ok := config.Lookup(i.ChannelID)
	if !ok || !cfg.Has(config.Nudge) {
		b.ephemeral(s, i, voice.Say(voice.WrongChannel, cfg.Tone, b.Pick))
		return
	}

	switch id.Action {
	case actChicken:
		b.openChicken(s, i, user, id.Arg, cfg.Tone)
	case actJoin:
		b.joinChicken(s, i, user, cfg.Tone)
	case actPoll:
		b.postTimePoll(s, i, cfg.Tone)
	case actMute:
		b.Nudges.MuteToday(id.Arg, i.ChannelID, b.Now())
		b.update(s, i, voice.Say(voice.NudgeMuted, cfg.Tone, b.Pick), nil)
	}
}

// openChicken 은 참가 모집을 시작한다. 판 상태(Game.State)에는 걸린 음식 이름(string)을 둔다.
func (b *Bot) openChicken(s *discordgo.Session, i *discordgo.InteractionCreate, user, food string, tone config.Tone) {
	msgID, channelID := i.Message.ID, i.ChannelID
	_, err := b.Reg.Start(msgID, channelID, []string{user}, food, chickenJoin, func(g *session.Game) {
		b.resolveChicken(s, g, tone)
	})
	switch {
	case errors.Is(err, session.ErrExists):
		b.joinChicken(s, i, user, tone) // 동시에 누른 사람은 참가로 친다
		return
	case errors.Is(err, session.ErrBusy):
		b.ephemeral(s, i, voice.Say(voice.AlreadyInGame, tone, b.Pick))
		return
	case err != nil:
		log.Printf("치킨 판 시작 실패: %v", err)
		b.ephemeral(s, i, voice.Say(voice.SessionExpired, tone, b.Pick))
		return
	}
	b.update(s, i, RenderLobby(food, b.Reg.Users(msgID), tone, b.Pick), b.joinRow(tone))
}

// gameFood 는 판에 걸린 음식 이름이다. State 는 Start 때 한 번 정해지고 바뀌지 않는다.
func gameFood(g *session.Game) string {
	food, _ := g.State.(string)
	return food
}

func (b *Bot) joinChicken(s *discordgo.Session, i *discordgo.InteractionCreate, user string, tone config.Tone) {
	msgID := i.Message.ID
	switch err := b.Reg.Join(msgID, user); {
	case errors.Is(err, session.ErrNotFound):
		b.expired(s, i, tone)
	case errors.Is(err, session.ErrAlreadyIn):
		b.ack(s, i)
	case errors.Is(err, session.ErrBusy):
		b.ephemeral(s, i, voice.Say(voice.AlreadyInGame, tone, b.Pick))
	case err != nil:
		log.Printf("치킨 판 참가 실패: %v", err)
		b.ack(s, i)
	default:
		food := ""
		if g, ok := b.Reg.Get(msgID); ok {
			food = gameFood(g)
		}
		b.update(s, i, RenderLobby(food, b.Reg.Users(msgID), tone, b.Pick), b.joinRow(tone))
	}
}

// resolveChicken 은 참가 모집 시간이 끝나면 한 번 불린다 (session 이 1회를 보장).
func (b *Bot) resolveChicken(s *discordgo.Session, g *session.Game, tone config.Tone) {
	content := voice.Say(voice.ChickenCancel, tone, b.Pick)
	if res, err := yacht.Chicken(g.Players, b.Roll, yacht.Pick(b.Pick)); err == nil {
		content = RenderChicken(gameFood(g), res, tone, b.Pick)
	}
	b.edit(s, g.ChannelID, g.MessageID, content)
}

func (b *Bot) postTimePoll(s *discordgo.Session, i *discordgo.InteractionCreate, tone config.Tone) {
	now := b.Now()
	var answers []discordgo.PollAnswer
	for _, slot := range TimeSlots(now) {
		answers = append(answers, discordgo.PollAnswer{Media: &discordgo.PollMedia{Text: SlotLabel(slot, now, tone, b.Pick)}})
	}
	_, err := s.ChannelMessageSendComplex(i.ChannelID, &discordgo.MessageSend{
		Poll: &discordgo.Poll{
			Question:         discordgo.PollMedia{Text: voice.Say(voice.TimePollTitle, tone, b.Pick)},
			Answers:          answers,
			AllowMultiselect: true,
			Duration:         pollHours,
		},
	})
	if err != nil {
		log.Printf("시간 투표 게시 실패: %v", err)
	}
	b.update(s, i, i.Message.Content, nil) // 버튼만 지운다
}

// Shutdown 은 진행 중인 판의 버튼을 끄고 재시작 안내를 띄운다. timeout 이 지나면 기다리지 않는다.
func (b *Bot) Shutdown(s *discordgo.Session, timeout time.Duration) {
	games := b.Reg.CloseAll()
	var wg sync.WaitGroup
	for _, g := range games {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tone := config.Default.Tone
			if cfg, ok := config.Lookup(g.ChannelID); ok {
				tone = cfg.Tone
			}
			b.edit(s, g.ChannelID, g.MessageID, voice.Say(voice.SessionRestart, tone, b.Pick))
		}()
	}
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(timeout):
		log.Printf("종료: %d개 판 정리 중 시간 초과", len(games))
	}
}

// ── 디스코드 응답 도우미 ──────────────────────────────────────

func (b *Bot) joinRow(tone config.Tone) []discordgo.MessageComponent {
	return row(button(discordgo.SuccessButton, "🙋", voice.Say(voice.JoinBtnLabel, tone, b.Pick), actJoin, ""))
}

// update 는 버튼이 달린 메시지를 바꾼다. components 가 nil 이면 버튼을 모두 지운다.
func (b *Bot) update(s *discordgo.Session, i *discordgo.InteractionCreate, content string, components []discordgo.MessageComponent) {
	if components == nil {
		components = []discordgo.MessageComponent{}
	}
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{Content: content, Components: components},
	})
	if err != nil {
		log.Printf("버튼 응답 실패: %v", err)
	}
}

// expired 는 끝난 판의 버튼을 눌렀을 때: 버튼을 끄고 본인에게만 안내한다.
func (b *Bot) expired(s *discordgo.Session, i *discordgo.InteractionCreate, tone config.Tone) {
	b.update(s, i, i.Message.Content, nil)
	_, err := s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
		Content: voice.Say(voice.SessionExpired, tone, b.Pick),
		Flags:   discordgo.MessageFlagsEphemeral,
	})
	if err != nil {
		log.Printf("만료 안내 실패: %v", err)
	}
}

func (b *Bot) ephemeral(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: content, Flags: discordgo.MessageFlagsEphemeral},
	})
	if err != nil {
		log.Printf("ephemeral 응답 실패: %v", err)
	}
}

// ack 는 화면을 바꾸지 않고 클릭만 받는다.
func (b *Bot) ack(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate})
	if err != nil {
		log.Printf("버튼 ack 실패: %v", err)
	}
}

// edit 는 타이머·종료처럼 상호작용 토큰 없이 메시지를 바꿀 때 쓴다. 버튼은 모두 지운다.
func (b *Bot) edit(s *discordgo.Session, channelID, messageID, content string) {
	e := discordgo.NewMessageEdit(channelID, messageID).SetContent(content)
	e.Components = &[]discordgo.MessageComponent{}
	if _, err := s.ChannelMessageEditComplex(e); err != nil {
		log.Printf("메시지 수정 실패(%s): %v", messageID, err)
	}
}

func interactionUser(i *discordgo.InteractionCreate) string {
	switch {
	case i.Member != nil && i.Member.User != nil:
		return i.Member.User.ID
	case i.User != nil:
		return i.User.ID
	}
	return ""
}

func row(buttons ...discordgo.MessageComponent) []discordgo.MessageComponent {
	return []discordgo.MessageComponent{discordgo.ActionsRow{Components: buttons}}
}

func button(style discordgo.ButtonStyle, emoji, label, action, arg string) discordgo.Button {
	id, err := interactions.Encode(interactions.ID{Feature: featNudge, Action: action, Arg: arg})
	if err != nil {
		panic(err) // 고정된 상수 조합이라 여기 오면 프로그래밍 실수다
	}
	return discordgo.Button{Style: style, Label: label, CustomID: id, Emoji: &discordgo.ComponentEmoji{Name: emoji}}
}
