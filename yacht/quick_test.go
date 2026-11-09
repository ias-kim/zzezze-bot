package yacht

import (
	"errors"
	"testing"
)

// seq 는 정해진 눈을 차례대로 내는 주사위다.
func seq(faces ...int) Roller {
	i := 0
	return func() int {
		f := faces[i%len(faces)]
		i++
		return f
	}
}

func five(n int) []int { return []int{n, n, n, n, n} }

func cat(xs ...[]int) []int {
	var out []int
	for _, x := range xs {
		out = append(out, x...)
	}
	return out
}

func noPick(int) int { panic("pick 이 불리면 안 됨") }

func TestQuickNeedsTwo(t *testing.T) {
	for _, users := range [][]string{nil, {"a"}} {
		if _, err := Quick(users, seq(1), noPick); !errors.Is(err, ErrNotEnoughPlayers) {
			t.Errorf("Quick(%v) err = %v", users, err)
		}
	}
}

func TestQuickLowestLoses(t *testing.T) {
	// a=30, b=5, c=15 → b 당첨
	res, err := Quick([]string{"a", "b", "c"}, seq(cat(five(6), five(1), five(3))...), noPick)
	if err != nil {
		t.Fatal(err)
	}
	if res.Loser != "b" || res.ByLot {
		t.Errorf("result = %+v", res)
	}
	if len(res.Rounds) != 1 || len(res.Rounds[0]) != 3 {
		t.Fatalf("rounds = %+v", res.Rounds)
	}
	if r := res.Rounds[0][0]; r.User != "a" || r.Sum != 30 || r.Dice != [5]int{6, 6, 6, 6, 6} {
		t.Errorf("a roll = %+v", r)
	}
}

// 최저 동점자끼리만 다시 굴린다.
func TestQuickTieRerollsOnlyTied(t *testing.T) {
	// 1라운드: a=10, b=10, c=30 → a,b 재굴림. 2라운드: a=25, b=20 → b
	faces := cat(five(2), five(2), five(6), five(5), five(4))
	res, err := Quick([]string{"a", "b", "c"}, seq(faces...), noPick)
	if err != nil {
		t.Fatal(err)
	}
	if res.Loser != "b" || len(res.Rounds) != 2 || len(res.Rounds[1]) != 2 {
		t.Errorf("result = %+v", res)
	}
}

// 첫 판 + 재굴림 3라운드까지 동점이면 제비뽑기.
func TestQuickTieBreaksByLot(t *testing.T) {
	var picked int
	pick := func(n int) int { picked = n; return 1 }
	res, err := Quick([]string{"a", "b"}, seq(3), pick)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Rounds) != 1+MaxRerolls {
		t.Errorf("rounds = %d, want %d", len(res.Rounds), 1+MaxRerolls)
	}
	if !res.ByLot || res.Loser != "b" || picked != 2 {
		t.Errorf("result = %+v picked=%d", res, picked)
	}
}

func TestQuickClampsBadRoller(t *testing.T) {
	res, err := Quick([]string{"a", "b"}, seq(0, 9, 9, 9, 9, 6, 6, 6, 6, 6), noPick)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range res.Rounds[0] {
		for _, d := range r.Dice {
			if d < 1 || d > 6 {
				t.Fatalf("주사위 범위 밖: %+v", r)
			}
		}
	}
}
