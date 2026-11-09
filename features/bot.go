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
	nudgeCooldown = 30 * time.Minute // 대화 트리거(음식, 시간): 트리거별·채널별
	memeCooldown  = 3 * time.Minute  // 밈 자동 반응: 채널별
	yachtJoin     = 60 * time.Second // 야추 판 참가 모집 시간
)

// custom_id: "<기능>:<동작>[:인자]" (interactions 패키지 형식)
const (
	featNudge     = "nudge"
	featYacht     = "yacht"
	actJoin       = "join" // yacht:join      [🙋 참가]
	actPoll       = "poll" // nudge:poll[:음식] [⏰ 시간 투표]
	actMute       = "mute" // nudge:mute:<NudgeKind> [🔕 오늘 그만]
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

	switch kind, food := DetectNudge(m.Content); {
	case kind == NudgeYacht && cfg.Has(config.Yacht):
		b.openYacht(s, m.ChannelID, m.Author.ID, cfg.Tone) // 직접 부른 거라 쿨다운 없음
		return
	case kind != NoNudge && kind != NudgeYacht && cfg.Has(config.Nudge):
		if b.Nudges.Allow(string(kind), m.ChannelID, now) {
			b.sendNudge(s, m.ChannelID, kind, food, cfg.Tone)
		}
		return // 트리거에 걸린 말에는 밈 반응을 겹쳐 하지 않는다
	}
	if cfg.Has(config.AutoReply) && IsMeme(m.Content) && b.Memes.Allow(memeTriggerID, m.ChannelID, now) {
		if _, err := s.ChannelMessageSend(m.ChannelID, voice.Say(voice.MemeReply, cfg.Tone, b.Pick)); err != nil {
			log.Printf("밈 반응 실패: %v", err)
		}
	}
}

// sendNudge 는 음식·시간 얘기에 [⏰ 시간 투표] [🔕 오늘 그만] 버튼을 단 제안을 올린다.
// 음식 이름은 투표 버튼의 custom_id 인자로 실어 보낸다 (투표 제목에 쓴다).
func (b *Bot) sendNudge(s *discordgo.Session, channelID string, kind NudgeKind, food string, tone config.Tone) {
	var content string
	switch kind {
	case NudgeFood:
		content = NudgeFoodText(food, tone, b.Pick)
	case NudgeTime:
		content, food = voice.Say(voice.NudgeTime, tone, b.Pick), ""
	default:
		return
	}
	msg := &discordgo.MessageSend{
		Content: content,
		Components: row(
			button(discordgo.PrimaryButton, "⏰", voice.Say(voice.TimeBtnLabel, tone, b.Pick), featNudge, actPoll, food),
			button(discordgo.SecondaryButton, "🔕", voice.Say(voice.MuteBtnLabel, tone, b.Pick), featNudge, actMute, string(kind)),
		),
	}
	if _, err := s.ChannelMessageSendComplex(channelID, msg); err != nil {
		log.Printf("nudge(%s) 전송 실패: %v", kind, err)
	}
}

// openYacht 는 "야추겜 ㄱㄱ"에 바로 참가 모집 판을 올린다. 말한 사람은 자동 참가한다.
func (b *Bot) openYacht(s *discordgo.Session, channelID, user string, tone config.Tone) {
	if b.Reg.Busy(user) {
		if _, err := s.ChannelMessageSend(channelID, voice.Say(voice.AlreadyInGame, tone, b.Pick)); err != nil {
			log.Printf("야추 판 중복 안내 실패: %v", err)
		}
		return
	}
	msg, err := s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Content:    RenderLobby([]string{user}, tone, b.Pick),
		Components: b.joinRow(tone),
	})
	if err != nil {
		log.Printf("야추 판 게시 실패: %v", err)
		return
	}
	_, err = b.Reg.Start(msg.ID, channelID, []string{user}, nil, yachtJoin, func(g *session.Game) {
		b.resolveYacht(s, g, tone)
	})
	if err != nil { // 게시와 등록 사이에 같은 사람이 다른 판에 들어간 드문 경우
		log.Printf("야추 판 등록 실패: %v", err)
		b.edit(s, channelID, msg.ID, voice.Say(voice.AlreadyInGame, tone, b.Pick))
	}
}

// OnInteraction 은 InteractionCreate 핸들러다. 버튼만 처리한다.
func (b *Bot) OnInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionMessageComponent || i.Message == nil {
		return
	}
	id, err := interactions.Decode(i.MessageComponentData().CustomID)
	if err != nil {
		return
	}
	user := interactionUser(i)
	if user == "" {
		return
	}
	cfg, _ := config.Lookup(i.ChannelID)
	now := b.Now()

	switch {
	case id.Feature == featYacht && id.Action == actJoin && cfg.Has(config.Yacht):
		b.joinYacht(s, i, user, cfg.Tone)
	case id.Feature == featNudge && id.Action == actPoll && cfg.Has(config.Nudge):
		b.postTimePoll(s, i, id.Arg, cfg.Tone)
	case id.Feature == featNudge && id.Action == actMute && cfg.Has(config.Nudge):
		b.Nudges.MuteToday(id.Arg, i.ChannelID, now)
		b.update(s, i, voice.Say(voice.NudgeMuted, cfg.Tone, b.Pick), nil)
	case id.Feature == featYacht || id.Feature == featNudge:
		// 예전 버전이 올린 버튼(예: 옛 nudge:chicken)이나 꺼진 기능: 버튼을 끄고 안내한다.
		b.expired(s, i, cfg.Tone)
	}
}

func (b *Bot) joinYacht(s *discordgo.Session, i *discordgo.InteractionCreate, user string, tone config.Tone) {
	msgID := i.Message.ID
	switch err := b.Reg.Join(msgID, user); {
	case errors.Is(err, session.ErrNotFound):
		b.expired(s, i, tone)
	case errors.Is(err, session.ErrAlreadyIn):
		b.ack(s, i)
	case errors.Is(err, session.ErrBusy):
		b.ephemeral(s, i, voice.Say(voice.AlreadyInGame, tone, b.Pick))
	case err != nil:
		log.Printf("야추 판 참가 실패: %v", err)
		b.ack(s, i)
	default:
		b.update(s, i, RenderLobby(b.Reg.Users(msgID), tone, b.Pick), b.joinRow(tone))
	}
}

// resolveYacht 는 참가 모집 시간이 끝나면 한 번 불린다 (session 이 1회를 보장).
func (b *Bot) resolveYacht(s *discordgo.Session, g *session.Game, tone config.Tone) {
	content := voice.Say(voice.YachtCancel, tone, b.Pick)
	if res, err := yacht.Quick(g.Players, b.Roll, yacht.Pick(b.Pick)); err == nil {
		content = RenderResult(res, tone, b.Pick)
	}
	b.edit(s, g.ChannelID, g.MessageID, content)
}

// postTimePoll 은 디스코드 기본 투표(복수 선택)를 올린다. food 가 있으면 제목에 넣는다.
func (b *Bot) postTimePoll(s *discordgo.Session, i *discordgo.InteractionCreate, food string, tone config.Tone) {
	now := b.Now()
	var answers []discordgo.PollAnswer
	for _, slot := range TimeSlots(now) {
		answers = append(answers, discordgo.PollAnswer{Media: &discordgo.PollMedia{Text: SlotLabel(slot, now, tone, b.Pick)}})
	}
	_, err := s.ChannelMessageSendComplex(i.ChannelID, &discordgo.MessageSend{
		Poll: &discordgo.Poll{
			Question:         discordgo.PollMedia{Text: PollTitle(food, tone, b.Pick)},
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
	return row(button(discordgo.SuccessButton, "🙋", voice.Say(voice.JoinBtnLabel, tone, b.Pick), featYacht, actJoin, ""))
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

func button(style discordgo.ButtonStyle, emoji, label, feature, action, arg string) discordgo.Button {
	id, err := interactions.Encode(interactions.ID{Feature: feature, Action: action, Arg: arg})
	if err != nil {
		panic(err) // 고정된 상수 조합이라 여기 오면 프로그래밍 실수다
	}
	return discordgo.Button{Style: style, Label: label, CustomID: id, Emoji: &discordgo.ComponentEmoji{Name: emoji}}
}
