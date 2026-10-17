// Package session 은 버튼 달린 판(약속 카드, 2차 대결)의 메모리 상태를 한 곳에서 관리한다.
//
// 판은 봇 메시지 ID 를 키로 보관한다. 방치 타이머, 1회 종료 보장, 유저 역인덱스,
// 종료할 때 전부 닫기를 여기서 맡아서 기능마다 동시성 코드를 따로 짜지 않게 한다.
// 근거: docs/adr/0003-pure-core-shared-session.md
//
// 락 순서: 판 락(Session.Lock) → 레지스트리 락. 레지스트리는 자기 락을 쥔 채로
// 판 락을 잡거나 OnExpire 를 부르지 않는다. 그래서 판 락을 쥔 채 Finish/Touch 를 불러도 된다.
package session

import (
	"errors"
	"fmt"
	"slices"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrInvalid   = errors.New("session: invalid spec")
	ErrDuplicate = errors.New("session: duplicate message id")
	ErrBusy      = errors.New("session: user already in a session") // voice.AlreadyInGame 로 안내
	ErrClosed    = errors.New("session: registry closed")
)

// timer 는 time.Timer 중 레지스트리가 쓰는 부분이다. 테스트에서 가짜 시계로 바꾼다.
type timer interface{ Stop() bool }

func realAfterFunc(d time.Duration, f func()) timer { return time.AfterFunc(d, f) }

// Spec 은 새 판을 등록할 때 넘기는 값이다.
type Spec struct {
	MsgID string
	// Users 는 유저 역인덱스에 묶을 참가자다. 묶인 유저는 동시에 판 1개만 가진다.
	// 약속 카드처럼 여러 판에 동시에 응답할 수 있는 기능은 비워 둔다.
	Users []string
	// Idle 은 방치 시간이다. 마지막 Touch 부터 센다.
	Idle  time.Duration
	State any
	// OnExpire 는 방치 시간이 지나 판이 끝났을 때 한 번 불린다(타이머 고루틴에서).
	// Finish 나 CloseAll 로 끝난 판에는 불리지 않는다.
	OnExpire func(*Session)
}

// Session 은 판 하나다. State 는 Lock/Unlock 사이에서만 읽고 쓴다.
type Session struct {
	MsgID string
	State any

	mu       sync.Mutex
	users    []string
	onExpire func(*Session)
	done     atomic.Bool

	// 아래는 레지스트리 락 아래서만 쓴다.
	t   timer
	gen uint64 // Touch 할 때마다 올린다. 멈추기 직전에 울린 옛 타이머를 무시하기 위해서다.
}

func (s *Session) Lock()   { s.mu.Lock() }
func (s *Session) Unlock() { s.mu.Unlock() }

// Done 은 판이 이미 끝났는지 알려준다. Get 으로 받은 뒤 판 락을 잡고 다시 확인할 때 쓴다.
func (s *Session) Done() bool { return s.done.Load() }

// Registry 는 진행 중인 판 목록이다. 0 값은 쓸 수 없고 New 로 만든다.
type Registry struct {
	mu       sync.Mutex
	sessions map[string]*Session // 메시지 ID → 판
	byUser   map[string]string   // 유저 ID → 메시지 ID
	closed   bool

	afterFunc func(time.Duration, func()) timer
}

func New() *Registry {
	return &Registry{
		sessions:  map[string]*Session{},
		byUser:    map[string]string{},
		afterFunc: realAfterFunc,
	}
}

// Add 는 판을 등록하고 방치 타이머를 건다.
func (r *Registry) Add(sp Spec) error {
	if sp.MsgID == "" || sp.Idle <= 0 {
		return fmt.Errorf("%w: msgID=%q idle=%v", ErrInvalid, sp.MsgID, sp.Idle)
	}
	users := slices.Clone(sp.Users)
	slices.Sort(users)
	if len(slices.Compact(slices.Clone(users))) != len(users) {
		return fmt.Errorf("%w: duplicate user in %v", ErrInvalid, sp.Users)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return ErrClosed
	}
	if _, ok := r.sessions[sp.MsgID]; ok {
		return fmt.Errorf("%w: %s", ErrDuplicate, sp.MsgID)
	}
	for _, u := range users {
		if _, ok := r.byUser[u]; ok {
			return fmt.Errorf("%w: %s", ErrBusy, u)
		}
	}

	s := &Session{MsgID: sp.MsgID, State: sp.State, users: users, onExpire: sp.OnExpire}
	r.sessions[s.MsgID] = s
	for _, u := range users {
		r.byUser[u] = s.MsgID
	}
	r.armLocked(s, sp.Idle)
	return nil
}

// Get 은 진행 중인 판을 돌려준다.
func (r *Registry) Get(msgID string) (*Session, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.sessions[msgID]
	return s, ok
}

// Busy 는 유저가 역인덱스에 묶인 판을 갖고 있는지 알려준다.
func (r *Registry) Busy(userID string) bool {
	_, ok := r.UserSession(userID)
	return ok
}

// UserSession 은 유저가 묶인 판의 메시지 ID 를 돌려준다.
func (r *Registry) UserSession(userID string) (string, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	id, ok := r.byUser[userID]
	return id, ok
}

// Touch 는 방치 타이머를 지금부터 d 로 다시 건다. 판이 없거나 끝났으면 false.
func (r *Registry) Touch(msgID string, d time.Duration) bool {
	if d <= 0 {
		return false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.sessions[msgID]
	if !ok {
		return false
	}
	r.armLocked(s, d)
	return true
}

// Finish 는 판을 끝낸다. 처음 부른 쪽만 판과 true 를 받는다.
// 타이머 만료와 클릭이 동시에 들어와도 한쪽만 끝낸다.
func (r *Registry) Finish(msgID string) (*Session, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.sessions[msgID]
	if !ok || !r.removeLocked(s) {
		return nil, false
	}
	return s, true
}

// CloseAll 은 종료할 때 부른다. 모든 타이머를 멈추고 남은 판을 돌려준다.
// 이후의 Add 는 ErrClosed 를 받는다. 버튼 끄기와 재시작 안내는 호출한 쪽이 한다.
func (r *Registry) CloseAll() []*Session {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closed = true
	left := make([]*Session, 0, len(r.sessions))
	for _, s := range r.sessions {
		if r.removeLocked(s) {
			left = append(left, s)
		}
	}
	return left
}

// armLocked 는 기존 타이머를 멈추고 새 타이머를 건다. r.mu 를 쥔 채로 부른다.
func (r *Registry) armLocked(s *Session, d time.Duration) {
	if s.t != nil {
		s.t.Stop()
	}
	s.gen++
	gen := s.gen
	s.t = r.afterFunc(d, func() { r.expire(s, gen) })
}

func (r *Registry) expire(s *Session, gen uint64) {
	r.mu.Lock()
	if s.gen != gen || !r.removeLocked(s) {
		r.mu.Unlock()
		return
	}
	r.mu.Unlock()
	if s.onExpire != nil {
		s.onExpire(s)
	}
}

// removeLocked 는 판과 역인덱스를 지우고 타이머를 멈춘다. 이미 끝난 판이면 false.
func (r *Registry) removeLocked(s *Session) bool {
	if s.done.Load() {
		return false
	}
	s.done.Store(true)
	delete(r.sessions, s.MsgID)
	for _, u := range s.users {
		if r.byUser[u] == s.MsgID {
			delete(r.byUser, u)
		}
	}
	if s.t != nil {
		s.t.Stop()
	}
	return true
}
