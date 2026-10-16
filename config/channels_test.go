package config

import "testing"

const (
	chatRoom = "1482655053646135440" // #채팅방 (1번방)
	gaeggul  = "1538222697501298782" // #개꿀이즤 (2번방)
)

func TestEnabled(t *testing.T) {
	tests := []struct {
		name    string
		channel string
		feature Feature
		want    bool
	}{
		{"1번방 자동 반응", chatRoom, AutoReply, true},
		{"1번방 야추", chatRoom, Yacht, true},
		{"1번방은 nudge 없음", chatRoom, Nudge, false},
		{"2번방 자동 반응", gaeggul, AutoReply, true},
		{"2번방 야추", gaeggul, Yacht, true},
		{"2번방 nudge", gaeggul, Nudge, true},
		{"등록 안 된 채널", "999", AutoReply, false},
		{"빈 채널 ID", "", Yacht, false},
		{"알 수 없는 기능", gaeggul, Feature("moderation"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Enabled(tt.channel, tt.feature); got != tt.want {
				t.Errorf("Enabled(%q, %q) = %v, want %v", tt.channel, tt.feature, got, tt.want)
			}
		})
	}
}

func TestLookupTone(t *testing.T) {
	tests := []struct {
		channel string
		want    Tone
	}{
		{chatRoom, ToneMild},
		{gaeggul, ToneFull},
	}
	for _, tt := range tests {
		cfg, ok := Lookup(tt.channel)
		if !ok {
			t.Fatalf("Lookup(%q) not found", tt.channel)
		}
		if cfg.Tone != tt.want {
			t.Errorf("Lookup(%q).Tone = %q, want %q", tt.channel, cfg.Tone, tt.want)
		}
	}
}

func TestLookupUnknown(t *testing.T) {
	if _, ok := Lookup("999"); ok {
		t.Error("Lookup(unknown) should return ok=false")
	}
}

// 모든 등록 채널은 톤과 이름이 비어 있으면 안 된다 (설정 실수 방지).
func TestChannelsAreComplete(t *testing.T) {
	for id, cfg := range Channels {
		if cfg.Name == "" {
			t.Errorf("channel %s: Name is empty", id)
		}
		if cfg.Tone != ToneMild && cfg.Tone != ToneFull {
			t.Errorf("channel %s: invalid Tone %q", id, cfg.Tone)
		}
	}
}
