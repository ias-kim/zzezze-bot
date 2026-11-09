package session

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestStartAndGet(t *testing.T) {
	r := New()
	g, err := r.Start("m1", "c1", []string{"u1"}, "state", time.Hour, nil)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := r.Get("m1")
	if !ok || got != g || got.ChannelID != "c1" || got.State != "state" {
		t.Fatalf("Get = %+v, %v", got, ok)
	}
	if _, ok := r.Get("nope"); ok {
		t.Error("없는 판이 조회됨")
	}
}

func TestStartRejectsDuplicateMessage(t *testing.T) {
	r := New()
	if _, err := r.Start("m1", "c1", nil, nil, time.Hour, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Start("m1", "c1", nil, nil, time.Hour, nil); !errors.Is(err, ErrExists) {
		t.Errorf("err = %v, want ErrExists", err)
	}
}

// 한 유저는 동시에 판 1개만 가진다.
func TestOneGamePerUser(t *testing.T) {
	r := New()
	if _, err := r.Start("m1", "c1", []string{"u1"}, nil, time.Hour, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Start("m2", "c1", []string{"u2", "u1"}, nil, time.Hour, nil); !errors.Is(err, ErrBusy) {
		t.Errorf("Start err = %v, want ErrBusy", err)
	}
	// 실패한 Start 는 u2 를 묶어두면 안 된다.
	if _, err := r.Start("m3", "c1", []string{"u2"}, nil, time.Hour, nil); err != nil {
		t.Errorf("u2 should be free: %v", err)
	}
	if err := r.Join("m3", "u1"); !errors.Is(err, ErrBusy) {
		t.Errorf("Join err = %v, want ErrBusy", err)
	}
	if err := r.Join("m1", "u1"); !errors.Is(err, ErrAlreadyIn) {
		t.Errorf("Join same game err = %v, want ErrAlreadyIn", err)
	}
	if err := r.Join("zz", "u9"); !errors.Is(err, ErrNotFound) {
		t.Errorf("Join missing err = %v, want ErrNotFound", err)
	}
	if err := r.Join("m1", "u9"); err != nil {
		t.Fatal(err)
	}
	if got := r.Users("m1"); len(got) != 2 || got[0] != "u1" || got[1] != "u9" {
		t.Errorf("users = %v", got)
	}
}

func TestFinishOnlyOnceAndFreesUsers(t *testing.T) {
	r := New()
	r.Start("m1", "c1", []string{"u1"}, nil, time.Hour, nil)
	if _, ok := r.Finish("m1"); !ok {
		t.Fatal("첫 Finish 는 true")
	}
	if _, ok := r.Finish("m1"); ok {
		t.Error("두 번째 Finish 는 false")
	}
	if _, ok := r.Get("m1"); ok {
		t.Error("Finish 뒤에도 판이 남아 있음")
	}
	if _, err := r.Start("m2", "c1", []string{"u1"}, nil, time.Hour, nil); err != nil {
		t.Errorf("Finish 뒤 u1 이 풀려야 함: %v", err)
	}
}

func TestExpireCallsOnce(t *testing.T) {
	r := New()
	done := make(chan *Game, 2)
	r.Start("m1", "c1", []string{"u1"}, nil, 10*time.Millisecond, func(g *Game) { done <- g })
	select {
	case g := <-done:
		if g.MessageID != "m1" {
			t.Errorf("expired %q", g.MessageID)
		}
	case <-time.After(time.Second):
		t.Fatal("만료 콜백이 오지 않음")
	}
	if _, ok := r.Get("m1"); ok {
		t.Error("만료된 판이 남아 있음")
	}
	select {
	case <-done:
		t.Error("만료 콜백이 두 번 옴")
	case <-time.After(30 * time.Millisecond):
	}
}

// 방치 시간은 마지막 행동부터 센다.
func TestTouchResetsTimer(t *testing.T) {
	r := New()
	var fired atomic.Bool
	r.Start("m1", "c1", nil, nil, 60*time.Millisecond, func(*Game) { fired.Store(true) })
	for range 4 {
		time.Sleep(30 * time.Millisecond)
		if !r.Touch("m1", 60*time.Millisecond) {
			t.Fatal("Touch 실패")
		}
	}
	if fired.Load() {
		t.Fatal("Touch 했는데 만료됨")
	}
	if r.Touch("nope", time.Second) {
		t.Error("없는 판 Touch 가 true")
	}
}

// 타이머 만료와 클릭(Finish)이 동시에 와도 정확히 한 번만 끝난다.
func TestFinishRacesWithExpiry(t *testing.T) {
	for range 50 {
		r := New()
		var wins atomic.Int32
		r.Start("m", "c", []string{"u"}, nil, time.Millisecond, func(*Game) { wins.Add(1) })
		var wg sync.WaitGroup
		for range 4 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				time.Sleep(time.Millisecond)
				if _, ok := r.Finish("m"); ok {
					wins.Add(1)
				}
			}()
		}
		wg.Wait()
		time.Sleep(5 * time.Millisecond)
		if n := wins.Load(); n != 1 {
			t.Fatalf("끝난 횟수 = %d, want 1", n)
		}
	}
}

func TestCloseAll(t *testing.T) {
	r := New()
	var fired atomic.Bool
	r.Start("m1", "c1", []string{"u1"}, nil, 20*time.Millisecond, func(*Game) { fired.Store(true) })
	r.Start("m2", "c2", []string{"u2"}, nil, 20*time.Millisecond, func(*Game) { fired.Store(true) })
	got := r.CloseAll()
	if len(got) != 2 {
		t.Fatalf("CloseAll = %d개, want 2", len(got))
	}
	time.Sleep(40 * time.Millisecond)
	if fired.Load() {
		t.Error("CloseAll 뒤에 만료 콜백이 불림")
	}
	if _, ok := r.Get("m1"); ok {
		t.Error("CloseAll 뒤에 판이 남아 있음")
	}
	if _, err := r.Start("m3", "c1", []string{"u1"}, nil, time.Hour, nil); !errors.Is(err, ErrClosed) {
		t.Errorf("CloseAll 뒤 Start err = %v, want ErrClosed", err)
	}
}

// 끝난 판은 Players 에 참가자 스냅샷을 남긴다 (만료 콜백에서 결과를 그릴 때 쓴다).
func TestPlayersSnapshotOnFinish(t *testing.T) {
	r := New()
	got := make(chan []string, 1)
	r.Start("m1", "c1", []string{"u1"}, nil, 20*time.Millisecond, func(g *Game) { got <- g.Players })
	r.Join("m1", "u2")
	select {
	case p := <-got:
		if len(p) != 2 || p[0] != "u1" || p[1] != "u2" {
			t.Errorf("Players = %v", p)
		}
	case <-time.After(time.Second):
		t.Fatal("만료 안 됨")
	}
}

func TestBusy(t *testing.T) {
	r := New()
	r.Start("m1", "c1", []string{"u1"}, nil, time.Hour, nil)
	if !r.Busy("u1") || r.Busy("u2") {
		t.Error("Busy 판정 오류")
	}
	r.Finish("m1")
	if r.Busy("u1") {
		t.Error("Finish 뒤에는 풀려야 함")
	}
}
