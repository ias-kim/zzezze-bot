// Package yacht 는 주사위 게임 규칙을 담는 순수 패키지다 (discordgo 를 모른다).
// 무작위성은 Roller, Pick 으로 주입받아 테스트에서 고정한다.
package yacht

import "errors"

// Roller 는 주사위 하나를 굴려 1~6 을 돌려준다. 운영에서는 rand, 테스트에서는 고정 수열.
type Roller func() int

// Pick 은 n 명 중 한 명의 인덱스(0~n-1)를 고른다.
type Pick func(n int) int

// MaxRerolls 는 최저점 동점일 때 다시 굴리는 최대 라운드 수다. 그래도 같으면 제비뽑기.
const MaxRerolls = 3

var ErrNotEnoughPlayers = errors.New("yacht: quick game needs at least 2 players")

// Roll 은 한 사람이 한 라운드에 굴린 결과다.
type Roll struct {
	User string
	Dice [5]int
	Sum  int
}

// QuickResult 는 빠른 야추 판("야추겜 ㄱㄱ") 전체 결과다. Rounds[0] 은 전원, 그 뒤는 최저 동점자만 굴린 기록.
type QuickResult struct {
	Rounds [][]Roll
	Loser  string // 꼴찌 (벌칙은 친구들끼리 정한다)
	ByLot  bool   // 재굴림으로도 못 가려 제비뽑기로 정했으면 true
}

// Quick 은 참가자마다 주사위 5개를 굴려 합계가 가장 낮은 사람을 고른다.
// 최저점 동점이면 동점자끼리 MaxRerolls 라운드까지 다시 굴리고, 그래도 같으면 pick 으로 정한다.
func Quick(users []string, roll Roller, pick Pick) (QuickResult, error) {
	if len(users) < 2 {
		return QuickResult{}, ErrNotEnoughPlayers
	}
	var res QuickResult
	alive := users
	for round := 0; round <= MaxRerolls; round++ {
		rolls := make([]Roll, len(alive))
		for i, u := range alive {
			rolls[i] = rollFive(u, roll)
		}
		res.Rounds = append(res.Rounds, rolls)
		alive = lowest(rolls)
		if len(alive) == 1 {
			res.Loser = alive[0]
			return res, nil
		}
	}
	i := pick(len(alive))
	if i < 0 || i >= len(alive) {
		i = 0
	}
	res.Loser, res.ByLot = alive[i], true
	return res, nil
}

func rollFive(user string, roll Roller) Roll {
	r := Roll{User: user}
	for i := range r.Dice {
		r.Dice[i] = min(max(roll(), 1), 6)
		r.Sum += r.Dice[i]
	}
	return r
}

func lowest(rolls []Roll) []string {
	low := rolls[0].Sum
	for _, r := range rolls[1:] {
		low = min(low, r.Sum)
	}
	var out []string
	for _, r := range rolls {
		if r.Sum == low {
			out = append(out, r.User)
		}
	}
	return out
}
