package session

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// fakeTimer 는 Stop 여부와 상관없이 테스트가 직접 발화시킬 수 있다.
// time.Timer.Stop 이 false 를 돌려주는 "이미 발화해서 콜백이 대기 중" 상황을 흉내 낸다.
type fakeTimer struct {
	d       time.Duration
	f       func()
	stopped atomic.Bool
}

func (t *fakeTimer) Stop() bool { return !t.stopped.Swap(true) }

func (t *fakeTimer) fire() { t.f() }

type fakeClock struct {
	mu     sync.Mutex
	timers []*fakeTimer
}

func (c *fakeClock) afterFunc(d time.Duration, f func()) stopper {
	c.mu.Lock()
	defer c.mu.Unlock()
	t := &fakeTimer{d: d, f: f}
	c.timers = append(c.timers, t)
	return t
}

func (c *fakeClock) last(t *testing.T) *fakeTimer {
	t.Helper()
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.timers) == 0 {
		t.Fatal("걸린 타이머가 없다")
	}
	return c.timers[len(c.timers)-1]
}

func newTestRegistry() (*Registry, *fakeClock) {
	c := &fakeClock{}
	r := New()
	r.afterFunc = c.afterFunc
	return r, c
}

func game(users ...string) Game {
	return Game{State: "state", Idle: time.Minute, Users: users}
}

func TestStartRegistersUsers(t *testing.T) {
	r, _ := newTestRegistry()
	if err := r.Start("m1", game("a", "b")); err != nil {
		t.Fatal(err)
	}
	for _, u := range []string{"a", "b"} {
		if got, ok := r.UserGame(u); !ok || got != "m1" {
			t.Errorf("UserGame(%q) = %q, %v; want m1, true", u, got, ok)
		}
	}
}

func TestStartRejects(t *testing.T) {
	tests := []struct {
		name  string
		msgID string
		g     Game
		want  error
	}{
		{"이미 있는 msgID", "m1", game("x"), ErrExists},
		{"다른 판에 있는 유저", "m2", game("x", "a"), ErrUserBusy},
		{"목록 안 중복 유저", "m2", game("x", "x"), ErrDuplicateUser},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, _ := newTestRegistry()
			if err := r.Start("m1", game("a")); err != nil {
				t.Fatal(err)
			}
			if err := r.Start(tt.msgID, tt.g); !errors.Is(err, tt.want) {
				t.Errorf("Start error = %v, want %v", err, tt.want)
			}
			// 실패한 Start 는 아무 유저도 남기지 않는다(원자성).
			if got, ok := r.UserGame("x"); ok {
				t.Errorf("실패한 Start 뒤 x 가 %q 에 남아 있다", got)
			}
		})
	}
}

func TestJoin(t *testing.T) {
	r, _ := newTestRegistry()
	if err := r.Start("m1", game("a")); err != nil {
		t.Fatal(err)
	}
	if err := r.Start("m2", game("b")); err != nil {
		t.Fatal(err)
	}
	if err := r.Join("m1", "c"); err != nil {
		t.Errorf("Join 새 유저: %v", err)
	}
	if got, _ := r.UserGame("c"); got != "m1" {
		t.Errorf("UserGame(c) = %q, want m1", got)
	}
	tests := []struct {
		name, msgID, user string
		want              error
	}{
		{"같은 판 재참가", "m1", "a", ErrAlreadyJoined},
		{"다른 판 유저", "m1", "b", ErrUserBusy},
		{"없는 판", "nope", "d", ErrNotFound},
	}
	for _, tt := range tests {
		if err := r.Join(tt.msgID, tt.user); !errors.Is(err, tt.want) {
			t.Errorf("%s: Join error = %v, want %v", tt.name, err, tt.want)
		}
	}
}

func TestFinishOnceAndCleansIndex(t *testing.T) {
	r, c := newTestRegistry()
	if err := r.Start("m1", game("a", "b")); err != nil {
		t.Fatal(err)
	}
	if !r.Finish("m1") {
		t.Fatal("첫 Finish 는 true 여야 한다")
	}
	if r.Finish("m1") {
		t.Error("두 번째 Finish 는 false 여야 한다")
	}
	if _, ok := r.UserGame("a"); ok {
		t.Error("Finish 뒤에도 a 가 역인덱스에 남아 있다")
	}
	if !c.last(t).stopped.Load() {
		t.Error("Finish 가 타이머를 멈추지 않았다")
	}
	// 끝난 판의 유저는 새 판을 시작할 수 있다.
	if err := r.Start("m2", game("a")); err != nil {
		t.Errorf("Finish 뒤 새 판 Start: %v", err)
	}
}

func TestFinishOnlyRemovesOwnUsers(t *testing.T) {
	r, _ := newTestRegistry()
	_ = r.Start("m1", game("a"))
	_ = r.Start("m2", game("b"))
	r.Finish("m1")
	if got, ok := r.UserGame("b"); !ok || got != "m2" {
		t.Errorf("다른 판 유저 b 가 지워졌다: %q, %v", got, ok)
	}
}

func TestFinishConcurrentExactlyOnce(t *testing.T) {
	r, _ := newTestRegistry()
	_ = r.Start("m1", game("a"))
	var wins atomic.Int32
	var wg sync.WaitGroup
	for range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if r.Finish("m1") {
				wins.Add(1)
			}
		}()
	}
	wg.Wait()
	if wins.Load() != 1 {
		t.Errorf("Finish true 횟수 = %d, want 1", wins.Load())
	}
}

func TestTimerExpiryCallsOnExpire(t *testing.T) {
	r, c := newTestRegistry()
	var gotID string
	var gotState any
	g := game("a")
	g.State = "판"
	g.OnExpire = func(msgID string, state any) { gotID, gotState = msgID, state }
	_ = r.Start("m1", g)
	if d := c.last(t).d; d != time.Minute {
		t.Errorf("타이머 시간 = %v, want %v", d, time.Minute)
	}
	c.last(t).fire()
	if gotID != "m1" || gotState != "판" {
		t.Errorf("onExpire(%q, %v), want (m1, 판)", gotID, gotState)
	}
	if r.Finish("m1") {
		t.Error("만료된 판의 Finish 는 false 여야 한다")
	}
	if _, ok := r.UserGame("a"); ok {
		t.Error("만료 뒤에도 a 가 역인덱스에 남아 있다")
	}
}

func TestExpiryVsFinishExactlyOneWinner(t *testing.T) {
	for range 100 {
		r, c := newTestRegistry()
		var expired atomic.Int32
		g := game("a")
		g.OnExpire = func(string, any) { expired.Add(1) }
		_ = r.Start("m1", g)
		tm := c.last(t)

		var finished atomic.Bool
		var wg sync.WaitGroup
		wg.Add(2)
		go func() { defer wg.Done(); tm.fire() }()
		go func() { defer wg.Done(); finished.Store(r.Finish("m1")) }()
		wg.Wait()

		n := int(expired.Load())
		if finished.Load() {
			n++
		}
		if n != 1 {
			t.Fatalf("만료 %d회 + Finish %v: 승자는 정확히 하나여야 한다", expired.Load(), finished.Load())
		}
	}
}

func TestTouchResetsTimerAndIgnoresStaleFire(t *testing.T) {
	r, c := newTestRegistry()
	var expired atomic.Int32
	g := game("a")
	g.OnExpire = func(string, any) { expired.Add(1) }
	_ = r.Start("m1", g)
	old := c.last(t)

	if !r.Touch("m1") {
		t.Fatal("있는 판의 Touch 는 true 여야 한다")
	}
	if !old.stopped.Load() {
		t.Error("Touch 가 이전 타이머를 멈추지 않았다")
	}
	fresh := c.last(t)
	if fresh == old || fresh.d != time.Minute {
		t.Fatalf("Touch 가 새 타이머(%v)를 걸지 않았다", time.Minute)
	}

	old.fire() // 늦게 도착한 이전 타이머
	if expired.Load() != 0 {
		t.Fatal("Touch 이전 타이머가 판을 끝냈다")
	}
	if _, ok := r.UserGame("a"); !ok {
		t.Fatal("늦은 발화 뒤 판이 사라졌다")
	}

	fresh.fire()
	if expired.Load() != 1 {
		t.Errorf("새 타이머 발화 뒤 만료 횟수 = %d, want 1", expired.Load())
	}
	if r.Touch("m1") {
		t.Error("끝난 판의 Touch 는 false 여야 한다")
	}
}

func TestOnExpireRunsOutsideLocks(t *testing.T) {
	r, c := newTestRegistry()
	done := make(chan struct{})
	g := game("a")
	g.OnExpire = func(msgID string, _ any) {
		// 콜백 안에서 Registry 를 다시 불러도 데드락이 없어야 한다.
		_ = r.Start("m2", game("a"))
		r.Finish("m2")
		close(done)
	}
	_ = r.Start("m1", g)
	go c.last(t).fire()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("onExpire 안에서 Registry 호출이 데드락에 걸렸다")
	}
}

func TestDo(t *testing.T) {
	r, _ := newTestRegistry()
	g := game("a")
	g.State = &[]int{}
	_ = r.Start("m1", g)

	var wg sync.WaitGroup
	for range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = r.Do("m1", func(s any) {
				p := s.(*[]int)
				*p = append(*p, 1)
			})
		}()
	}
	wg.Wait()

	var n int
	if err := r.Do("m1", func(s any) { n = len(*s.(*[]int)) }); err != nil {
		t.Fatal(err)
	}
	if n != 50 {
		t.Errorf("Do 50회 뒤 길이 = %d, want 50 (판별 락이 직렬화해야 한다)", n)
	}
	if err := r.Do("nope", func(any) {}); !errors.Is(err, ErrNotFound) {
		t.Errorf("없는 판 Do error = %v, want ErrNotFound", err)
	}
}

func TestDoCanCallTouchAndFinish(t *testing.T) {
	r, _ := newTestRegistry()
	_ = r.Start("m1", game("a"))
	done := make(chan bool)
	go func() {
		var finished bool
		_ = r.Do("m1", func(any) {
			r.Touch("m1")
			finished = r.Finish("m1")
		})
		done <- finished
	}()
	select {
	case ok := <-done:
		if !ok {
			t.Error("Do 안의 Finish 가 true 를 돌려주지 않았다")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Do 안에서 Touch/Finish 호출이 데드락에 걸렸다")
	}
}

func TestCloseAll(t *testing.T) {
	r, c := newTestRegistry()
	var expired atomic.Int32
	g1 := game("a")
	g1.State = "s1"
	g1.OnExpire = func(string, any) { expired.Add(1) }
	g2 := game("b")
	g2.State = "s2"
	_ = r.Start("m1", g1)
	t1 := c.last(t)
	_ = r.Start("m2", g2)
	_ = r.Start("m3", game("c"))
	r.Finish("m3")

	got := map[string]any{}
	for _, cl := range r.CloseAll() {
		got[cl.MsgID] = cl.State
	}
	if len(got) != 2 || got["m1"] != "s1" || got["m2"] != "s2" {
		t.Errorf("CloseAll = %v, want m1:s1, m2:s2", got)
	}
	if !t1.stopped.Load() {
		t.Error("CloseAll 이 타이머를 멈추지 않았다")
	}

	t1.fire() // 종료 뒤 늦게 도착한 타이머
	if expired.Load() != 0 {
		t.Error("CloseAll 뒤 발화한 타이머가 onExpire 를 불렀다")
	}
	if err := r.Start("m4", game("d")); !errors.Is(err, ErrClosed) {
		t.Errorf("CloseAll 뒤 Start error = %v, want ErrClosed", err)
	}
	if len(r.CloseAll()) != 0 {
		t.Error("두 번째 CloseAll 은 빈 목록이어야 한다")
	}
}
