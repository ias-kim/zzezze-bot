package features

import (
	"log"
	"math/rand/v2"
	"regexp"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"zzezze-bot/config"
	"zzezze-bot/interactions"
	"zzezze-bot/quiet"
	"zzezze-bot/voice"
)

// NudgeCooldown 은 같은 채널에서 같은 트리거(chicken/time)의 제안 사이 최소 간격이다.
const NudgeCooldown = 30 * time.Minute

// nudge 트리거 이름. 쿨다운·음소거 키이자 [오늘 그만] 버튼의 인자다.
const (
	nudgeTriggerChicken = "chicken"
	nudgeTriggerTime    = "time"
)

// custom_id 의 feature 이름. 버튼을 받는 쪽(B 치킨, C 시간 투표)과 맞춘 값이다.
const (
	nudgeFeature       = "nudge"
	nudgeActionMute    = "mute"
	nudgeChickenFeat   = "chicken"
	nudgeTimeVoteFeat  = "timevote"
	nudgeActionOpenBtn = "open"
)

var (
	// 치킨: "치킨"이 있고 + 의도 표현이 함께 있을 때만. "어제 치킨 맛있었다" 같은 말은 걸러진다.
	nudgeChickenWord   = regexp.MustCompile(`치킨`)
	nudgeChickenIntent = regexp.MustCompile(`시킬|시켜|먹을|ㄱ|뭐먹|\?|？`)

	// 시간: "몇 시"는 뒤에 "간"이 오면 "몇 시간"(걸린 시간)이므로 제외한다.
	// Go regexp 는 lookahead 가 없어서 "시" 뒤에 '간이 아닌 글자 또는 끝'을 요구하는 식으로 쓴다.
	nudgeTimePattern = regexp.MustCompile(`몇\s*시([^간]|$)|언제\s*(봄|볼|만나|만날)`)
)

// nudgeTrigger 는 메시지가 어떤 트리거에 걸리는지만 판단한다(채널·쿨다운은 보지 않는다).
// 둘 다 걸리면 치킨을 우선한다.
func nudgeTrigger(content string) (string, bool) {
	if nudgeChickenWord.MatchString(content) && nudgeChickenIntent.MatchString(content) {
		return nudgeTriggerChicken, true
	}
	if nudgeTimePattern.MatchString(content) {
		return nudgeTriggerTime, true
	}
	return "", false
}

// Nudge 는 2번방 대화에서 "야추로 정하기"/"시간 투표"를 제안한다. Pick 과 Now 는 테스트에서 바꾼다.
type Nudge struct {
	gate *quiet.Gate
	Pick voice.Picker
	Now  func() time.Time
}

func NewNudge(gate *quiet.Gate) *Nudge {
	return &Nudge{gate: gate, Pick: rand.IntN, Now: time.Now}
}

// Decide 는 이 메시지에 제안을 달아야 하는지 판단하고, 그렇다면 트리거 이름을 돌려준다.
// Nudge 가 꺼진 채널, 봇 메시지, 명령어(/ !)는 무시한다.
// 쿨다운·음소거는 gate.Allow 가 처리하며, 트리거에 걸린 경우에만 부른다(안 걸린 메시지가 쿨다운을 쓰지 않게).
func (n *Nudge) Decide(channelID string, fromBot bool, content string) (string, bool) {
	if fromBot || strings.HasPrefix(content, "/") || strings.HasPrefix(content, "!") {
		return "", false
	}
	if !config.Enabled(channelID, config.Nudge) {
		return "", false
	}
	trigger, ok := nudgeTrigger(content)
	if !ok {
		return "", false
	}
	if !n.gate.Allow(channelID, trigger, n.Now()) {
		return "", false
	}
	return trigger, true
}

// nudgeReply 는 제안 대사와 버튼 한 줄을 만든다. 순수 함수라 디스코드 없이 테스트한다.
func nudgeReply(trigger string, tone config.Tone, pick voice.Picker) (string, []discordgo.MessageComponent) {
	var (
		text      voice.Key
		openLabel voice.Key
		openID    interactions.ID
	)
	switch trigger {
	case nudgeTriggerChicken:
		text, openLabel = voice.NudgeChicken, voice.ChickenBtnLabel
		openID = interactions.ID{Feature: nudgeChickenFeat, Action: nudgeActionOpenBtn}
	case nudgeTriggerTime:
		text, openLabel = voice.NudgeTime, voice.TimeBtnLabel
		openID = interactions.ID{Feature: nudgeTimeVoteFeat, Action: nudgeActionOpenBtn}
	default:
		return "", nil
	}
	muteID := interactions.ID{Feature: nudgeFeature, Action: nudgeActionMute, Arg: trigger}

	openCustom, err1 := interactions.Encode(openID)
	muteCustom, err2 := interactions.Encode(muteID)
	if err1 != nil || err2 != nil {
		// 상수로 만든 ID 라 일어나지 않는다. 일어나면 코드 버그이므로 버튼 없이 대사만 보낸다.
		log.Printf("nudge custom_id 인코딩 실패: %v %v", err1, err2)
		return voice.Say(text, tone, pick), nil
	}
	row := discordgo.ActionsRow{Components: []discordgo.MessageComponent{
		discordgo.Button{Label: voice.Say(openLabel, tone, pick), Style: discordgo.PrimaryButton, CustomID: openCustom},
		discordgo.Button{Label: voice.Say(voice.MuteBtnLabel, tone, pick), Style: discordgo.SecondaryButton, CustomID: muteCustom},
	}}
	return voice.Say(text, tone, pick), []discordgo.MessageComponent{row}
}

// Handle 은 MessageCreate 핸들러다. 원문에 답장으로 제안과 버튼을 단다. 멘션 알림은 끈다.
func (n *Nudge) Handle(s *discordgo.Session, m *discordgo.MessageCreate) {
	trigger, ok := n.Decide(m.ChannelID, m.Author != nil && m.Author.Bot, m.Content)
	if !ok {
		return
	}
	cfg, _ := config.Lookup(m.ChannelID)
	content, comps := nudgeReply(trigger, cfg.Tone, n.Pick)
	_, err := s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Content:         content,
		Components:      comps,
		Reference:       m.Reference(),
		AllowedMentions: noMentions,
	})
	if err != nil {
		log.Printf("nudge 전송 실패 (channel=%s trigger=%s): %v", m.ChannelID, trigger, err)
	}
}

// applyMute 는 [오늘 그만] 버튼을 처리하는 판단부다. 받아들이면 해당 트리거를 KST 자정까지 멈추고
// 메시지에 보여줄 대사를 돌려준다. 모르는 동작·트리거, Nudge 가 꺼진 채널은 거절한다(상태 변화 없음).
func (n *Nudge) applyMute(channelID string, id interactions.ID) (string, bool) {
	if id.Action != nudgeActionMute {
		return "", false
	}
	if id.Arg != nudgeTriggerChicken && id.Arg != nudgeTriggerTime {
		return "", false
	}
	if !config.Enabled(channelID, config.Nudge) {
		return "", false
	}
	cfg, _ := config.Lookup(channelID)
	n.gate.MuteToday(channelID, id.Arg, n.Now())
	return voice.Say(voice.NudgeMuted, cfg.Tone, n.Pick), true
}

// HandleComponent 는 "nudge:*" 버튼 핸들러다. mute 만 처리하고 나머지는 로그만 남기고 무시한다.
func (n *Nudge) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate, id interactions.ID) {
	text, ok := n.applyMute(i.ChannelID, id)
	if !ok {
		log.Printf("nudge 버튼 무시 (channel=%s action=%q arg=%q)", i.ChannelID, id.Action, id.Arg)
		ackSilently(s, i)
		return
	}
	updateMessage(s, i, text, nil)
}
