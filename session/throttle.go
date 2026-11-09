package session

import (
	"sync"
	"time"
	_ "time/tzdata" // 서버에 tzdata 가 없어도 Asia/Seoul 을 쓴다
)

// KST 는 봇이 쓰는 기준 시간대다.
var KST = mustLoad("Asia/Seoul")

func mustLoad(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

// NextMidnight 는 loc 기준으로 now 다음에 오는 자정을 돌려준다.
func NextMidnight(now time.Time, loc *time.Location) time.Time {
	t := now.In(loc)
	return time.Date(t.Year(), t.Month(), t.Day()+1, 0, 0, 0, 0, loc)
}

// Throttle 은 트리거별·채널별 소음 제어다: 쿨다운과 [🔕 오늘 그만].
// 시각은 호출하는 쪽이 넘긴다(테스트에서 시계를 고정할 수 있게).
type Throttle struct {
	cooldown time.Duration

	mu    sync.Mutex
	last  map[string]time.Time // 마지막으로 반응한 시각
	muted map[string]time.Time // 이 시각까지 멈춤
}

func NewThrottle(cooldown time.Duration) *Throttle {
	return &Throttle{cooldown: cooldown, last: map[string]time.Time{}, muted: map[string]time.Time{}}
}

func key(trigger, channelID string) string { return trigger + "\x00" + channelID }

// Allow 는 지금 반응해도 되는지 보고, 된다면 반응한 것으로 기록한다.
func (t *Throttle) Allow(trigger, channelID string, now time.Time) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	k := key(trigger, channelID)
	if until, ok := t.muted[k]; ok && now.Before(until) {
		return false
	}
	if last, ok := t.last[k]; ok && now.Sub(last) < t.cooldown {
		return false
	}
	t.last[k] = now
	return true
}

// MuteToday 는 KST 자정까지 해당 채널의 트리거를 멈춘다.
func (t *Throttle) MuteToday(trigger, channelID string, now time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.muted[key(trigger, channelID)] = NextMidnight(now, KST)
}
