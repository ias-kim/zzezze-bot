package features

import (
	"testing"
	"time"

	"zzezze-bot/config"
	"zzezze-bot/session"
)

func at(day, h, m int) time.Time { return time.Date(2026, 11, day, h, m, 0, 0, session.KST) }

func hours(ts []time.Time) []int {
	var out []int
	for _, t := range ts {
		out = append(out, t.Hour())
	}
	return out
}

func TestTimeSlots(t *testing.T) {
	tests := []struct {
		name    string
		now     time.Time
		wantDay int
		want    []int
	}{
		{"오후 3시: 오늘 전부", at(9, 15, 0), 9, []int{18, 19, 20, 21, 22, 23}},
		{"18시 정각은 지난 것으로 본다", at(9, 18, 0), 9, []int{19, 20, 21, 22, 23}},
		{"22:30: 23시만", at(9, 22, 30), 9, []int{23}},
		{"23:30: 내일 시간대", at(9, 23, 30), 10, []int{18, 19, 20, 21, 22, 23}},
		{"UTC 로 들어와도 KST 기준", time.Date(2026, 11, 9, 13, 30, 0, 0, time.UTC), 9, []int{23}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TimeSlots(tt.now)
			if h := hours(got); !equal(h, tt.want) {
				t.Errorf("hours = %v, want %v", h, tt.want)
			}
			if got[0].In(session.KST).Day() != tt.wantDay {
				t.Errorf("day = %d, want %d", got[0].Day(), tt.wantDay)
			}
		})
	}
}

func TestSlotLabel(t *testing.T) {
	now := at(9, 23, 30)
	if got := SlotLabel(at(10, 18, 0), now, config.ToneFull, first); got != "내일 18시" {
		t.Errorf("tomorrow label = %q", got)
	}
	if got := SlotLabel(at(9, 23, 0), at(9, 22, 30), config.ToneFull, first); got != "23시" {
		t.Errorf("today label = %q", got)
	}
}

func equal(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func first(int) int { return 0 }
