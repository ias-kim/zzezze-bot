// Package features 는 디스코드 이벤트와 순수 로직(session, yacht, voice)을 잇는 연결 계층이다.
// discordgo 는 이 패키지와 main 만 import 한다 (scripts/check-layers.sh).
package features

import (
	"regexp"
	"strings"
)

// NudgeKind 는 대화 트리거 종류다. 쿨다운과 [오늘 그만] 도 이 단위로 센다.
type NudgeKind string

const (
	NoNudge      NudgeKind = ""
	NudgeChicken NudgeKind = "chicken"
	NudgeTime    NudgeKind = "time"
)

var (
	// "치킨"과 의도 표현이 함께 나올 때만 반응한다 (docs/design/six-member.md).
	chickenWord   = regexp.MustCompile(`치킨`)
	chickenIntent = regexp.MustCompile(`시킬|시켜|먹을|ㄱ|뭐먹|\?`)
	timeAsk       = regexp.MustCompile(`몇\s*시|언제\s*(봄|볼|만나)`)
	// "쮀쮀" 계열에만 반응한다. "줘줘", "졸졸"은 평범한 말에 오작동해서 뺀다.
	meme = regexp.MustCompile(`쮀\s*쮀`)
)

// DetectNudge 는 문장이 어떤 대화 트리거에 걸리는지 돌려준다. 치킨이 시간보다 먼저다.
func DetectNudge(text string) NudgeKind {
	switch {
	case chickenWord.MatchString(text) && chickenIntent.MatchString(text):
		return NudgeChicken
	case timeAsk.MatchString(text):
		return NudgeTime
	}
	return NoNudge
}

// IsMeme 은 쮀쮀이야 밈 자동 반응 대상인지 본다.
func IsMeme(text string) bool { return meme.MatchString(text) }

// Ignore 는 봇 메시지, 명령어 메시지, 빈 메시지를 거른다.
func Ignore(text string, fromBot bool) bool {
	if fromBot {
		return true
	}
	t := strings.TrimSpace(text)
	return t == "" || strings.HasPrefix(t, "/") || strings.HasPrefix(t, "!")
}
