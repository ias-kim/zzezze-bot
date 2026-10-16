// Package quiet 는 봇이 시끄러워지지 않게 막는 소음 제어다.
// 채널+트리거별 쿨다운을 둔다. 시각은 인자로 받으므로 sleep 없이 테스트한다.
package quiet

import (
	"sync"
	"time"
)

type key struct{ channelID, trigger string }

// Gate 는 채널+트리거별로 마지막 반응 시각을 기억한다. 여러 고루틴에서 같이 써도 된다.
type Gate struct {
	cooldown time.Duration

	mu   sync.Mutex
	last map[key]time.Time
}

func New(cooldown time.Duration) *Gate {
	return &Gate{cooldown: cooldown, last: map[key]time.Time{}}
}

// Allow 는 쿨다운이 지났으면 true 를 돌려주고 now 를 마지막 반응 시각으로 기록한다.
// 막힌 시도는 기록하지 않는다(쿨다운이 늘어나지 않는다).
func (g *Gate) Allow(channelID, trigger string, now time.Time) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	k := key{channelID, trigger}
	if last, ok := g.last[k]; ok && now.Sub(last) < g.cooldown {
		return false
	}
	g.last[k] = now
	return true
}
