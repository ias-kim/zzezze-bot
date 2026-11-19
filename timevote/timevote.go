// Package timevote 는 디스코드 기본 투표에 올릴 시간 선택지를 계산하는 순수 패키지다.
// 시각은 인자로 받으므로 sleep 없이 테스트한다.
package timevote

import (
	"time"
	_ "time/tzdata" // 배포 환경에 zoneinfo 가 없어도 Asia/Seoul 을 읽을 수 있게 내장한다.
)

// Duration 은 투표 마감까지의 시간이다.
const Duration = 3 * time.Hour

const (
	firstHour = 18 // 선택지 시작 시각(KST, 포함)
	lastHour  = 23 // 선택지 끝 시각(KST, 포함)
)

// kst 는 패키지 초기화 때 한 번 읽는다.
// 로드에 실패하면 panic 한다. tzdata 를 내장했으므로 실패는 곧 빌드·환경 이상이고,
// FixedZone 으로 조용히 대체하면 문제가 숨겨지기 때문이다.
var kst = mustLoadKST()

func mustLoadKST() *time.Location {
	loc, err := time.LoadLocation("Asia/Seoul")
	if err != nil {
		panic("timevote: Asia/Seoul 로드 실패: " + err.Error())
	}
	return loc
}

// Options 는 now 이후의 KST 정각 선택지를 오름차순으로 돌려준다.
// 오늘(KST) 18~23시 중 now 보다 뒤인 것만 담고, 같은 시각은 지난 것으로 본다.
// 모두 지났으면 내일 18~23시 6개를 돌려준다. now 의 시간대는 무엇이든 상관없다.
func Options(now time.Time) []time.Time {
	n := now.In(kst)
	day := time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, kst)

	if opts := hoursAfter(day, n); len(opts) > 0 {
		return opts
	}
	// time.Date 는 월말·연말 넘김을 정규화한다.
	next := time.Date(day.Year(), day.Month(), day.Day()+1, 0, 0, 0, 0, kst)
	return hoursAfter(next, n)
}

// hoursAfter 는 day(KST 자정) 날의 18~23시 정각 중 after 보다 엄격히 뒤인 것을 모은다.
func hoursAfter(day, after time.Time) []time.Time {
	var out []time.Time
	for h := firstHour; h <= lastHour; h++ {
		t := time.Date(day.Year(), day.Month(), day.Day(), h, 0, 0, 0, kst)
		if t.After(after) {
			out = append(out, t)
		}
	}
	return out
}
