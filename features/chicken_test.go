package features

import (
	"strings"
	"testing"

	"zzezze-bot/config"
	"zzezze-bot/yacht"
)

func TestRenderLobby(t *testing.T) {
	got := RenderLobby([]string{"1", "2"}, config.ToneFull, first)
	for _, want := range []string{"<@1>", "<@2>", "2명"} {
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
	got := RenderChicken(res, config.ToneFull, first)
	for _, want := range []string{"1라운드", "2라운드", "⚀ ⚁ ⚂ ⚃ ⚄", "30점", "<@2>"} {
		if !strings.Contains(got, want) {
			t.Errorf("result missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "{") {
		t.Errorf("치환 안 된 자리표시자:\n%s", got)
	}
}
