package config

import "testing"

// 방 구분 폐지(ADR 0006): 어떤 채널이든 기본 설정을 쓴다.
func TestEnabledEverywhere(t *testing.T) {
	for _, ch := range []string{"1482655053646135440", "1538222697501298782", "999"} {
		for _, f := range []Feature{AutoReply, Yacht, Nudge} {
			if !Enabled(ch, f) {
				t.Errorf("Enabled(%q, %q) = false, want true", ch, f)
			}
		}
		if cfg, _ := Lookup(ch); cfg.Tone != ToneFull {
			t.Errorf("Lookup(%q).Tone = %q, want full", ch, cfg.Tone)
		}
	}
}

func TestEnabledRejects(t *testing.T) {
	if Enabled("", AutoReply) {
		t.Error("빈 채널 ID는 꺼져 있어야 한다")
	}
	if Enabled("999", Feature("moderation")) {
		t.Error("알 수 없는 기능은 꺼져 있어야 한다")
	}
}

func TestOverrideWins(t *testing.T) {
	old := Overrides
	t.Cleanup(func() { Overrides = old })
	Overrides = map[string]ChannelConfig{"42": {Name: "조용한 방", Features: []Feature{AutoReply}, Tone: ToneMild}}

	if Enabled("42", Nudge) {
		t.Error("override 채널은 nudge가 꺼져 있어야 한다")
	}
	if cfg, _ := Lookup("42"); cfg.Tone != ToneMild {
		t.Errorf("override tone = %q, want mild", cfg.Tone)
	}
	if !Enabled("43", Nudge) {
		t.Error("override 없는 채널은 기본 설정이어야 한다")
	}
}

// 기본 설정과 모든 override 는 톤과 이름이 비어 있으면 안 된다 (설정 실수 방지).
func TestConfigsAreComplete(t *testing.T) {
	all := map[string]ChannelConfig{"<default>": Default}
	for id, cfg := range Overrides {
		all[id] = cfg
	}
	for id, cfg := range all {
		if cfg.Name == "" {
			t.Errorf("%s: Name is empty", id)
		}
		if cfg.Tone != ToneMild && cfg.Tone != ToneFull {
			t.Errorf("%s: invalid Tone %q", id, cfg.Tone)
		}
	}
}
