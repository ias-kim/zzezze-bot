// Package session 은 버튼 달린 판(치킨 판, 대결)의 메모리 상태를 관리한다.
// 메시지 ID 를 키로 판을 보관하고, 방치 타이머와 "종료 1회 보장", 유저 역인덱스를 담당한다.
// 인스턴스는 봇 전체에 하나다(ADR 0003). 그래서 "한 유저 동시에 판 1개"는 기능을 가로질러 적용된다.
//
// 락 규칙: Registry.mu 는 짧게만 잡고, 판별 락(entry.mu)은 Do 만 잡는다.
// 중첩은 "판별 락 → Registry.mu" 한 방향뿐이라 Do 의 fn 안에서 Touch/Finish 를 불러도 된다.
// 단, fn 안에서 같은 판의 Do 를 다시 부르면 데드락이다.
//
// fn 안에서는 I/O(디스코드 호출)를 하지 않는다. 상태만 바꾸고 화면 그리기는 Do 밖에서 한다.
// 만료와 CloseAll 도 판별 락을 기다리므로, fn 이 길어지면 3초 응답 제한과 종료 5초 제한이 깨진다.
package session

import (
	"errors"
	"slices"
	"sync"
	"time"
)

// Start, Join, Do 가 돌려주는 에러. errors.Is 로 비교한다.
var (
	ErrExists        = errors.New("session: already exists")
	ErrNotFound      = errors.New("session: not found")
	ErrUserBusy      = errors.New("session: user in another game")
	ErrAlreadyJoined = errors.New("session: user already joined")
	ErrDuplicateUser = errors.New("session: duplicate user")
	ErrClosed        = errors.New("session: registry closed")
)

// stopper 는 *time.Timer 가 만족한다. 테스트는 가짜 타이머를 주입한다.
type stopper interface{ Stop() bool }

// Game 은 Start 에 넘기는 판 정보다.
type Game struct {
	State    any                           // 판 상태. 꺼내는 쪽(features)이 타입을 단언한다.
	Idle     time.Duration                 // 마지막 Touch 부터 이 시간이 지나면 만료된다.
	OnExpire func(msgID string, state any) // 만료됐을 때 락 밖에서 한 번 호출된다. nil 가능.
	Users    []string                      // 시작할 때 참가하는 유저. 비어 있어도 된다.
}

// Closed 는 CloseAll 이 돌려주는 남은 판이다.
type Closed struct {
	MsgID string
	State any
}

type entry struct {
	mu    sync.Mutex // state 보호. Do 만 잡는다.
	state any

	// 아래는 Registry.mu 로 보호한다.
	users    []string
	idle     time.Duration
	onExpire func(string, any)
	timer    stopper
	gen      uint64 // Touch 할 때마다 올린다. 늦게 발화한 이전 타이머를 걸러낸다.
}

// Registry 는 진행 중인 판을 보관한다. 0 값은 쓰지 말고 New 로 만든다.
type Registry struct {
	mu     sync.Mutex
	games  map[string]*entry // msgID → 판
	users  map[string]string // userID → msgID
	closed bool

	afterFunc func(time.Duration, func()) stopper
}

// New 는 실제 시계(time.AfterFunc)를 쓰는 Registry 를 만든다.
func New() *Registry {
	return &Registry{
		games: map[string]*entry{},
		users: map[string]string{},
		afterFunc: func(d time.Duration, f func()) stopper {
			return time.AfterFunc(d, f)
		},
	}
}

// Start 는 판을 등록하고 방치 타이머를 건다. 유저 중 한 명이라도 막히면 아무것도 등록하지 않는다.
func (r *Registry) Start(msgID string, g Game) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return ErrClosed
	}
	if _, ok := r.games[msgID]; ok {
		return ErrExists
	}
	seen := make(map[string]bool, len(g.Users))
	for _, u := range g.Users {
		if seen[u] {
			return ErrDuplicateUser
		}
		seen[u] = true
		if _, busy := r.users[u]; busy {
			return ErrUserBusy
		}
	}
	e := &entry{
		state:    g.State,
		users:    slices.Clone(g.Users),
		idle:     g.Idle,
		onExpire: g.OnExpire,
	}
	for _, u := range e.users {
		r.users[u] = msgID
	}
	r.games[msgID] = e
	r.arm(msgID, e)
	return nil
}

// Join 은 진행 중인 판에 유저를 추가한다.
func (r *Registry) Join(msgID, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	e, ok := r.games[msgID]
	if !ok {
		return ErrNotFound
	}
	if cur, busy := r.users[userID]; busy {
		if cur == msgID {
			return ErrAlreadyJoined
		}
		return ErrUserBusy
	}
	e.users = append(e.users, userID)
	r.users[userID] = msgID
	return nil
}

// Touch 는 방치 타이머를 처음부터 다시 건다. 판이 없으면 false.
func (r *Registry) Touch(msgID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	e, ok := r.games[msgID]
	if !ok {
		return false
	}
	r.arm(msgID, e)
	return true
}

// Finish 는 판을 끝낸다. 처음 호출됐을 때만 true 다(타이머 만료와 경합해도 한 번만 끝난다).
func (r *Registry) Finish(msgID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	e, ok := r.games[msgID]
	if !ok {
		return false
	}
	r.remove(msgID, e)
	return true
}

// Do 는 판별 락 안에서 fn 에 판 상태를 넘긴다. 판이 없거나 기다리는 사이 끝났으면 ErrNotFound.
func (r *Registry) Do(msgID string, fn func(state any)) error {
	r.mu.Lock()
	e, ok := r.games[msgID]
	r.mu.Unlock()
	if !ok {
		return ErrNotFound
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	r.mu.Lock()
	alive := r.games[msgID] == e
	r.mu.Unlock()
	if !alive {
		return ErrNotFound
	}
	fn(e.state)
	return nil
}

// UserGame 은 유저가 들어가 있는 판의 msgID 를 돌려준다.
func (r *Registry) UserGame(userID string) (string, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	msgID, ok := r.users[userID]
	return msgID, ok
}

// CloseAll 은 종료할 때 부른다. 모든 타이머를 멈추고 남은 판을 돌려주며, 이후 Start 는 ErrClosed 다.
func (r *Registry) CloseAll() []Closed {
	r.mu.Lock()
	r.closed = true
	type pair struct {
		msgID string
		e     *entry
	}
	left := make([]pair, 0, len(r.games))
	for msgID, e := range r.games {
		left = append(left, pair{msgID, e})
		r.remove(msgID, e)
	}
	r.mu.Unlock()

	out := make([]Closed, 0, len(left))
	for _, p := range left {
		p.e.mu.Lock()
		out = append(out, Closed{MsgID: p.msgID, State: p.e.state})
		p.e.mu.Unlock()
	}
	return out
}

// arm 은 r.mu 를 잡은 채로 부른다.
func (r *Registry) arm(msgID string, e *entry) {
	if e.timer != nil {
		e.timer.Stop()
	}
	e.gen++
	gen := e.gen
	e.timer = r.afterFunc(e.idle, func() { r.expire(msgID, e, gen) })
}

// remove 는 r.mu 를 잡은 채로 부른다. 이 판을 가리키는 역인덱스만 지운다.
func (r *Registry) remove(msgID string, e *entry) {
	delete(r.games, msgID)
	for _, u := range e.users {
		if r.users[u] == msgID {
			delete(r.users, u)
		}
	}
	if e.timer != nil {
		e.timer.Stop()
	}
}

// expire 는 타이머 콜백이다. Finish 와 같은 종료 경로를 거치고, onExpire 는 모든 락 밖에서 부른다.
func (r *Registry) expire(msgID string, e *entry, gen uint64) {
	r.mu.Lock()
	if r.games[msgID] != e || e.gen != gen {
		r.mu.Unlock()
		return
	}
	r.remove(msgID, e)
	r.mu.Unlock()

	if e.onExpire == nil {
		return
	}
	e.mu.Lock()
	state := e.state
	e.mu.Unlock()
	e.onExpire(msgID, state)
}
