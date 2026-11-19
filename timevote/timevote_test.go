package timevote

import (
	"testing"
	"time"
)

var testKST = time.FixedZone("KST", 9*60*60)

// hoursOf 는 선택지를 KST 기준 "월-일 시" 문자열 대신 (일, 시) 쌍으로 단순화해 비교한다.
func hoursOf(opts []time.Time) [][2]int {
	out := make([][2]int, len(opts))
	for i, o := range opts {
		k := o.In(testKST)
		out[i] = [2]int{k.Day(), k.Hour()}
	}
	return out
}

func TestOptions(t *testing.T) {
	cases := []struct {
		name string
		now  time.Time
		want [][2]int // {KST 일, KST 시}
	}{
		{"17:59 는 오늘 18~23시", time.Date(2026, 11, 19, 17, 59, 0, 0, testKST), [][2]int{{19, 18}, {19, 19}, {19, 20}, {19, 21}, {19, 22}, {19, 23}}},
		{"18:00 정각은 지난 것이라 19~23시", time.Date(2026, 11, 19, 18, 0, 0, 0, testKST), [][2]int{{19, 19}, {19, 20}, {19, 21}, {19, 22}, {19, 23}}},
		{"22:00 정각은 23시 하나", time.Date(2026, 11, 19, 22, 0, 0, 0, testKST), [][2]int{{19, 23}}},
		{"22:30 은 23시 하나", time.Date(2026, 11, 19, 22, 30, 0, 0, testKST), [][2]int{{19, 23}}},
		{"23:00 정각은 내일 18~23시", time.Date(2026, 11, 19, 23, 0, 0, 0, testKST), [][2]int{{20, 18}, {20, 19}, {20, 20}, {20, 21}, {20, 22}, {20, 23}}},
		{"23:30 은 내일 18~23시", time.Date(2026, 11, 19, 23, 30, 0, 0, testKST), [][2]int{{20, 18}, {20, 19}, {20, 20}, {20, 21}, {20, 22}, {20, 23}}},
		{"월말 1/31 23:30 은 2/1 18~23시", time.Date(2026, 1, 31, 23, 30, 0, 0, testKST), [][2]int{{1, 18}, {1, 19}, {1, 20}, {1, 21}, {1, 22}, {1, 23}}},
		{"00:10 은 오늘 18~23시", time.Date(2026, 11, 20, 0, 10, 0, 0, testKST), [][2]int{{20, 18}, {20, 19}, {20, 20}, {20, 21}, {20, 22}, {20, 23}}},
		{"UTC 08:30 = KST 17:30 은 오늘 18~23시", time.Date(2026, 11, 19, 8, 30, 0, 0, time.UTC), [][2]int{{19, 18}, {19, 19}, {19, 20}, {19, 21}, {19, 22}, {19, 23}}},
		{"UTC 14:30 = KST 23:30 은 내일 18~23시", time.Date(2026, 11, 19, 14, 30, 0, 0, time.UTC), [][2]int{{20, 18}, {20, 19}, {20, 20}, {20, 21}, {20, 22}, {20, 23}}},
		{"UTC 15:10 = KST 다음날 00:10 은 그날 18~23시", time.Date(2026, 11, 19, 15, 10, 0, 0, time.UTC), [][2]int{{20, 18}, {20, 19}, {20, 20}, {20, 21}, {20, 22}, {20, 23}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := hoursOf(Options(c.now))
			if len(got) != len(c.want) {
				t.Fatalf("got %v, want %v", got, c.want)
			}
			for i := range got {
				if got[i] != c.want[i] {
					t.Fatalf("got %v, want %v", got, c.want)
				}
			}
		})
	}
}

func TestOptionsAreExactHoursInKST(t *testing.T) {
	for _, o := range Options(time.Date(2026, 11, 19, 17, 59, 0, 0, testKST)) {
		if o.Minute() != 0 || o.Second() != 0 || o.Nanosecond() != 0 {
			t.Errorf("정각이 아니다: %v", o)
		}
		if _, off := o.Zone(); off != 9*60*60 {
			t.Errorf("KST 가 아니다: %v", o)
		}
	}
}

func TestOptionsAscendingAndAfterNow(t *testing.T) {
	now := time.Date(2026, 11, 19, 19, 20, 0, 0, testKST)
	opts := Options(now)
	for i, o := range opts {
		if !o.After(now) {
			t.Errorf("now 이후여야 한다: %v", o)
		}
		if i > 0 && !o.After(opts[i-1]) {
			t.Errorf("오름차순이어야 한다: %v", opts)
		}
	}
}

func TestDuration(t *testing.T) {
	if Duration != 3*time.Hour {
		t.Errorf("Duration = %v, want 3h", Duration)
	}
}
