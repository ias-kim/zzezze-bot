package features

import (
	"strconv"
	"strings"

	"zzezze-bot/config"
	"zzezze-bot/voice"
	"zzezze-bot/yacht"
)

// mention 은 디스코드 유저 멘션 문법이다.
func mention(userID string) string { return "<@" + userID + ">" }

func mentions(users []string) string {
	ms := make([]string, len(users))
	for i, u := range users {
		ms[i] = mention(u)
	}
	return strings.Join(ms, " ")
}

// foodName 은 {food} 자리에 들어갈 말이다. 메뉴를 못 뽑았으면("뭐 먹을까") 일반 단어를 쓴다.
func foodName(food string, tone config.Tone, pick voice.Picker) string {
	if food == "" {
		return voice.Say(voice.FoodGeneric, tone, pick)
	}
	return food
}

// NudgeFoodText 는 음식 얘기에 반응해 약속 시간을 잡자고 하는 말이다.
func NudgeFoodText(food string, tone config.Tone, pick voice.Picker) string {
	return voice.Say(voice.NudgeFood, tone, pick, "{food}", foodName(food, tone, pick))
}

// PollTitle 은 시간 투표 질문이다. 음식 얘기에서 왔으면 음식 이름을 넣는다.
func PollTitle(food string, tone config.Tone, pick voice.Picker) string {
	if food == "" {
		return voice.Say(voice.TimePollTitle, tone, pick)
	}
	return voice.Say(voice.FoodPollTitle, tone, pick, "{food}", food)
}

// RenderLobby 는 참가 모집 중인 야추 판 화면이다.
func RenderLobby(users []string, tone config.Tone, pick voice.Picker) string {
	return voice.Say(voice.YachtOpen, tone, pick) + "\n" +
		voice.Say(voice.YachtPlayers, tone, pick, "{n}", strconv.Itoa(len(users)), "{users}", mentions(users))
}

// RenderResult 는 야추 판 결과 화면이다. 메시지를 한 번만 수정해서 보여준다.
func RenderResult(res yacht.QuickResult, tone config.Tone, pick voice.Picker) string {
	var b strings.Builder
	for i, round := range res.Rounds {
		if i > 0 {
			b.WriteString(voice.Say(voice.YachtTie, tone, pick) + "\n")
		}
		b.WriteString(voice.Say(voice.YachtRound, tone, pick, "{n}", strconv.Itoa(i+1)) + "\n")
		for _, r := range round {
			faces := make([]string, len(r.Dice))
			for j, d := range r.Dice {
				faces[j] = voice.Die(d)
			}
			b.WriteString(voice.Say(voice.YachtRow, tone, pick,
				"{user}", mention(r.User), "{dice}", strings.Join(faces, " "), "{sum}", strconv.Itoa(r.Sum)) + "\n")
		}
	}
	if res.ByLot {
		b.WriteString(voice.Say(voice.YachtByLot, tone, pick) + "\n")
	}
	b.WriteString(voice.Say(voice.YachtLoser, tone, pick, "{user}", mention(res.Loser)))
	return b.String()
}
