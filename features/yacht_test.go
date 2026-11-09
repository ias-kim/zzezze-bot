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

func TestRenderResult(t *testing.T) {
	res := yacht.QuickResult{
		Rounds: [][]yacht.Roll{
			{{User: "1", Dice: [5]int{1, 1, 1, 1, 1}, Sum: 5}, {User: "2", Dice: [5]int{1, 1, 1, 1, 1}, Sum: 5}},
			{{User: "1", Dice: [5]int{6, 6, 6, 6, 6}, Sum: 30}, {User: "2", Dice: [5]int{1, 2, 3, 4, 5}, Sum: 15}},
		},
		Loser: "2",
	}
	got := RenderResult(res, config.ToneFull, first)
	for _, want := range []string{"1라운드", "2라운드", "⚀ ⚁ ⚂ ⚃ ⚄", "30점", "<@2>"} {
		if !strings.Contains(got, want) {
			t.Errorf("result missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "{") {
		t.Errorf("치환 안 된 자리표시자:\n%s", got)
	}
}

// 음식 얘기 → 약속 잡기 제안과 투표 제목에 음식 이름이 들어간다. 없으면 일반 문구.
func TestFoodTexts(t *testing.T) {
	for _, tone := range []config.Tone{config.ToneFull, config.ToneMild} {
		if got := NudgeFoodText("곱창", tone, first); !strings.Contains(got, "곱창") {
			t.Errorf("nudge 에 음식 이름이 없음: %q", got)
		}
		if got := PollTitle("곱창", tone, first); !strings.Contains(got, "곱창") {
			t.Errorf("투표 제목에 음식 이름이 없음: %q", got)
		}
		for _, got := range []string{NudgeFoodText("", tone, first), PollTitle("", tone, first), RenderLobby([]string{"1"}, tone, first)} {
			if got == "" || strings.Contains(got, "{") {
				t.Errorf("빈 문구이거나 치환 안 된 자리표시자: %q", got)
			}
		}
	}
}
