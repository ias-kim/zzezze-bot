package features

import (
	"bufio"
	"os"
	"strings"
	"testing"
)

func readLines(t *testing.T, path string) []string {
	t.Helper()
	f, err := os.Open(path)
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
	return out
}

func TestNudgeHit(t *testing.T) {
	lines := readLines(t, "testdata/nudge_hit.txt")
	if len(lines) == 0 {
		t.Fatal("nudge_hit.txt 가 비어 있음")
	}
	for _, line := range lines {
		want, text, ok := strings.Cut(line, "\t")
		if !ok {
			t.Fatalf("형식 오류(탭 없음): %q", line)
		}
		if got := DetectNudge(text); string(got) != want {
			t.Errorf("DetectNudge(%q) = %q, want %q", text, got, want)
		}
	}
}

func TestNudgeMiss(t *testing.T) {
	for _, text := range readLines(t, "testdata/nudge_miss.txt") {
		if got := DetectNudge(text); got != NoNudge {
			t.Errorf("DetectNudge(%q) = %q, want none", text, got)
		}
	}
}

func TestIsMeme(t *testing.T) {
	tests := []struct {
		text string
		want bool
	}{
		{"쮀쮀이야", true},
		{"아 쮀쮀", true},
		{"쮀 쮀 이야", true},
		{"줘줘", false},
		{"졸졸 따라와", false},
		{"쮀", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := IsMeme(tt.text); got != tt.want {
			t.Errorf("IsMeme(%q) = %v, want %v", tt.text, got, tt.want)
		}
	}
}

func TestIgnoreMessage(t *testing.T) {
	tests := []struct {
		text string
		bot  bool
		want bool
	}{
		{"치킨 시킬까", false, false},
		{"치킨 시킬까", true, true},
		{"/야추", false, true},
		{"!help", false, true},
		{"   ", false, true},
	}
	for _, tt := range tests {
		if got := Ignore(tt.text, tt.bot); got != tt.want {
			t.Errorf("Ignore(%q, %v) = %v, want %v", tt.text, tt.bot, got, tt.want)
		}
	}
}
