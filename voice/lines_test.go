package voice

import (
	"strings"
	"testing"

	"zzezze-bot/config"
)

// 첫 번째 변형을 고르는 결정적 picker.
func first(int) int { return 0 }

func TestEveryKeyHasBothTones(t *testing.T) {
	for _, k := range AllKeys() {
		for _, tone := range []config.Tone{config.ToneFull, config.ToneMild} {
			if n := len(lines[k][tone]); n == 0 {
				t.Errorf("key %q has no %q lines", k, tone)
			}
		}
	}
}

func TestAllKeysMatchesTable(t *testing.T) {
	if len(AllKeys()) != len(lines) {
		t.Errorf("AllKeys()=%d but lines table has %d keys", len(AllKeys()), len(lines))
	}
}

func TestSayReplacesPlaceholders(t *testing.T) {
	got := Say(ChickenLoser, config.ToneFull, first, "{user}", "니얼굴디크")
	if !strings.Contains(got, "니얼굴디크") {
		t.Errorf("placeholder not replaced: %q", got)
	}
	if strings.Contains(got, "{user}") {
		t.Errorf("raw placeholder left: %q", got)
	}
}

func TestSayUsesPicker(t *testing.T) {
	variants := lines[MemeReply][config.ToneFull]
	if len(variants) < 2 {
		t.Skip("need 2+ variants")
	}
	last := func(n int) int { return n - 1 }
	if got := Say(MemeReply, config.ToneFull, last); got != variants[len(variants)-1] {
		t.Errorf("Say with last picker = %q, want %q", got, variants[len(variants)-1])
	}
}

func TestSayPickerOutOfRangeIsClamped(t *testing.T) {
	bad := func(n int) int { return n + 5 }
	if got := Say(MemeReply, config.ToneMild, bad); got == "" {
		t.Error("out-of-range picker should not produce empty line")
	}
	neg := func(int) int { return -1 }
	if got := Say(MemeReply, config.ToneMild, neg); got == "" {
		t.Error("negative picker should not produce empty line")
	}
}

func TestSayUnknownKeyOrTone(t *testing.T) {
	if got := Say(Key("nope"), config.ToneFull, first); got != "" {
		t.Errorf("unknown key should return empty, got %q", got)
	}
	// 모르는 톤이면 mild 로 대체한다 (안전한 쪽).
	got := Say(MemeReply, config.Tone("weird"), first)
	if got != lines[MemeReply][config.ToneMild][0] {
		t.Errorf("unknown tone should fall back to mild, got %q", got)
	}
}

func TestDie(t *testing.T) {
	for n := 1; n <= 6; n++ {
		if Die(n) == "" {
			t.Errorf("Die(%d) empty", n)
		}
	}
	if Die(1) == Die(2) {
		t.Error("dice faces must differ")
	}
	for _, n := range []int{0, 7, -1} {
		if Die(n) != "?" {
			t.Errorf("Die(%d) = %q, want ?", n, Die(n))
		}
	}
}
