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

// kst 는 테스트용 KST 시각을 만든다.
func kst(t *testing.T, y int, m time.Month, d, h, mi, sec int) time.Time {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Seoul")
	if err != nil {
		t.Fatal(err)
	}
	return time.Date(y, m, d, h, mi, sec, 0, loc)
}

func TestMuteTodayBlocksUntilKSTMidnight(t *testing.T) {
	g := New(30 * time.Minute)
	g.MuteToday("ch1", "meme", kst(t, 2026, 10, 16, 20, 0, 0))
	if g.Allow("ch1", "meme", kst(t, 2026, 10, 16, 23, 59, 59)) {
		t.Error("23:59:59 KST 에는 막혀야 한다")
	}
	if !g.Allow("ch1", "meme", kst(t, 2026, 10, 17, 0, 0, 0)) {
		t.Error("다음날 00:00 KST 에는 열려야 한다")
	}
}

func TestMuteTodayDoesNotAffectOthers(t *testing.T) {
	g := New(30 * time.Minute)
	now := kst(t, 2026, 10, 16, 20, 0, 0)
	g.MuteToday("ch1", "meme", now)
	if !g.Allow("ch2", "meme", now) {
		t.Error("다른 채널이 막혔다")
	}
	if !g.Allow("ch1", "chicken", now) {
		t.Error("다른 트리거가 막혔다")
	}
}

func TestMuteTodayUsesKSTWhenNowIsUTC(t *testing.T) {
	g := New(30 * time.Minute)
	// UTC 14:59 = KST 23:59 (같은 날), UTC 15:00 = KST 다음날 00:00
	g.MuteToday("ch1", "meme", time.Date(2026, 10, 16, 14, 59, 0, 0, time.UTC))
	if g.Allow("ch1", "meme", time.Date(2026, 10, 16, 14, 59, 59, 0, time.UTC)) {
		t.Error("UTC 14:59:59(KST 23:59:59)에는 막혀야 한다")
	}
	if !g.Allow("ch1", "meme", time.Date(2026, 10, 16, 15, 0, 0, 0, time.UTC)) {
		t.Error("UTC 15:00(KST 자정)에는 열려야 한다")
	}
}

func TestMuteTodayFromUTCAfterKSTDateChange(t *testing.T) {
	g := New(30 * time.Minute)
	// UTC 15:30 은 이미 KST 다음날 00:30 이므로 자정은 그 다음날 00:00 KST 다.
	g.MuteToday("ch1", "meme", time.Date(2026, 10, 16, 15, 30, 0, 0, time.UTC))
	if g.Allow("ch1", "meme", kst(t, 2026, 10, 17, 23, 59, 59)) {
		t.Error("KST 17일 23:59:59 에는 막혀야 한다")
	}
	if !g.Allow("ch1", "meme", kst(t, 2026, 10, 18, 0, 0, 0)) {
		t.Error("KST 18일 00:00 에는 열려야 한다")
	}
}

func TestMuteBlockedAllowKeepsCooldownRecord(t *testing.T) {
	g := New(30 * time.Minute)
	first := kst(t, 2026, 10, 16, 23, 0, 0)
	g.Allow("ch1", "meme", first)
	g.MuteToday("ch1", "meme", first.Add(10*time.Minute))
	// 음소거 중 막힌 시도는 last 를 갱신하면 안 된다.
	if g.Allow("ch1", "meme", kst(t, 2026, 10, 16, 23, 59, 0)) {
		t.Fatal("음소거 중에는 막혀야 한다")
	}
	// 해제 직후: last 가 23:00 그대로라면 쿨다운(30분)이 지났으니 열린다.
	if !g.Allow("ch1", "meme", kst(t, 2026, 10, 17, 0, 0, 0)) {
		t.Fatal("자정에는 열려야 한다")
	}
	// 방금 열린 시도가 last 로 기록됐으므로 바로 다음은 쿨다운으로 막힌다.
	if g.Allow("ch1", "meme", kst(t, 2026, 10, 17, 0, 10, 0)) {
		t.Error("열린 뒤 10분 시점에는 쿨다운으로 막혀야 한다")
	}
}

func TestMuteKeepsPreMuteCooldownState(t *testing.T) {
	g := New(30 * time.Minute)
	// 자정 직전에 반응했고, 곧바로 음소거. 음소거 중 막힌 시도가 last 를 밀면 안 된다.
	g.Allow("ch1", "meme", kst(t, 2026, 10, 16, 23, 50, 0))
	g.MuteToday("ch1", "meme", kst(t, 2026, 10, 16, 23, 51, 0))
	g.Allow("ch1", "meme", kst(t, 2026, 10, 16, 23, 59, 0)) // 막힘
	// 00:10 KST: last 가 23:50 이면 20분 경과 → 쿨다운으로 여전히 막혀야 한다.
	if g.Allow("ch1", "meme", kst(t, 2026, 10, 17, 0, 10, 0)) {
		t.Error("음소거 전 쿨다운 기록(23:50)이 유지되어 00:10 에는 막혀야 한다")
	}
	if !g.Allow("ch1", "meme", kst(t, 2026, 10, 17, 0, 20, 0)) {
		t.Error("23:50 + 30분 = 00:20 에는 열려야 한다")
	}
}

func TestMuteTodayConcurrent(t *testing.T) {
	g := New(30 * time.Minute)
	now := kst(t, 2026, 10, 16, 20, 0, 0)
	var wg sync.WaitGroup
	for i := range 50 {
		wg.Add(2)
		go func() {
			defer wg.Done()
			g.MuteToday("ch1", "meme", now)
		}()
		go func() {
			defer wg.Done()
			g.Allow("ch1", "meme", now.Add(time.Duration(i)*time.Second))
		}()
	}
	wg.Wait()
	if g.Allow("ch1", "meme", kst(t, 2026, 10, 16, 23, 59, 59)) {
		t.Error("동시 호출 뒤에도 음소거는 유지되어야 한다")
	}
}
