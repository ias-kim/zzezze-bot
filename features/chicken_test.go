package features

import (
	"strings"
	"testing"

	"zzezze-bot/config"
	"zzezze-bot/yacht"
)

func TestRenderLobby(t *testing.T) {
	got := RenderLobby("곱창", []string{"1", "2"}, config.ToneFull, first)
	for _, want := range []string{"<@1>", "<@2>", "2명", "곱창"} {
		if !strings.Contains(got, want) {
			t.Errorf("lobby %q missing %q", got, want)
		}
	}
}

func TestRenderChicken(t *testing.T) {
	res := yacht.ChickenResult{
		Rounds: [][]yacht.Roll{
			{{User: "1", Dice: [5]int{1, 1, 1, 1, 1}, Sum: 5}, {User: "2", Dice: [5]int{1, 1, 1, 1, 1}, Sum: 5}},
			{{User: "1", Dice: [5]int{6, 6, 6, 6, 6}, Sum: 30}, {User: "2", Dice: [5]int{1, 2, 3, 4, 5}, Sum: 15}},
		},
		Loser: "2",
	}
	got := RenderChicken("양꼬치", res, config.ToneFull, first)
	for _, want := range []string{"1라운드", "2라운드", "⚀ ⚁ ⚂ ⚃ ⚄", "30점", "<@2>", "양꼬치"} {
		if !strings.Contains(got, want) {
			t.Errorf("result missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "{") {
		t.Errorf("치환 안 된 자리표시자:\n%s", got)
	}
}

// 음식 이름이 없으면("뭐 시킬까") 일반 단어로 채운다. 자리표시자가 남으면 안 된다.
func TestRenderWithoutFood(t *testing.T) {
	for _, tone := range []config.Tone{config.ToneFull, config.ToneMild} {
		for _, got := range []string{
			RenderLobby("", []string{"1"}, tone, first),
			RenderChicken("", yacht.ChickenResult{Loser: "1"}, tone, first),
			NudgeText("", tone, first),
		} {
			if strings.Contains(got, "{") {
				t.Errorf("치환 안 된 자리표시자: %q", got)
			}
		}
	}
}
