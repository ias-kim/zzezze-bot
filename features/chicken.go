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

// foodName 은 {food} 자리에 들어갈 말이다. 메뉴를 못 뽑았으면("뭐 시킬까") 일반 단어를 쓴다.
func foodName(food string, tone config.Tone, pick voice.Picker) string {
	if food == "" {
		return voice.Say(voice.FoodGeneric, tone, pick)
	}
	return food
}

// NudgeText 는 음식 얘기에 반응해 주사위 판을 제안하는 말이다.
func NudgeText(food string, tone config.Tone, pick voice.Picker) string {
	return voice.Say(voice.NudgeFood, tone, pick, "{food}", foodName(food, tone, pick))
}

// RenderLobby 는 참가 모집 중인 판 화면이다. food 는 걸린 음식(빈 문자열이면 일반 단어).
func RenderLobby(food string, users []string, tone config.Tone, pick voice.Picker) string {
	return voice.Say(voice.ChickenOpen, tone, pick, "{food}", foodName(food, tone, pick)) + "\n" +
		voice.Say(voice.ChickenPlayers, tone, pick, "{n}", strconv.Itoa(len(users)), "{users}", mentions(users))
}

// RenderChicken 은 판 결과 화면이다. 메시지를 한 번만 수정해서 보여준다.
func RenderChicken(food string, res yacht.ChickenResult, tone config.Tone, pick voice.Picker) string {
	var b strings.Builder
	for i, round := range res.Rounds {
		if i > 0 {
			b.WriteString(voice.Say(voice.ChickenTie, tone, pick) + "\n")
		}
		b.WriteString(voice.Say(voice.ChickenRound, tone, pick, "{n}", strconv.Itoa(i+1)) + "\n")
		for _, r := range round {
			faces := make([]string, len(r.Dice))
			for j, d := range r.Dice {
				faces[j] = voice.Die(d)
			}
			b.WriteString(voice.Say(voice.ChickenRow, tone, pick,
				"{user}", mention(r.User), "{dice}", strings.Join(faces, " "), "{sum}", strconv.Itoa(r.Sum)) + "\n")
		}
	}
	if res.ByLot {
		b.WriteString(voice.Say(voice.ChickenByLot, tone, pick) + "\n")
	}
	b.WriteString(voice.Say(voice.ChickenLoser, tone, pick, "{user}", mention(res.Loser), "{food}", foodName(food, tone, pick)))
	return b.String()
}
