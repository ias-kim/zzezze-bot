package features

import (
	"bufio"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"

	"zzezze-bot/config"
	"zzezze-bot/interactions"
	"zzezze-bot/quiet"
	"zzezze-bot/voice"
)

// nudgeLines 는 testdata 파일에서 빈 줄과 # 주석을 뺀 줄들을 읽는다.
func nudgeLines(t *testing.T, name string) []string {
	t.Helper()
	f, err := os.Open("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var out []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, line)
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestNudgeTriggerHitData(t *testing.T) {
	counts := map[string]int{}
	for _, line := range nudgeLines(t, "nudge_hit.txt") {
		want, sentence, ok := strings.Cut(line, "\t")
		if !ok {
			t.Fatalf("hit 줄 형식 오류(탭 없음): %q", line)
		}
		counts[want]++
		got, hit := nudgeTrigger(sentence)
		if !hit || got != want {
			t.Errorf("nudgeTrigger(%q) = (%q, %v), want (%q, true)", sentence, got, hit, want)
		}
	}
	for _, trig := range []string{nudgeTriggerChicken, nudgeTriggerTime} {
		if counts[trig] < 10 {
			t.Errorf("hit 데이터가 %s 트리거에 %d개뿐 (10개 이상 필요)", trig, counts[trig])
		}
	}
}

func TestNudgeTriggerMissData(t *testing.T) {
	lines := nudgeLines(t, "nudge_miss.txt")
	if len(lines) < 15 {
		t.Errorf("miss 데이터 %d개 (15개 이상 필요)", len(lines))
	}
	for _, s := range lines {
		if got, hit := nudgeTrigger(s); hit {
			t.Errorf("nudgeTrigger(%q) = (%q, true), want miss", s, got)
		}
	}
}

// nudgeChannel 은 Nudge 기능이 켜졌거나(on) 꺼진 채널을 config 에서 찾는다.
func nudgeChannel(t *testing.T, on bool) string {
	t.Helper()
	for id, cfg := range config.Channels {
		if cfg.Has(config.Nudge) == on {
			return id
		}
	}
	t.Fatalf("Nudge=%v 인 채널이 config 에 없다", on)
	return ""
}

func newTestNudge(now *time.Time) *Nudge {
	n := NewNudge(quiet.New(NudgeCooldown))
	n.Pick = pickFirst
	n.Now = func() time.Time { return *now }
	return n
}

func TestNudgeDecideIgnores(t *testing.T) {
	now := time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC)
	on, off := nudgeChannel(t, true), nudgeChannel(t, false)
	cases := []struct {
		name    string
		channel string
		bot     bool
		content string
	}{
		{"Nudge 꺼진 채널(1번방)", off, false, "치킨 시킬까?"},
		{"등록 안 된 채널", "0", false, "치킨 시킬까?"},
		{"봇 메시지", on, true, "치킨 시킬까?"},
		{"슬래시 명령어", on, false, "/야추 치킨 시킬까?"},
		{"느낌표 명령어", on, false, "!치킨 시킬까?"},
		{"트리거 없음", on, false, "오늘 날씨 좋다"},
	}
	for _, c := range cases {
		n := newTestNudge(&now)
		if trig, ok := n.Decide(c.channel, c.bot, c.content); ok {
			t.Errorf("%s: Decide = (%q, true), want 무시", c.name, trig)
		}
	}
}

func TestNudgeDecideTriggers(t *testing.T) {
	now := time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC)
	on := nudgeChannel(t, true)
	n := newTestNudge(&now)
	if trig, ok := n.Decide(on, false, "치킨 시킬까?"); !ok || trig != nudgeTriggerChicken {
		t.Errorf("chicken: got (%q, %v)", trig, ok)
	}
	if trig, ok := n.Decide(on, false, "몇시에 볼까?"); !ok || trig != nudgeTriggerTime {
		t.Errorf("time: got (%q, %v)", trig, ok)
	}
}

func TestNudgeCooldown(t *testing.T) {
	now := time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC)
	on := nudgeChannel(t, true)
	n := newTestNudge(&now)

	if _, ok := n.Decide(on, false, "치킨 시킬까?"); !ok {
		t.Fatal("첫 반응이 막혔다")
	}
	now = now.Add(NudgeCooldown - time.Second)
	if _, ok := n.Decide(on, false, "치킨 시킬까?"); ok {
		t.Error("쿨다운 안인데 반응했다")
	}
	// 다른 트리거는 따로 센다.
	if _, ok := n.Decide(on, false, "몇시에 볼까?"); !ok {
		t.Error("다른 트리거(time)가 치킨 쿨다운에 막혔다")
	}
	now = now.Add(time.Second)
	if _, ok := n.Decide(on, false, "치킨 시킬까?"); !ok {
		t.Error("쿨다운이 지났는데 막혔다")
	}
}

func TestNudgeIgnoredMessageDoesNotBurnCooldown(t *testing.T) {
	now := time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC)
	on := nudgeChannel(t, true)
	n := newTestNudge(&now)
	n.Decide(on, false, "오늘 날씨 좋다")
	n.Decide(on, true, "치킨 시킬까?") // 봇 메시지
	if _, ok := n.Decide(on, false, "치킨 시킬까?"); !ok {
		t.Error("무시된 메시지가 쿨다운을 소비했다")
	}
}

func TestNudgeMute(t *testing.T) {
	// 2026-11-19 21:00 KST. 음소거는 11-20 00:00 KST(= 11-19 15:00 UTC)까지.
	now := time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC)
	on := nudgeChannel(t, true)
	n := newTestNudge(&now)

	id := interactions.ID{Feature: "nudge", Action: "mute", Arg: nudgeTriggerChicken}
	got, ok := n.applyMute(on, id)
	if !ok {
		t.Fatal("applyMute 가 거절했다")
	}
	cfg, _ := config.Lookup(on)
	if want := voice.Say(voice.NudgeMuted, cfg.Tone, pickFirst); got != want || got == "" {
		t.Errorf("mute 응답 = %q, want %q", got, want)
	}

	now = now.Add(31 * time.Minute) // 쿨다운은 지났지만 음소거 중
	if _, ok := n.Decide(on, false, "치킨 시킬까?"); ok {
		t.Error("음소거 중인데 chicken 에 반응했다")
	}
	if _, ok := n.Decide(on, false, "몇시에 볼까?"); !ok {
		t.Error("chicken 음소거가 time 까지 막았다")
	}
	now = time.Date(2026, 11, 19, 15, 0, 0, 0, time.UTC) // KST 자정
	if _, ok := n.Decide(on, false, "치킨 시킬까?"); !ok {
		t.Error("KST 자정이 지났는데 계속 막혔다")
	}
}

func TestNudgeApplyMuteRejects(t *testing.T) {
	now := time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC)
	on, off := nudgeChannel(t, true), nudgeChannel(t, false)
	cases := []struct {
		name    string
		channel string
		id      interactions.ID
	}{
		{"모르는 트리거", on, interactions.ID{Feature: "nudge", Action: "mute", Arg: "pizza"}},
		{"인자 없음", on, interactions.ID{Feature: "nudge", Action: "mute"}},
		{"모르는 동작", on, interactions.ID{Feature: "nudge", Action: "open", Arg: nudgeTriggerChicken}},
		{"Nudge 꺼진 채널", off, interactions.ID{Feature: "nudge", Action: "mute", Arg: nudgeTriggerChicken}},
	}
	for _, c := range cases {
		n := newTestNudge(&now)
		if _, ok := n.applyMute(c.channel, c.id); ok {
			t.Errorf("%s: applyMute 가 받아들였다", c.name)
		}
		if _, ok := n.Decide(on, false, "치킨 시킬까?"); !ok {
			t.Errorf("%s: 거절했는데 음소거가 걸렸다", c.name)
		}
	}
}

func TestNudgeReply(t *testing.T) {
	cases := []struct {
		trigger string
		text    voice.Key
		openID  string
		openLbl voice.Key
		muteID  string
	}{
		{nudgeTriggerChicken, voice.NudgeChicken, "chicken:open", voice.ChickenBtnLabel, "nudge:mute:chicken"},
		{nudgeTriggerTime, voice.NudgeTime, "timevote:open", voice.TimeBtnLabel, "nudge:mute:time"},
	}
	for _, tone := range []config.Tone{config.ToneMild, config.ToneFull} {
		for _, c := range cases {
			content, comps := nudgeReply(c.trigger, tone, pickFirst)
			if want := voice.Say(c.text, tone, pickFirst); content != want || content == "" {
				t.Errorf("%s/%s content = %q, want %q", c.trigger, tone, content, want)
			}
			if len(comps) != 1 {
				t.Fatalf("%s: ActionsRow %d개, want 1", c.trigger, len(comps))
			}
			row, ok := comps[0].(discordgo.ActionsRow)
			if !ok || len(row.Components) != 2 {
				t.Fatalf("%s: 버튼 2개짜리 ActionsRow 가 아님: %#v", c.trigger, comps[0])
			}
			open, ok1 := row.Components[0].(discordgo.Button)
			mute, ok2 := row.Components[1].(discordgo.Button)
			if !ok1 || !ok2 {
				t.Fatalf("%s: 버튼 타입 불일치", c.trigger)
			}
			if open.CustomID != c.openID || open.Label != voice.Say(c.openLbl, tone, pickFirst) {
				t.Errorf("%s open = %+v", c.trigger, open)
			}
			if mute.CustomID != c.muteID || mute.Label != voice.Say(voice.MuteBtnLabel, tone, pickFirst) {
				t.Errorf("%s mute = %+v", c.trigger, mute)
			}
			// custom_id 는 Decode 로 되돌려 읽혀야 한다(라우터 계약).
			if _, err := interactions.Decode(open.CustomID); err != nil {
				t.Errorf("open custom_id Decode 실패: %v", err)
			}
		}
	}
}
