package features

import (
	"log"
	"math/rand/v2"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"

	"zzezze-bot/config"
	"zzezze-bot/interactions"
	"zzezze-bot/timevote"
	"zzezze-bot/voice"
)

// TimeVote 는 [시간 투표] 버튼을 받아 디스코드 기본 투표를 올린다.
// Now 와 Pick 은 테스트에서 바꾼다.
type TimeVote struct {
	Now  func() time.Time
	Pick voice.Picker
}

func NewTimeVote() *TimeVote {
	return &TimeVote{Now: time.Now, Pick: rand.IntN}
}

// timeVoteBuildPoll 은 now 이후의 시간 선택지로 복수 선택 투표를 만든다.
// 오늘/내일은 KST 날짜 기준이며, timevote.Options 가 KST 시각을 돌려주므로 그 시간대를 그대로 쓴다.
func timeVoteBuildPoll(now time.Time, tone config.Tone, pick voice.Picker) *discordgo.Poll {
	opts := timevote.Options(now)
	answers := make([]discordgo.PollAnswer, 0, len(opts))
	for _, o := range opts {
		key := voice.TimeOptionTomorrow
		if sameDay(o, now.In(o.Location())) {
			key = voice.TimeOptionToday
		}
		answers = append(answers, discordgo.PollAnswer{
			Media: &discordgo.PollMedia{
				Text: voice.Say(key, tone, pick, "{hour}", strconv.Itoa(o.Hour())),
			},
		})
	}
	return &discordgo.Poll{
		Question:         discordgo.PollMedia{Text: voice.Say(voice.TimePollTitle, tone, pick)},
		Answers:          answers,
		AllowMultiselect: true,
		Duration:         int(timevote.Duration / time.Hour), // 디스코드 Poll.Duration 은 시간 단위
	}
}

// sameDay 는 두 시각이 같은 시간대(a 기준)에서 같은 날짜인지 본다.
func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

// HandleComponent 는 `timevote` 버튼을 처리한다. 지금은 "open" 만 있다.
func (tv *TimeVote) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate, id interactions.ID) {
	if id.Action != "open" {
		return
	}
	tone := config.ToneMild
	if cfg, ok := config.Lookup(i.ChannelID); ok {
		tone = cfg.Tone
	}

	// 3초 안에 응답해야 하므로 먼저 제안 메시지의 버튼을 없앤다(중복 클릭 방지).
	content := ""
	if i.Message != nil {
		content = i.Message.Content
	}
	updateMessage(s, i, content, nil)

	_, err := s.ChannelMessageSendComplex(i.ChannelID, &discordgo.MessageSend{
		Poll:            timeVoteBuildPoll(tv.Now(), tone, tv.Pick),
		AllowedMentions: noMentions,
	})
	if err == nil {
		return
	}
	log.Printf("시간 투표 게시 실패 (channel=%s): %v", i.ChannelID, err)
	_, err = s.ChannelMessageSendComplex(i.ChannelID, &discordgo.MessageSend{
		Content:         voice.Say(voice.TimePollFailed, tone, tv.Pick),
		AllowedMentions: noMentions,
	})
	if err != nil {
		log.Printf("시간 투표 실패 안내 전송 실패 (channel=%s): %v", i.ChannelID, err)
	}
}
