package features

import (
	"errors"
	"log"
	"math/rand/v2"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"zzezze-bot/config"
	"zzezze-bot/interactions"
	"zzezze-bot/session"
	"zzezze-bot/voice"
	"zzezze-bot/yacht"
)

// ChickenJoinWindow 는 치킨 판 참가 모집 시간이다. 참가해도 늘어나지 않는다(Touch 하지 않음).
const ChickenJoinWindow = 60 * time.Second

const chickenFeature = "chicken"

// Chicken 은 치킨 판(모집 → 주사위 정산)의 디스코드 연결이다. Roll 과 Pick 은 테스트에서 바꾼다.
type Chicken struct {
	reg  *session.Registry
	Roll func() int // 1~6
	Pick voice.Picker
}

func NewChicken(reg *session.Registry) *Chicken {
	return &Chicken{
		reg:  reg,
		Roll: func() int { return rand.IntN(6) + 1 },
		Pick: rand.IntN,
	}
}

// chickenState 는 Registry 에 담기는 판 상태다. 바꾸는 곳은 session.Registry.Do 안뿐이다.
type chickenState struct {
	channelID string
	tone      config.Tone
	header    string   // 모집 시작 때 고른 ChickenOpen 대사(참가할 때마다 바뀌지 않게 고정)
	players   []string // 참가 순서
	joined    []string // players 와 같은 순서의 참가 대사(역시 고정)
}

// closeInfo 는 종료 처리(W2)가 "봇 재시작" 안내를 달 채널과 말투를 얻는 데 쓴다.
func (st *chickenState) closeInfo() (channelID string, tone config.Tone) {
	return st.channelID, st.tone
}

func (st *chickenState) add(user string, pick voice.Picker) {
	st.players = append(st.players, user)
	st.joined = append(st.joined, voice.Say(voice.ChickenJoined, st.tone, pick, "{user}", mention(user)))
}

// openScreen 은 모집 화면이다: 머리말 + 참가자별 참가 대사.
func (st *chickenState) openScreen() string {
	return st.header + "\n" + strings.Join(st.joined, "\n")
}

// chickenOpen 은 제안 메시지를 판으로 등록하고 모집 화면 문자열을 돌려준다. 누른 사람이 첫 참가자다.
func chickenOpen(reg *session.Registry, msgID, channelID, user string, tone config.Tone, pick voice.Picker, onExpire func(string, any)) (string, error) {
	st := &chickenState{
		channelID: channelID,
		tone:      tone,
		header:    voice.Say(voice.ChickenOpen, tone, pick),
	}
	st.add(user, pick)
	err := reg.Start(msgID, session.Game{
		State:    st,
		Idle:     ChickenJoinWindow,
		OnExpire: onExpire,
		Users:    []string{user},
	})
	if err != nil {
		return "", err
	}
	return st.openScreen(), nil
}

// chickenJoin 은 참가자를 등록하고 갱신된 모집 화면을 돌려준다. 방치 타이머는 건드리지 않는다.
func chickenJoin(reg *session.Registry, msgID, user string, pick voice.Picker) (string, error) {
	if err := reg.Join(msgID, user); err != nil {
		return "", err
	}
	var text string
	// 상태만 바꾸고 I/O 는 하지 않는다. Join 과 Do 사이에 판이 끝나면 ErrNotFound 다.
	err := reg.Do(msgID, func(state any) {
		st := state.(*chickenState)
		st.add(user, pick)
		text = st.openScreen()
	})
	return text, err
}

// chickenRejectKey 는 open/join 실패를 누른 사람에게 보일 대사 키로 바꾼다.
func chickenRejectKey(err error) voice.Key {
	switch {
	case errors.Is(err, session.ErrUserBusy),
		errors.Is(err, session.ErrAlreadyJoined),
		errors.Is(err, session.ErrExists):
		return voice.AlreadyInGame
	default: // ErrNotFound, ErrClosed 등: 끝났거나 닫힌 판
		return voice.SessionExpired
	}
}

// chickenSettle 은 참가자로 주사위를 굴려 최종 화면 문자열을 만든다. 2명 미만이면 취소 문구다.
func chickenSettle(players []string, tone config.Tone, roll func() int, pick voice.Picker) string {
	res, err := yacht.PlayChicken(players, roll, pick)
	if err != nil {
		if !errors.Is(err, yacht.ErrNotEnoughPlayers) {
			// 참가 등록이 중복을 막으므로 정상 흐름에서는 나오지 않는다. 그래도 판이 멈추지 않게 취소로 닫는다.
			log.Printf("치킨 정산 실패: %v", err)
		}
		return voice.Say(voice.ChickenCancel, tone, pick)
	}
	return chickenResultText(res, tone, pick)
}

// chickenResultText 는 라운드별 주사위·합계와 당첨자를 한 문자열로 그린다.
// 두 번째 라운드부터는 앞에 동점 문구를 한 줄 넣는다. ByPick(무작위 결정)은 따로 표시하지 않는다.
func chickenResultText(res yacht.ChickenResult, tone config.Tone, pick voice.Picker) string {
	var lines []string
	for i, round := range res.Rounds {
		if i > 0 {
			lines = append(lines, voice.Say(voice.ChickenTie, tone, pick))
		}
		for _, r := range round.Rolls {
			var dice strings.Builder
			for _, d := range r.Dice {
				dice.WriteString(voice.Die(d))
			}
			lines = append(lines, mention(r.Player)+" "+dice.String()+" = "+strconv.Itoa(r.Sum))
		}
	}
	lines = append(lines, voice.Say(voice.ChickenLoser, tone, pick, "{user}", mention(res.Loser)))
	return strings.Join(lines, "\n")
}

// chickenJoinRow 는 모집 화면 버튼([참가])이다.
func chickenJoinRow(tone config.Tone, pick voice.Picker) []discordgo.MessageComponent {
	id, err := interactions.Encode(interactions.ID{Feature: chickenFeature, Action: "join"})
	if err != nil { // 상수 입력이라 일어나지 않는다
		log.Printf("chicken:join custom_id 인코딩 실패: %v", err)
		return nil
	}
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    voice.Say(voice.JoinBtnLabel, tone, pick),
				Style:    discordgo.PrimaryButton,
				CustomID: id,
			},
		}},
	}
}

// HandleComponent 는 chicken:open, chicken:join 버튼을 처리한다.
func (c *Chicken) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate, id interactions.ID) {
	if i.Message == nil {
		return
	}
	cfg, _ := config.Lookup(i.ChannelID)
	tone := cfg.Tone
	if tone == "" {
		tone = config.ToneMild
	}
	user := userID(i)

	switch id.Action {
	case "open":
		// 정산은 만료 타이머가 부르므로 s 를 캡처한다. 상태가 필요하면 state 로 받는다.
		onExpire := func(msgID string, state any) {
			st, ok := state.(*chickenState)
			if !ok {
				log.Printf("치킨 만료: 상태 타입이 다르다 (msg=%s)", msgID)
				return
			}
			text := chickenSettle(st.players, st.tone, c.Roll, c.Pick)
			if err := editMessage(s, st.channelID, msgID, text, nil); err != nil {
				log.Printf("치킨 결과 표시 실패 (msg=%s): %v", msgID, err)
			}
		}
		text, err := chickenOpen(c.reg, i.Message.ID, i.ChannelID, user, tone, c.Pick, onExpire)
		if err != nil {
			replyEphemeral(s, i, voice.Say(chickenRejectKey(err), tone, c.Pick))
			return
		}
		updateMessage(s, i, text, chickenJoinRow(tone, c.Pick))
	case "join":
		text, err := chickenJoin(c.reg, i.Message.ID, user, c.Pick)
		if err != nil {
			replyEphemeral(s, i, voice.Say(chickenRejectKey(err), tone, c.Pick))
			return
		}
		updateMessage(s, i, text, chickenJoinRow(tone, c.Pick))
	default:
		log.Printf("chicken: 모르는 동작 %q", id.Action)
	}
}
