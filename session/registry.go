// Package session 은 버튼 달린 판(빠른 야추 판, 2차의 야추 대결)의 메모리 상태를 관리한다.
//
//   - 메시지 ID 를 키로 판을 보관한다.
//   - 방치 타이머는 마지막 행동부터 센다(Touch).
//   - Finish 는 처음 한 번만 true 를 돌려준다. 그래서 타이머 만료와 클릭이 겹쳐도 한 번만 끝난다.
//   - userID → msgID 역인덱스로 한 유저가 동시에 판 1개만 갖게 한다.
//
// 설계: docs/design/six-member.md "session.Registry", docs/adr/0003-pure-core-shared-session.md
package session

import (
	"errors"
	"slices"
	"sync"
	"time"
)

var (
	ErrExists    = errors.New("session: game already exists for this message")
	ErrBusy      = errors.New("session: user already in another game")
	ErrAlreadyIn = errors.New("session: user already in this game")
	ErrNotFound  = errors.New("session: game not found")
	ErrClosed    = errors.New("session: registry closed")
)

// Game 은 판 하나다. State 는 기능마다 다른 상태를 담는다.
// State 를 읽고 쓸 때는 Mu 를 잡는다 (판마다 락이 따로 있다).
type Game struct {
	MessageID string
	ChannelID string

	Mu    sync.Mutex
	State any

	// Players 는 판이 끝날 때(Finish, 만료, CloseAll) 채워지는 참가자 스냅샷이다.
	// 끝난 뒤에는 바뀌지 않으므로 락 없이 읽어도 된다.
	Players []string

	users    []string // 레지스트리 락으로 보호
	timer    *time.Timer
	onExpire func(*Game)
}

// Registry 는 진행 중인 판 목록이다. 동시 사용에 안전하다.
type Registry struct {
	mu     sync.Mutex
	games  map[string]*Game  // msgID → 판
	byUser map[string]string // userID → msgID
	closed bool
}

func New() *Registry {
	return &Registry{games: map[string]*Game{}, byUser: map[string]string{}}
}

// Start 는 판을 등록하고 방치 타이머(idle)를 건다.
// 타이머가 만료되면 판을 끝내고(Finish) onExpire 를 한 번 부른다. onExpire 는 nil 일 수 있다.
func (r *Registry) Start(msgID, channelID string, users []string, state any, idle time.Duration, onExpire func(*Game)) (*Game, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return nil, ErrClosed
	}
	if _, ok := r.games[msgID]; ok {
		return nil, ErrExists
	}
	for _, u := range users {
		if _, busy := r.byUser[u]; busy {
			return nil, ErrBusy
		}
	}
	g := &Game{MessageID: msgID, ChannelID: channelID, State: state, users: slices.Clone(users), onExpire: onExpire}
	g.timer = time.AfterFunc(idle, func() { r.expire(msgID) })
	r.games[msgID] = g
	for _, u := range users {
		r.byUser[u] = msgID
	}
	return g, nil
}

func (r *Registry) expire(msgID string) {
	if g, ok := r.Finish(msgID); ok && g.onExpire != nil {
		g.onExpire(g)
	}
}

// Get 은 진행 중인 판을 찾는다.
func (r *Registry) Get(msgID string) (*Game, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	g, ok := r.games[msgID]
	return g, ok
}

// Join 은 진행 중인 판에 유저를 더한다.
func (r *Registry) Join(msgID, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	g, ok := r.games[msgID]
	if !ok {
		return ErrNotFound
	}
	if cur, busy := r.byUser[userID]; busy {
		if cur == msgID {
			return ErrAlreadyIn
		}
		return ErrBusy
	}
	g.users = append(g.users, userID)
	r.byUser[userID] = msgID
	return nil
}

// Busy 는 유저가 이미 진행 중인 판에 들어가 있는지 본다.
func (r *Registry) Busy(userID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.byUser[userID]
	return ok
}

// Users 는 판에 들어온 유저를 들어온 순서대로 돌려준다(복사본).
func (r *Registry) Users(msgID string) []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if g, ok := r.games[msgID]; ok {
		return slices.Clone(g.users)
	}
	return nil
}

// Touch 는 방치 타이머를 d 로 다시 건다. 판이 없으면 false.
func (r *Registry) Touch(msgID string, d time.Duration) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	g, ok := r.games[msgID]
	if !ok {
		return false
	}
	g.timer.Reset(d)
	return true
}

// Finish 는 판을 지우고 타이머를 멈춘다. 처음 호출했을 때만 ok=true 다.
func (r *Registry) Finish(msgID string) (*Game, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	g, ok := r.games[msgID]
	if !ok {
		return nil, false
	}
	r.remove(g)
	return g, true
}

func (r *Registry) remove(g *Game) {
	g.timer.Stop()
	g.Players = slices.Clone(g.users)
	delete(r.games, g.MessageID)
	for _, u := range g.users {
		if r.byUser[u] == g.MessageID {
			delete(r.byUser, u)
		}
	}
}

// CloseAll 은 종료할 때 부른다. 모든 타이머를 멈추고 남은 판을 돌려준다.
// 이후의 Start 는 ErrClosed 로 실패한다.
func (r *Registry) CloseAll() []*Game {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closed = true
	out := make([]*Game, 0, len(r.games))
	for _, g := range r.games {
		out = append(out, g)
	}
	for _, g := range out {
		r.remove(g)
	}
	return out
}
