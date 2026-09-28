package interactions

import (
	"strings"
	"testing"
)

func TestEncodeDecodeRoundTrip(t *testing.T) {
	tests := []ID{
		{Feature: "yacht", Action: "accept"},
		{Feature: "yacht", Action: "hold", Arg: "3"},
		{Feature: "nudge", Action: "chicken", Arg: "join"},
		{Feature: "nudge", Action: "mute"},
	}
	for _, id := range tests {
		s, err := Encode(id)
		if err != nil {
			t.Fatalf("Encode(%+v) error: %v", id, err)
		}
		got, err := Decode(s)
		if err != nil {
			t.Fatalf("Decode(%q) error: %v", s, err)
		}
		if got != id {
			t.Errorf("round trip %+v -> %q -> %+v", id, s, got)
		}
	}
}

func TestEncodeFormat(t *testing.T) {
	s, err := Encode(ID{Feature: "yacht", Action: "hold", Arg: "3"})
	if err != nil {
		t.Fatal(err)
	}
	if s != "yacht:hold:3" {
		t.Errorf("Encode = %q, want %q", s, "yacht:hold:3")
	}
	s, _ = Encode(ID{Feature: "yacht", Action: "accept"})
	if s != "yacht:accept" {
		t.Errorf("Encode without arg = %q, want %q", s, "yacht:accept")
	}
}

func TestEncodeRejects(t *testing.T) {
	tests := []struct {
		name string
		id   ID
	}{
		{"빈 기능", ID{Action: "x"}},
		{"빈 동작", ID{Feature: "yacht"}},
		{"기능에 구분자", ID{Feature: "ya:cht", Action: "x"}},
		{"동작에 구분자", ID{Feature: "yacht", Action: "a:b"}},
		{"인자에 구분자", ID{Feature: "yacht", Action: "hold", Arg: "1:2"}},
		{"100자 초과", ID{Feature: "yacht", Action: "hold", Arg: strings.Repeat("x", 100)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := Encode(tt.id); err == nil {
				t.Errorf("Encode(%+v) should fail", tt.id)
			}
		})
	}
}

func TestEncodeExactly100(t *testing.T) {
	// "yacht:hold:" = 11자 + 89자 = 100자 → 허용
	id := ID{Feature: "yacht", Action: "hold", Arg: strings.Repeat("x", 89)}
	if _, err := Encode(id); err != nil {
		t.Errorf("100자는 허용되어야 한다: %v", err)
	}
}

func TestDecodeRejects(t *testing.T) {
	tests := []string{
		"",
		"yacht",
		":accept",
		"yacht:",
		"yacht::3",
		"yacht:hold:",
		"a:b:c:d",
		strings.Repeat("x", 101),
	}
	for _, s := range tests {
		if _, err := Decode(s); err == nil {
			t.Errorf("Decode(%q) should fail", s)
		}
	}
}
