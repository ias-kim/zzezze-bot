package session

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// fakeClock 은 테스트에서 타이머를 손으로 터뜨리기 위한 가짜 시계다.
type fakeClock struct {
	mu     sync.Mutex
	timers []*fakeTimer
}

type fakeTimer struct {
	d       time.Duration
	f       func()
	stopped bool
}

func (c *fakeClock) afterFunc(d time.Duration, f func()) timer {
	c.mu.Lock()
	defer c.mu.Unlock()
	t := &fakeTimer{d: d, f: f}
	c.timers = append(c.timers, t)
	return t
}

func (t *fakeTimer) Stop() bool {
	was := !t.stopped
	t.stopped = true
	return was
}

// last 는 가장 최근에 만든 타이머를 돌려준다.
func (c *fakeClock) last(tb testing.TB) *fakeTimer {
	tb.Helper()
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.timers) == 0 {
		tb.Fatal("타이머가 하나도 없음")
	}
	return c.timers[len(c.timers)-1]
}

func newFake() (*Registry, *fakeClock) {
	c := &fakeClock{}
	r := New()
	r.afterFunc = c.afterFunc
	return r, c
}

func TestAddAndGet(t *testing.T) {
	r, _ := newFake()
	if err := r.Add(Spec{MsgID: "m1", Idle: time.Minute, State: 42}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	s, ok := r.Get("m1")
	if !ok {
		t.Fatal("Get(m1) 실패")
	}
	if s.MsgID != "m1" || s.State != 42 {
		t.Errorf("Get(m1) = %+v", s)
	}
	if _, ok := r.Get("없음"); ok {
		t.Error("없는 판이 조회됨")
	}
}

func TestAddRejects(t *testing.T) {
	r, _ := newFake()
	if err := r.Add(Spec{MsgID: "m1", Idle: time.Minute, Users: []string{"a"}}); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		spec Spec
		want error
	}{
		{"빈 메시지 ID", Spec{Idle: time.Minute}, ErrInvalid},
		{"방치 시간 0", Spec{MsgID: "m2"}, ErrInvalid},
		{"같은 메시지 ID", Spec{MsgID: "m1", Idle: time.Minute}, ErrDuplicate},
		{"이미 판이 있는 유저", Spec{MsgID: "m2", Idle: time.Minute, Users: []string{"b", "a"}}, ErrBusy},
		{"한 판에 같은 유저 두 번", Spec{MsgID: "m3", Idle: time.Minute, Users: []string{"c", "c"}}, ErrInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := r.Add(tt.spec); !errors.Is(err, tt.want) {
				t.Errorf("Add(%+v) = %v, want %v", tt.spec, err, tt.want)
			}
		})
	}
	// 거절된 Add 는 흔적을 남기지 않는다: b 는 여전히 판이 없다.
	if r.Busy("b") {
		t.Error("거절된 Add 가 b 를 역인덱스에 남김")
	}
}

func TestUserIndexIsOptional(t *testing.T) {
	r, _ := newFake()
	// 약속 카드처럼 Users 없이 넣은 판은 유저를 묶지 않는다.
	if err := r.Add(Spec{MsgID: "card1", Idle: time.Hour}); err != nil {
		t.Fatal(err)
	}
	if err := r.Add(Spec{MsgID: "card2", Idle: time.Hour}); err != nil {
		t.Fatal(err)
	}
	if err := r.Add(Spec{MsgID: "duel", Idle: time.Minute, Users: []string{"a", "b"}}); err != nil {
		t.Fatalf("카드가 있어도 대결은 만들 수 있어야 함: %v", err)
	}
	if !r.Busy("a") || !r.Busy("b") {
		t.Error("대결 참가자가 Busy 가 아님")
	}
	if id, ok := r.UserSession("a"); !ok || id != "duel" {
		t.Errorf("UserSession(a) = %q, %v", id, ok)
	}
}

func TestFinishOnlyOnce(t *testing.T) {
	r, c := newFake()
	if err := r.Add(Spec{MsgID: "m1", Idle: time.Minute, Users: []string{"a"}}); err != nil {
		t.Fatal(err)
	}
	s, ok := r.Finish("m1")
	if !ok || s == nil || s.MsgID != "m1" {
		t.Fatalf("첫 Finish = %v, %v", s, ok)
	}
	if !s.Done() {
		t.Error("Finish 뒤에 Done() 이 false")
	}
	if _, ok := r.Finish("m1"); ok {
		t.Error("두 번째 Finish 가 true")
	}
	if _, ok := r.Get("m1"); ok {
		t.Error("Finish 뒤에도 판이 남아 있음")
	}
	if r.Busy("a") {
		t.Error("Finish 뒤에도 유저 역인덱스가 남아 있음")
	}
	if !c.last(t).stopped {
		t.Error("Finish 가 타이머를 멈추지 않음")
	}
	// 끝난 판의 유저는 새 판을 만들 수 있다.
	if err := r.Add(Spec{MsgID: "m2", Idle: time.Minute, Users: []string{"a"}}); err != nil {
		t.Errorf("Finish 뒤 새 판 Add: %v", err)
	}
}

func TestExpireCallsOnExpireOnce(t *testing.T) {
	r, c := newFake()
	var calls atomic.Int32
	err := r.Add(Spec{
		MsgID: "m1", Idle: time.Minute, Users: []string{"a"},
		OnExpire: func(s *Session) {
			if s.MsgID != "m1" {
				t.Errorf("OnExpire 에 다른 판: %s", s.MsgID)
			}
			calls.Add(1)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	c.last(t).f()
	c.last(t).f() // 같은 타이머가 두 번 울려도 한 번만 끝난다.
	if calls.Load() != 1 {
		t.Errorf("OnExpire 호출 %d번, want 1", calls.Load())
	}
	if _, ok := r.Get("m1"); ok || r.Busy("a") {
		t.Error("만료 뒤에도 판이나 역인덱스가 남아 있음")
	}
	if _, ok := r.Finish("m1"); ok {
		t.Error("만료된 판의 Finish 가 true")
	}
}

func TestFinishBeforeExpireSkipsOnExpire(t *testing.T) {
	r, c := newFake()
	called := false
	if err := r.Add(Spec{MsgID: "m1", Idle: time.Minute, OnExpire: func(*Session) { called = true }}); err != nil {
		t.Fatal(err)
	}
	stale := c.last(t)
	r.Finish("m1")
	stale.f() // Stop 직전에 이미 발화가 시작된 타이머를 흉내 낸다.
	if called {
		t.Error("Finish 된 판에 OnExpire 가 호출됨")
	}
}

func TestTouchResetsTimer(t *testing.T) {
	r, c := newFake()
	called := false
	if err := r.Add(Spec{MsgID: "m1", Idle: time.Minute, OnExpire: func(*Session) { called = true }}); err != nil {
		t.Fatal(err)
	}
	old := c.last(t)
	if !r.Touch("m1", 3*time.Minute) {
		t.Fatal("Touch(m1) = false")
	}
	if !old.stopped {
		t.Error("Touch 가 이전 타이머를 멈추지 않음")
	}
	fresh := c.last(t)
	if fresh == old || fresh.d != 3*time.Minute {
		t.Errorf("Touch 뒤 새 타이머 = %+v", fresh)
	}
	// 멈추기 직전에 울린 옛 타이머는 판을 끝내면 안 된다.
	old.f()
	if called {
		t.Fatal("옛 타이머가 판을 만료시킴 (방치 시간은 마지막 행동부터 세야 함)")
	}
	if _, ok := r.Get("m1"); !ok {
		t.Fatal("옛 타이머 때문에 판이 사라짐")
	}
	fresh.f()
	if !called {
		t.Error("새 타이머가 울렸는데 만료되지 않음")
	}
}

func TestTouchUnknownOrFinished(t *testing.T) {
	r, _ := newFake()
	if r.Touch("없음", time.Minute) {
		t.Error("없는 판 Touch = true")
	}
	if err := r.Add(Spec{MsgID: "m1", Idle: time.Minute}); err != nil {
		t.Fatal(err)
	}
	r.Finish("m1")
	if r.Touch("m1", time.Minute) {
		t.Error("끝난 판 Touch = true")
	}
}

func TestCloseAll(t *testing.T) {
	r, c := newFake()
	called := false
	for _, id := range []string{"m1", "m2"} {
		if err := r.Add(Spec{MsgID: id, Idle: time.Minute, Users: []string{"u" + id}, OnExpire: func(*Session) { called = true }}); err != nil {
			t.Fatal(err)
		}
	}
	left := r.CloseAll()
	if len(left) != 2 {
		t.Fatalf("CloseAll 이 판 %d개를 돌려줌, want 2", len(left))
	}
	for _, s := range left {
		if !s.Done() {
			t.Errorf("CloseAll 이 돌려준 %s 가 Done 이 아님", s.MsgID)
		}
	}
	c.mu.Lock()
	for i, tm := range c.timers {
		if !tm.stopped {
			t.Errorf("타이머 %d 가 멈추지 않음", i)
		}
	}
	c.mu.Unlock()
	c.last(t).f()
	if called {
		t.Error("CloseAll 뒤에 OnExpire 가 호출됨")
	}
	if _, ok := r.Get("m1"); ok || r.Busy("um1") {
		t.Error("CloseAll 뒤에도 판이나 역인덱스가 남아 있음")
	}
	if err := r.Add(Spec{MsgID: "m3", Idle: time.Minute}); !errors.Is(err, ErrClosed) {
		t.Errorf("CloseAll 뒤 Add = %v, want ErrClosed", err)
	}
	if again := r.CloseAll(); len(again) != 0 {
		t.Errorf("두 번째 CloseAll 이 판 %d개를 돌려줌", len(again))
	}
}

// 아래부터는 진짜 타이머로 경합을 돌린다. go test -race 로 의미가 있다.

func TestRaceFinishConcurrent(t *testing.T) {
	r := New()
	if err := r.Add(Spec{MsgID: "m1", Idle: time.Hour, Users: []string{"a"}}); err != nil {
		t.Fatal(err)
	}
	var wins atomic.Int32
	var wg sync.WaitGroup
	for range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, ok := r.Finish("m1"); ok {
				wins.Add(1)
			}
		}()
	}
	wg.Wait()
	if wins.Load() != 1 {
		t.Errorf("Finish 성공 %d번, want 1", wins.Load())
	}
}

func TestRaceExpireVsClick(t *testing.T) {
	for i := range 200 {
		r := New()
		var ends atomic.Int32
		err := r.Add(Spec{
			MsgID: "m", Idle: time.Millisecond, Users: []string{"a"},
			OnExpire: func(*Session) { ends.Add(1) },
		})
		if err != nil {
			t.Fatal(err)
		}
		time.Sleep(time.Duration(i%3) * time.Millisecond / 2)
		if s, ok := r.Get("m"); ok {
			s.Lock()
			s.State = i // 판마다 락 아래서 상태를 바꾼다.
			s.Unlock()
		}
		r.Touch("m", time.Millisecond)
		if _, ok := r.Finish("m"); ok {
			ends.Add(1)
		}
		time.Sleep(3 * time.Millisecond) // 남은 타이머가 있으면 울리게 둔다.
		if n := ends.Load(); n != 1 {
			t.Fatalf("회차 %d: 판이 %d번 끝남, want 1", i, n)
		}
	}
}
