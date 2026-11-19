package features

import (
	"testing"

	"github.com/bwmarrin/discordgo"

	"zzezze-bot/config"
	"zzezze-bot/interactions"
	"zzezze-bot/session"
	"zzezze-bot/voice"
)

func TestRouterLookup(t *testing.T) {
	r := NewRouter()
	var got interactions.ID
	r.On("chicken", func(_ *discordgo.Session, _ *discordgo.InteractionCreate, id interactions.ID) { got = id })

	h, id, ok := r.lookup("chicken:join")
	if !ok {
		t.Fatal("chicken:join 을 찾지 못함")
	}
	h(nil, nil, id)
	if got != (interactions.ID{Feature: "chicken", Action: "join"}) {
		t.Errorf("전달된 ID = %+v", got)
	}

	for _, bad := range []string{"", "nope:open", "chicken", "a:b:c:d"} {
		if _, _, ok := r.lookup(bad); ok {
			t.Errorf("lookup(%q) 는 실패해야 한다", bad)
		}
	}
}

func TestRouterIgnoresNonComponent(t *testing.T) {
	r := NewRouter()
	called := false
	r.On("chicken", func(*discordgo.Session, *discordgo.InteractionCreate, interactions.ID) { called = true })
	i := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		Type: discordgo.InteractionApplicationCommand,
	}}
	r.Handle(nil, i)
	if called {
		t.Error("버튼이 아닌 상호작용은 무시해야 한다")
	}
}

func TestRouterRecoversPanic(t *testing.T) {
	r := NewRouter()
	r.On("boom", func(*discordgo.Session, *discordgo.InteractionCreate, interactions.ID) { panic("boom") })
	i := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		Type: discordgo.InteractionMessageComponent,
		Data: discordgo.MessageComponentInteractionData{CustomID: "boom:x"},
	}}
	r.Handle(nil, i) // 패닉이 밖으로 새면 테스트가 죽는다
}

func TestSafeMessage(t *testing.T) {
	h := SafeMessage("x", func(*discordgo.Session, *discordgo.MessageCreate) { panic("boom") })
	h(nil, &discordgo.MessageCreate{})
}

type fakeClosable struct {
	ch   string
	tone config.Tone
}

func (f fakeClosable) closeInfo() (string, config.Tone) { return f.ch, f.tone }

func TestShutdownEdits(t *testing.T) {
	first := func(int) int { return 0 }
	closed := []session.Closed{
		{MsgID: "m1", State: fakeClosable{"c1", config.ToneFull}},
		{MsgID: "m2", State: "버튼 없는 상태"},
		{MsgID: "m3", State: fakeClosable{"c3", config.ToneMild}},
	}
	got := shutdownEdits(closed, first)
	if len(got) != 2 {
		t.Fatalf("수정할 메시지 = %d개, want 2: %+v", len(got), got)
	}
	want := []shutdownEdit{
		{ChannelID: "c1", MsgID: "m1", Content: voice.Say(voice.SessionRestart, config.ToneFull, first)},
		{ChannelID: "c3", MsgID: "m3", Content: voice.Say(voice.SessionRestart, config.ToneMild, first)},
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

// 치킨 판 상태가 종료 인터페이스를 실제로 구현하는지 (W1 B 와의 계약).
var _ closable = (*chickenState)(nil)

func TestSafeRunReportsPanic(t *testing.T) {
	if !safeRun("x", func() { panic("boom") }) {
		t.Error("패닉이 있었는데 false")
	}
	if safeRun("x", func() {}) {
		t.Error("패닉이 없었는데 true")
	}
}
