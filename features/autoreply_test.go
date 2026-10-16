package features

import (
	"testing"
	"time"

	"zzezze-bot/config"
	"zzezze-bot/quiet"
	"zzezze-bot/voice"
)

func TestMemeHit(t *testing.T) {
	hits := []string{
		"쮀쮀이야",
		"쮀쮀이야 티~원~",
		"아.. 예?.. 쮀쮀이야~",
		"ㅋㅋㅋ쮀쮀쮀쮀",
		"쮀쮀",
		"줴줴",
		"줴줴이야~",
		"체재희야~",
		"제제이야~",
		"재재이야",
		"쩨쩨이야",
		"주ㅔ주ㅔ이야",
		"주 ㅔ 제제이야~",
		"쮀쮀 이야",
	}
	misses := []string{
		"",
		"쮀",
		"줘줘",
		"졸졸 따라와",
		"이제 재희야 밥 먹자",
		"제발 이야기 좀 하자",
		"재미있다",
		"치킨 시킬까?",
		"제 이야기는요",
	}
	for _, s := range hits {
		if !MemeHit(s) {
			t.Errorf("MemeHit(%q) = false, want true", s)
		}
	}
	for _, s := range misses {
		if MemeHit(s) {
			t.Errorf("MemeHit(%q) = true, want false", s)
		}
	}
}

// channelWith 는 config 에서 AutoReply 가 켜진 해당 톤의 채널을 찾는다.
// 채널 ID 를 테스트에 복사하지 않기 위해서다.
func channelWith(t *testing.T, tone config.Tone) string {
	t.Helper()
	for id, cfg := range config.Channels {
		if cfg.Tone == tone && cfg.Has(config.AutoReply) {
			return id
		}
	}
	t.Fatalf("AutoReply 가 켜진 %s 톤 채널이 config 에 없다", tone)
	return ""
}

func pickFirst(int) int { return 0 }

func newTestAutoReply(now *time.Time) *AutoReply {
	a := NewAutoReply(quiet.New(MemeCooldown))
	a.Pick = pickFirst
	a.Now = func() time.Time { return *now }
	return a
}

func TestDecideUsesChannelTone(t *testing.T) {
	now := time.Date(2026, 10, 16, 20, 0, 0, 0, time.UTC)
	for _, tone := range []config.Tone{config.ToneMild, config.ToneFull} {
		a := newTestAutoReply(&now)
		got, ok := a.Decide(channelWith(t, tone), false, "쮀쮀이야~")
		want := voice.Say(voice.MemeReply, tone, pickFirst)
		if !ok || got != want {
			t.Errorf("%s: Decide = %q, %v; want %q, true", tone, got, ok, want)
		}
	}
}

func TestDecideIgnores(t *testing.T) {
	now := time.Date(2026, 10, 16, 20, 0, 0, 0, time.UTC)
	ch := channelWith(t, config.ToneFull)
	tests := []struct {
		name    string
		channel string
		fromBot bool
		content string
	}{
		{"봇 메시지", ch, true, "쮀쮀이야"},
		{"/ 명령어", ch, false, "/쮀쮀이야"},
		{"! 명령어", ch, false, "!쮀쮀이야"},
		{"등록 안 된 채널", "000", false, "쮀쮀이야"},
		{"트리거 아님", ch, false, "치킨 시킬까"},
	}
	for _, tt := range tests {
		a := newTestAutoReply(&now)
		if got, ok := a.Decide(tt.channel, tt.fromBot, tt.content); ok {
			t.Errorf("%s: Decide = %q, true; want 반응 없음", tt.name, got)
		}
	}
}

func TestDecideIgnoresChannelWithoutFeature(t *testing.T) {
	const id = "test-no-autoreply"
	config.Channels[id] = config.ChannelConfig{Name: "테스트", Features: []config.Feature{config.Yacht}, Tone: config.ToneFull}
	t.Cleanup(func() { delete(config.Channels, id) })

	now := time.Date(2026, 10, 16, 20, 0, 0, 0, time.UTC)
	if _, ok := newTestAutoReply(&now).Decide(id, false, "쮀쮀이야"); ok {
		t.Error("AutoReply 가 꺼진 채널에서 반응했다")
	}
}

func TestDecideCooldown(t *testing.T) {
	now := time.Date(2026, 10, 16, 20, 0, 0, 0, time.UTC)
	a := newTestAutoReply(&now)
	mild, full := channelWith(t, config.ToneMild), channelWith(t, config.ToneFull)

	if _, ok := a.Decide(full, false, "쮀쮀이야"); !ok {
		t.Fatal("첫 반응이 없다")
	}
	now = now.Add(MemeCooldown - time.Second)
	if _, ok := a.Decide(full, false, "쮀쮀이야"); ok {
		t.Error("쿨다운 안에 다시 반응했다")
	}
	if _, ok := a.Decide(mild, false, "쮀쮀이야"); !ok {
		t.Error("다른 채널이 쿨다운에 막혔다")
	}
	now = now.Add(time.Second)
	if _, ok := a.Decide(full, false, "쮀쮀이야"); !ok {
		t.Error("쿨다운이 끝났는데 반응하지 않았다")
	}
}

func TestDecideNonTriggerDoesNotStartCooldown(t *testing.T) {
	now := time.Date(2026, 10, 16, 20, 0, 0, 0, time.UTC)
	a := newTestAutoReply(&now)
	ch := channelWith(t, config.ToneFull)
	a.Decide(ch, false, "치킨 시킬까")
	if _, ok := a.Decide(ch, false, "쮀쮀이야"); !ok {
		t.Error("트리거가 아닌 말이 쿨다운을 걸었다")
	}
}
