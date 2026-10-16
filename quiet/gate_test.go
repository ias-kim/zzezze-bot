package quiet

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

var t0 = time.Date(2026, 10, 16, 20, 0, 0, 0, time.UTC)

func TestAllowCooldown(t *testing.T) {
	g := New(30 * time.Minute)
	if !g.Allow("ch1", "meme", t0) {
		t.Fatal("첫 시도는 열려야 한다")
	}
	if g.Allow("ch1", "meme", t0.Add(30*time.Minute-time.Second)) {
		t.Error("29분 59초 뒤에는 막혀야 한다")
	}
	if !g.Allow("ch1", "meme", t0.Add(30*time.Minute)) {
		t.Error("30분 뒤에는 열려야 한다")
	}
}

func TestBlockedAttemptDoesNotExtend(t *testing.T) {
	g := New(30 * time.Minute)
	g.Allow("ch1", "meme", t0)
	g.Allow("ch1", "meme", t0.Add(20*time.Minute)) // 막힌 시도
	if !g.Allow("ch1", "meme", t0.Add(30*time.Minute)) {
		t.Error("막힌 시도가 쿨다운을 늘렸다")
	}
}

func TestKeysAreIndependent(t *testing.T) {
	g := New(30 * time.Minute)
	g.Allow("ch1", "meme", t0)
	if !g.Allow("ch2", "meme", t0) {
		t.Error("다른 채널이 막혔다")
	}
	if !g.Allow("ch1", "chicken", t0) {
		t.Error("다른 트리거가 막혔다")
	}
}

func TestAllowConcurrentOnlyOne(t *testing.T) {
	g := New(30 * time.Minute)
	var opened atomic.Int32
	var wg sync.WaitGroup
	for range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if g.Allow("ch1", "meme", t0) {
				opened.Add(1)
			}
		}()
	}
	wg.Wait()
	if opened.Load() != 1 {
		t.Errorf("동시에 열린 횟수 = %d, want 1", opened.Load())
	}
}
