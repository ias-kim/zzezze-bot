// Package quiet 는 봇이 시끄러워지지 않게 막는 소음 제어다.
// 채널+트리거별 쿨다운을 둔다. 시각은 인자로 받으므로 sleep 없이 테스트한다.
package quiet

import (
	"sync"
	"time"
	_ "time/tzdata" // 서버에 tzdata 가 없어도 Asia/Seoul 을 읽을 수 있게 내장한다.
)

// seoul 은 "오늘 그만"의 자정 기준 시간대다.
var seoul = mustLoadKST()

func mustLoadKST() *time.Location {
	loc, err := time.LoadLocation("Asia/Seoul")
	if err != nil {
		panic(err) // tzdata 를 내장했으므로 일어나지 않는다.
	}
	return loc
}

type key struct{ channelID, trigger string }

// Gate 는 채널+트리거별로 마지막 반응 시각을 기억한다. 여러 고루틴에서 같이 써도 된다.
type Gate struct {
	cooldown time.Duration

	mu         sync.Mutex
	last       map[key]time.Time
	mutedUntil map[key]time.Time // 이 시각 전까지 음소거
}

func New(cooldown time.Duration) *Gate {
	return &Gate{cooldown: cooldown, last: map[key]time.Time{}, mutedUntil: map[key]time.Time{}}
}

// Allow 는 쿨다운이 지났으면 true 를 돌려주고 now 를 마지막 반응 시각으로 기록한다.
// 막힌 시도는 기록하지 않는다(쿨다운이 늘어나지 않는다).
func (g *Gate) Allow(channelID, trigger string, now time.Time) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	k := key{channelID, trigger}
	if until, ok := g.mutedUntil[k]; ok {
		if now.Before(until) {
			return false // 음소거 중 막힌 시도는 last 를 바꾸지 않는다.
		}
		delete(g.mutedUntil, k)
	}
	if last, ok := g.last[k]; ok && now.Sub(last) < g.cooldown {
		return false
	}
	g.last[k] = now
	return true
}

// MuteToday 는 now 를 KST 로 바꾼 날짜의 다음날 00:00 KST 전까지
// 그 채널+트리거의 Allow 를 막는다. 쿨다운 기록(last)은 건드리지 않는다.
func (g *Gate) MuteToday(channelID, trigger string, now time.Time) {
	y, m, d := now.In(seoul).Date()
	until := time.Date(y, m, d+1, 0, 0, 0, 0, seoul)
	g.mu.Lock()
	defer g.mu.Unlock()
	g.mutedUntil[key{channelID, trigger}] = until
}
