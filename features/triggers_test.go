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
		parts := strings.Split(line, "\t")
		if len(parts) != 3 {
			t.Fatalf("형식 오류(탭 2개 필요): %q", line)
		}
		wantKind, wantFood, text := parts[0], parts[1], parts[2]
		if wantFood == "-" {
			wantFood = ""
		}
		kind, food := DetectNudge(text)
		if string(kind) != wantKind || food != wantFood {
			t.Errorf("DetectNudge(%q) = (%q, %q), want (%q, %q)", text, kind, food, wantKind, wantFood)
		}
	}
}

func TestNudgeMiss(t *testing.T) {
	for _, text := range readLines(t, "testdata/nudge_miss.txt") {
		if kind, _ := DetectNudge(text); kind != NoNudge {
			t.Errorf("DetectNudge(%q) = %q, want none", text, kind)
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
		{"줴줴이야~", true},
		{"조줴줴이야~", true},
		{"최재희야~~", true},
		{"최 재희야", true},
		{"줴~랄~", false},
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
