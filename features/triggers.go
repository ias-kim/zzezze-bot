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
	NoNudge   NudgeKind = ""
	NudgeFood NudgeKind = "food" // 배달 음식 (치킨, 곱창, 양꼬치 …) → 주사위로 누가 낼지 정하기
	NudgeTime NudgeKind = "time"
)

// Foods 는 반응하는 음식 이름이다. 새 메뉴가 생기면 여기에만 추가한다.
// 긴 이름을 먼저 둔다(정규식은 왼쪽부터 맞춰 보므로 "마라샹궈"가 "마라탕"보다, "떡볶이"가 짧은 말보다 앞).
// "회"처럼 한 글자는 "회사" 같은 평범한 말에 오작동하므로 넣지 않는다.
var Foods = []string{
	"마라샹궈", "양꼬치", "마라탕", "떡볶이", "탕수육", "삼겹살", "닭갈비", "쌀국수", "돈까스", "돈가스", "햄버거", "중국집",
	"치킨", "통닭", "피자", "곱창", "막창", "대창", "족발", "보쌈", "짜장", "짬뽕", "버거", "초밥", "스시",
	"국밥", "순대", "닭발", "찜닭", "라멘", "분식", "야식", "고기",
}

var (
	// 음식 이름과 의도 표현이 함께 나올 때만 반응한다 (docs/design/six-member.md).
	foodWord   = regexp.MustCompile(strings.Join(Foods, "|"))
	foodIntent = regexp.MustCompile(`시킬|시켜|먹을|먹자|ㄱ|뭐먹|\?`)
	// 메뉴 없이 "뭐 시킬까", "뭐먹지", "배달 시킬래" 같은 말.
	foodGeneric = regexp.MustCompile(`뭐\s*(시킬|시켜|먹)|배달\s*(시킬|시켜|ㄱ)`)
	timeAsk     = regexp.MustCompile(`몇\s*시|언제\s*(봄|볼|만나)`)
	// 쮀쮀이야 계열: "쮀쮀", "줴줴"(조줴줴이야 포함), "최재희". "줘줘", "졸졸", "줴~랄~"은 반응하지 않는다.
	meme = regexp.MustCompile(`쮀\s*쮀|줴\s*줴|최\s*재\s*희`)
)

// DetectNudge 는 문장이 어떤 대화 트리거에 걸리는지와, 음식 트리거면 음식 이름을 돌려준다.
// 메뉴 없이 "뭐 시킬까" 같은 말이면 food 는 빈 문자열이다. 음식이 시간보다 먼저다.
func DetectNudge(text string) (kind NudgeKind, food string) {
	if foodIntent.MatchString(text) {
		if f := foodWord.FindString(text); f != "" {
			return NudgeFood, f
		}
	}
	if foodGeneric.MatchString(text) {
		return NudgeFood, ""
	}
	if timeAsk.MatchString(text) {
		return NudgeTime, ""
	}
	return NoNudge, ""
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
