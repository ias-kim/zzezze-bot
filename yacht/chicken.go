// Package yacht 는 야추 게임의 순수 규칙이다. 디스코드·저장소를 모르고, 난수는 주입받는다.
package yacht

import "errors"

const (
	// DiceCount 는 한 사람이 한 번에 굴리는 주사위 수다.
	DiceCount = 5
	// MaxRerolls 는 첫 굴림 뒤 동점 때문에 다시 굴리는 최대 횟수다.
	MaxRerolls = 3
)

var (
	ErrNotEnoughPlayers = errors.New("yacht: 참가자는 2명 이상이어야 한다")
	ErrDuplicatePlayer  = errors.New("yacht: 중복 참가자가 있다")
	ErrInvalidPick      = errors.New("yacht: pick 이 범위 밖 값을 돌려줬다")
)

// PlayerRoll 은 한 사람의 한 라운드 굴림이다.
type PlayerRoll struct {
	Player string
	Dice   [DiceCount]int
	Sum    int
}

// Round 는 한 라운드의 굴림 모음이다(참가 순서대로).
type Round struct {
	Rolls []PlayerRoll
}

// ChickenResult 는 치킨 판 결과다. 화면은 Rounds 만으로 한 번에 그릴 수 있다.
type ChickenResult struct {
	Rounds []Round
	Loser  string // 합계가 가장 낮아 당첨(치킨값 내는 사람)
	ByPick bool   // 재굴림 한도 후에도 동점이라 pick 으로 정했는지
}

// PlayChicken 은 참가자마다 주사위 5개를 굴려 합계가 가장 낮은 사람을 정한다.
// 최저점 동점이면 동점자끼리만 다시 굴리고(최대 MaxRerolls번), 그래도 같으면 pick 으로 고른다.
// roll 은 1~6, pick(n) 은 0..n-1 을 돌려줘야 한다.
func PlayChicken(players []string, roll func() int, pick func(n int) int) (ChickenResult, error) {
	if len(players) < 2 {
		return ChickenResult{}, ErrNotEnoughPlayers
	}
	seen := make(map[string]struct{}, len(players))
	for _, p := range players {
		if _, dup := seen[p]; dup {
			return ChickenResult{}, ErrDuplicatePlayer
		}
		seen[p] = struct{}{}
	}

	var res ChickenResult
	current := append([]string(nil), players...) // 입력을 건드리지 않는다
	for round := 0; round <= MaxRerolls; round++ {
		rolls := make([]PlayerRoll, 0, len(current))
		for _, p := range current {
			pr := PlayerRoll{Player: p}
			for i := range pr.Dice {
				pr.Dice[i] = roll()
				pr.Sum += pr.Dice[i]
			}
			rolls = append(rolls, pr)
		}
		res.Rounds = append(res.Rounds, Round{Rolls: rolls})

		current = lowest(rolls)
		if len(current) == 1 {
			res.Loser = current[0]
			return res, nil
		}
	}

	i := pick(len(current))
	if i < 0 || i >= len(current) {
		return ChickenResult{}, ErrInvalidPick
	}
	res.Loser = current[i]
	res.ByPick = true
	return res, nil
}

// lowest 는 합계가 가장 낮은 사람들을 입력 순서대로 돌려준다.
func lowest(rolls []PlayerRoll) []string {
	min := rolls[0].Sum
	for _, r := range rolls[1:] {
		if r.Sum < min {
			min = r.Sum
		}
	}
	var out []string
	for _, r := range rolls {
		if r.Sum == min {
			out = append(out, r.Player)
		}
	}
	return out
}
