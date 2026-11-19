package features

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"zzezze-bot/config"
	"zzezze-bot/voice"
)

func timeVoteTestKST(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Seoul")
	if err != nil {
		t.Fatalf("Asia/Seoul 로드 실패: %v", err)
	}
	return loc
}

func TestTimeVoteBuildPoll(t *testing.T) {
	kst := timeVoteTestKST(t)
	hours := func(prefix string, from, to int) []string {
		var out []string
		for h := from; h <= to; h++ {
			out = append(out, prefix+" "+strconv.Itoa(h)+"시")
		}
		return out
	}

	cases := []struct {
		name string
		now  time.Time
		want []string
	}{
		{"17:59 는 오늘 6개", time.Date(2026, 11, 19, 17, 59, 0, 0, kst), hours("오늘", 18, 23)},
		{"22:30 은 오늘 23시 하나", time.Date(2026, 11, 19, 22, 30, 0, 0, kst), []string{"오늘 23시"}},
		{"23:30 은 내일 6개", time.Date(2026, 11, 19, 23, 30, 0, 0, kst), hours("내일", 18, 23)},
		{"UTC 08:30Z = KST 17:30 은 오늘", time.Date(2026, 11, 19, 8, 30, 0, 0, time.UTC), hours("오늘", 18, 23)},
		{"UTC 14:30Z = KST 23:30 은 내일", time.Date(2026, 11, 19, 14, 30, 0, 0, time.UTC), hours("내일", 18, 23)},
	}
	for _, tone := range []config.Tone{config.ToneFull, config.ToneMild} {
		for _, c := range cases {
			t.Run(string(tone)+"/"+c.name, func(t *testing.T) {
				p := timeVoteBuildPoll(c.now, tone, func(int) int { return 0 })
				if p == nil {
					t.Fatal("poll 이 nil")
				}
				if p.Duration != 3 {
					t.Errorf("Duration = %d, want 3 (시간 단위)", p.Duration)
				}
				if !p.AllowMultiselect {
					t.Error("복수 선택이어야 한다")
				}
				if strings.TrimSpace(p.Question.Text) == "" {
					t.Error("질문 텍스트가 비어 있다")
				}
				if len(p.Answers) == 0 || len(p.Answers) > 10 {
					t.Fatalf("선택지 개수 %d, 1~10 이어야 한다", len(p.Answers))
				}
				var got []string
				for _, a := range p.Answers {
					if a.Media == nil {
						t.Fatal("answer media 가 nil")
					}
					got = append(got, a.Media.Text)
				}
				if strings.Join(got, ",") != strings.Join(c.want, ",") {
					t.Errorf("선택지 = %v, want %v", got, c.want)
				}
			})
		}
	}
}

func TestTimeVoteBuildPollQuestionFromVoice(t *testing.T) {
	kst := timeVoteTestKST(t)
	now := time.Date(2026, 11, 19, 12, 0, 0, 0, kst)
	pick := func(int) int { return 0 }
	for _, tone := range []config.Tone{config.ToneFull, config.ToneMild} {
		p := timeVoteBuildPoll(now, tone, pick)
		if want := voice.Say(voice.TimePollTitle, tone, pick); p.Question.Text != want {
			t.Errorf("tone=%s 질문 = %q, want %q", tone, p.Question.Text, want)
		}
	}
}

func TestNewTimeVoteDefaults(t *testing.T) {
	tv := NewTimeVote()
	if tv.Now == nil || tv.Pick == nil {
		t.Fatal("Now, Pick 기본값이 채워져야 한다")
	}
}
