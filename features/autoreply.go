// Package features 는 디스코드 이벤트를 순수 로직에 연결하고 화면을 그린다.
// 판단은 디스코드 없이 테스트할 수 있는 함수(Decide 등)로 두고, 핸들러는 얇게 유지한다.
package features

import (
	"log"
	"math/rand/v2"
	"regexp"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"zzezze-bot/config"
	"zzezze-bot/quiet"
	"zzezze-bot/voice"
)

// MemeCooldown 은 같은 채널에서 밈 반응 사이의 최소 간격이다.
const MemeCooldown = 30 * time.Minute

const memeTrigger = "meme"

var (
	// "주 ㅔ" 처럼 자모를 쪼갠 표기를 한 글자로 합친다.
	jamoSplit = strings.NewReplacer("주ㅔ", "줴", "쭈ㅔ", "쮀", "주 ㅔ", "줴", "쭈 ㅔ", "쮀")

	// 쮀쮀이야 변형: 두 글자(띄어쓰기 없이) + 이야/희야. "이제 재희야" 같은 평범한 말은 띄어쓰기로 걸러진다.
	// "쮀쮀", "줴줴"는 일상어에 안 나오므로 단독으로도 잡는다. "줘줘", "졸졸"은 오작동이 많아 뺀다.
	memePattern = regexp.MustCompile(`[쮀줴제체재쩨졔][쮀줴제체재쩨졔]\s*[이희]\s*야|쮀\s*쮀|줴\s*줴`)
)

// MemeHit 은 메시지가 쮀쮀이야 계열 밈인지 판단한다.
func MemeHit(content string) bool {
	return memePattern.MatchString(jamoSplit.Replace(content))
}

// AutoReply 는 밈 자동 반응이다. Pick 과 Now 는 테스트에서 바꾼다.
type AutoReply struct {
	gate *quiet.Gate
	Pick voice.Picker
	Now  func() time.Time
}

func NewAutoReply(g *quiet.Gate) *AutoReply {
	return &AutoReply{gate: g, Pick: rand.IntN, Now: time.Now}
}

// Decide 는 반응할지와 보낼 대사를 정한다. 트리거에 걸렸을 때만 쿨다운을 소모한다.
func (a *AutoReply) Decide(channelID string, fromBot bool, content string) (string, bool) {
	if fromBot || strings.HasPrefix(content, "/") || strings.HasPrefix(content, "!") {
		return "", false
	}
	if !config.Enabled(channelID, config.AutoReply) {
		return "", false
	}
	if !MemeHit(content) {
		return "", false
	}
	if !a.gate.Allow(channelID, memeTrigger, a.Now()) {
		return "", false
	}
	cfg, _ := config.Lookup(channelID)
	return voice.Say(voice.MemeReply, cfg.Tone, a.Pick), true
}

// Handle 은 MessageCreate 핸들러다. 원문에 답장으로 보내고 멘션 알림은 끈다.
func (a *AutoReply) Handle(s *discordgo.Session, m *discordgo.MessageCreate) {
	text, ok := a.Decide(m.ChannelID, m.Author.Bot, m.Content)
	if !ok {
		return
	}
	_, err := s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Content:         text,
		Reference:       m.Reference(),
		AllowedMentions: &discordgo.MessageAllowedMentions{},
	})
	if err != nil {
		log.Printf("autoreply 전송 실패 (channel=%s): %v", m.ChannelID, err)
	}
}
