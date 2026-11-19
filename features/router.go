package features

import (
	"context"
	"log"
	"runtime/debug"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"

	"zzezze-bot/config"
	"zzezze-bot/interactions"
	"zzezze-bot/session"
	"zzezze-bot/voice"
)

// ComponentHandler 는 custom_id 의 feature 하나를 맡는 버튼 핸들러다.
type ComponentHandler func(s *discordgo.Session, i *discordgo.InteractionCreate, id interactions.ID)

// Router 는 버튼 상호작용을 custom_id 의 feature 로 나눠 보낸다. 등록은 시작할 때만 한다.
type Router struct {
	handlers map[string]ComponentHandler
}

func NewRouter() *Router { return &Router{handlers: map[string]ComponentHandler{}} }

// On 은 feature 에 핸들러를 등록한다.
func (r *Router) On(feature string, h ComponentHandler) { r.handlers[feature] = h }

// lookup 은 custom_id 를 해석해 핸들러를 찾는다. 형식이 틀리거나 모르는 feature 면 ok=false.
func (r *Router) lookup(customID string) (ComponentHandler, interactions.ID, bool) {
	id, err := interactions.Decode(customID)
	if err != nil {
		return nil, interactions.ID{}, false
	}
	h, ok := r.handlers[id.Feature]
	return h, id, ok
}

// Handle 은 InteractionCreate 핸들러다. 버튼이 아닌 상호작용은 무시하고, 핸들러 패닉은 로그로 남긴다.
func (r *Router) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i == nil || i.Interaction == nil || i.Type != discordgo.InteractionMessageComponent {
		return
	}
	customID := i.MessageComponentData().CustomID
	h, id, ok := r.lookup(customID)
	if !ok {
		log.Printf("모르는 버튼 custom_id=%q", customID)
		return
	}
	safeRun("button "+id.Feature, func() { h(s, i, id) })
}

// SafeMessage 는 메시지 핸들러의 패닉을 로그로 남기고 삼킨다(discordgo 는 핸들러 패닉을 잡지 않는다).
func SafeMessage(name string, h func(*discordgo.Session, *discordgo.MessageCreate)) func(*discordgo.Session, *discordgo.MessageCreate) {
	return func(s *discordgo.Session, m *discordgo.MessageCreate) {
		safeRun(name, func() { h(s, m) })
	}
}

// safeRun 은 f 의 패닉을 스택과 함께 로그로 남긴다. 그래도 프로세스가 죽으면 systemd 가 재시작한다.
func safeRun(name string, f func()) {
	defer func() {
		if v := recover(); v != nil {
			log.Printf("패닉 복구 (%s): %v\n%s", name, v, debug.Stack())
		}
	}()
	f()
}

// closable 은 종료할 때 버튼을 끄고 안내를 띄울 판 상태가 구현한다.
type closable interface {
	closeInfo() (channelID string, tone config.Tone)
}

// shutdownEdit 은 종료할 때 고칠 메시지 하나다.
type shutdownEdit struct {
	ChannelID, MsgID, Content string
}

// shutdownEdits 는 남은 판 중 closable 인 것만 골라 재시작 안내로 바꿀 목록을 만든다.
func shutdownEdits(closed []session.Closed, pick voice.Picker) []shutdownEdit {
	var out []shutdownEdit
	for _, c := range closed {
		cl, ok := c.State.(closable)
		if !ok {
			continue
		}
		ch, tone := cl.closeInfo()
		out = append(out, shutdownEdit{ChannelID: ch, MsgID: c.MsgID, Content: voice.Say(voice.SessionRestart, tone, pick)})
	}
	return out
}

// Shutdown 은 남은 판을 모두 닫고, timeout 안에서 각 판 메시지의 버튼을 끄고 재시작 안내를 띄운다.
// 시간이 넘으면 남은 수정은 포기한다(끝난 판의 버튼은 누르면 만료 안내가 나간다).
func Shutdown(s *discordgo.Session, reg *session.Registry, pick voice.Picker, timeout time.Duration) {
	edits := shutdownEdits(reg.CloseAll(), pick)
	if len(edits) == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	var wg sync.WaitGroup
	for _, e := range edits {
		wg.Add(1)
		go func(e shutdownEdit) {
			defer wg.Done()
			safeRun("shutdown edit", func() {
				if err := editMessage(s, e.ChannelID, e.MsgID, e.Content, nil); err != nil {
					log.Printf("종료 안내 실패 (msg=%s): %v", e.MsgID, err)
				}
			})
		}(e)
	}
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-ctx.Done():
		log.Printf("종료 안내 %d건 중 일부가 %s 안에 끝나지 않음", len(edits), timeout)
	}
}
