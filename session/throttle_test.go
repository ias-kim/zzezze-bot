package session

import (
	"testing"
	"time"
)

var kst = time.FixedZone("KST", 9*60*60)

func TestThrottleCooldown(t *testing.T) {
	th := NewThrottle(30 * time.Minute)
	t0 := time.Date(2026, 11, 9, 20, 0, 0, 0, kst)

	if !th.Allow("chicken", "c1", t0) {
		t.Fatal("첫 반응은 허용")
	}
	if th.Allow("chicken", "c1", t0.Add(29*time.Minute)) {
		t.Error("쿨다운 안에서는 막혀야 함")
	}
	if !th.Allow("chicken", "c2", t0.Add(time.Minute)) {
		t.Error("다른 채널은 따로 센다")
	}
	if !th.Allow("time", "c1", t0.Add(time.Minute)) {
		t.Error("다른 트리거는 따로 센다")
	}
	if !th.Allow("chicken", "c1", t0.Add(30*time.Minute)) {
		t.Error("쿨다운이 끝나면 허용")
	}
}

// [오늘 그만] 은 KST 자정까지 막는다.
func TestThrottleMuteUntilMidnight(t *testing.T) {
	th := NewThrottle(time.Minute)
	now := time.Date(2026, 11, 9, 23, 50, 0, 0, kst)
	th.MuteToday("chicken", "c1", now)

	if th.Allow("chicken", "c1", now.Add(9*time.Minute)) {
		t.Error("23:59 에는 막혀야 함")
	}
	if !th.Allow("time", "c1", now) {
		t.Error("다른 트리거는 막지 않는다")
	}
	if !th.Allow("chicken", "c1", time.Date(2026, 11, 10, 0, 0, 0, 0, kst)) {
		t.Error("자정이 지나면 풀려야 함")
	}
}

func TestNextMidnight(t *testing.T) {
	// UTC 로 들어와도 KST 기준 자정을 계산한다.
	now := time.Date(2026, 11, 9, 16, 30, 0, 0, time.UTC) // KST 11-10 01:30
	got := NextMidnight(now, kst)
	want := time.Date(2026, 11, 11, 0, 0, 0, 0, kst)
	if !got.Equal(want) {
		t.Errorf("NextMidnight = %v, want %v", got, want)
	}
}
