package yacht

import (
	"errors"
	"reflect"
	"testing"
)

// spread 는 합이 sum(5~30)인 주사위 5개를 만든다.
func spread(t *testing.T, sum int) []int {
	t.Helper()
	if sum < DiceCount || sum > DiceCount*6 {
		t.Fatalf("만들 수 없는 합계: %d", sum)
	}
	d := make([]int, DiceCount)
	for i := range d {
		d[i] = 1
	}
	rest := sum - DiceCount
	for i := 0; rest > 0; i = (i + 1) % DiceCount {
		if d[i] < 6 {
			d[i]++
			rest--
		}
	}
	return d
}

// scripted 는 라운드·참가자 순서대로 합계를 적어두면 그에 맞는 주사위를 하나씩 내주는 roll 을 만든다.
func scripted(t *testing.T, sums ...int) func() int {
	t.Helper()
	var seq []int
	for _, s := range sums {
		seq = append(seq, spread(t, s)...)
	}
	i := 0
	return func() int {
		if i >= len(seq) {
			t.Fatalf("준비된 주사위(%d개)보다 더 굴렸다", len(seq))
		}
		v := seq[i]
		i++
		return v
	}
}

func pickFirst(n int) int { return 0 }

func neverPick(t *testing.T) func(int) int {
	return func(n int) int {
		t.Errorf("pick 이 불리면 안 된다 (n=%d)", n)
		return 0
	}
}

func TestPlayChicken(t *testing.T) {
	players := []string{"a", "b", "c"}
	tests := []struct {
		name       string
		sums       []int // 라운드·참가자 순서
		wantLoser  string
		wantRounds int
		wantByPick bool
		pickIdx    int
	}{
		{"1라운드 단독 최저", []int{20, 10, 15}, "b", 1, false, 0},
		{"1라운드 첫 사람 최저", []int{7, 10, 15}, "a", 1, false, 0},
		{"동점 1번 재굴림 후 결판(a,c 동점)", []int{9, 20, 9, 12, 8}, "c", 2, false, 0},
		{"동점 2번 재굴림 후 결판", []int{9, 9, 9, 15, 10, 10, 5, 20}, "b", 3, false, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r, err := PlayChicken(players, scripted(t, tc.sums...), neverPick(t))
			if err != nil {
				t.Fatal(err)
			}
			if r.Loser != tc.wantLoser {
				t.Errorf("Loser = %q, want %q", r.Loser, tc.wantLoser)
			}
			if len(r.Rounds) != tc.wantRounds {
				t.Errorf("라운드 수 = %d, want %d", len(r.Rounds), tc.wantRounds)
			}
			if r.ByPick {
				t.Error("ByPick 이 true 면 안 된다")
			}
		})
	}
}

func TestPlayChickenTieRoundsOnlyTied(t *testing.T) {
	// 1R: a=9 b=20 c=9 → a,c 동점. 2R: 동점자 a,c 만 굴림 a=12 c=8 → c 당첨.
	r, err := PlayChicken([]string{"a", "b", "c"}, scripted(t, 9, 20, 9, 12, 8), neverPick(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Rounds) != 2 {
		t.Fatalf("라운드 수 = %d, want 2", len(r.Rounds))
	}
	r1, r2 := r.Rounds[0], r.Rounds[1]
	if len(r1.Rolls) != 3 || len(r2.Rolls) != 2 {
		t.Fatalf("참가자 수 = %d/%d, want 3/2", len(r1.Rolls), len(r2.Rolls))
	}
	if r2.Rolls[0].Player != "a" || r2.Rolls[1].Player != "c" {
		t.Errorf("재굴림 참가자 = %v, want a,c", r2.Rolls)
	}
	// 화면 렌더용: 주사위 5개와 합계가 남아 있어야 한다.
	if got := r1.Rolls[1]; got.Player != "b" || got.Sum != 20 || !reflect.DeepEqual(got.Dice[:], spread(t, 20)) {
		t.Errorf("1라운드 b = %+v", got)
	}
	if len(r2.Rolls[1].Dice) != DiceCount || r2.Rolls[1].Sum != 8 {
		t.Errorf("2라운드 c = %+v", r2.Rolls[1])
	}
}

func TestPlayChickenSumMatchesDice(t *testing.T) {
	r, err := PlayChicken([]string{"a", "b"}, scripted(t, 17, 23), neverPick(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, pr := range r.Rounds[0].Rolls {
		sum := 0
		for _, d := range pr.Dice {
			sum += d
		}
		if sum != pr.Sum {
			t.Errorf("%s: Sum=%d, 주사위 합=%d", pr.Player, pr.Sum, sum)
		}
	}
}

func TestPlayChickenThirdRerollDecides(t *testing.T) {
	// 첫 굴림 + 재굴림 3번 = 4라운드. 마지막(3번째 재굴림)에서 결판나면 pick 없음.
	// 1R a=b=10(c=20) → 2R a=b=11 → 3R a=b=12 → 4R a=13 b=9 → b 당첨.
	r, err := PlayChicken([]string{"a", "b", "c"},
		scripted(t, 10, 10, 20, 11, 11, 12, 12, 13, 9), neverPick(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Rounds) != 4 {
		t.Fatalf("라운드 수 = %d, want 4", len(r.Rounds))
	}
	if r.Loser != "b" || r.ByPick {
		t.Errorf("Loser=%q ByPick=%v, want b/false", r.Loser, r.ByPick)
	}
}

func TestPlayChickenPickAfterThreeRerolls(t *testing.T) {
	// 4라운드 모두 a,b 동점 → pick 으로 결정.
	var gotN int
	pick := func(n int) int { gotN = n; return 1 }
	r, err := PlayChicken([]string{"a", "b", "c"},
		scripted(t, 10, 10, 20, 11, 11, 12, 12, 13, 13), pick)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Rounds) != 4 {
		t.Fatalf("라운드 수 = %d, want 4 (재굴림 최대 3번)", len(r.Rounds))
	}
	if gotN != 2 {
		t.Errorf("pick 의 n = %d, want 2 (동점자 수)", gotN)
	}
	if r.Loser != "b" || !r.ByPick {
		t.Errorf("Loser=%q ByPick=%v, want b/true", r.Loser, r.ByPick)
	}
}

func TestPlayChickenPickAllTied(t *testing.T) {
	// 전원 매 라운드 동점 → 동점자 전체(3명) 중 pick.
	r, err := PlayChicken([]string{"a", "b", "c"},
		scripted(t, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10),
		func(n int) int {
			if n != 3 {
				t.Errorf("pick n = %d, want 3", n)
			}
			return 2
		})
	if err != nil {
		t.Fatal(err)
	}
	if r.Loser != "c" || !r.ByPick || len(r.Rounds) != 4 {
		t.Errorf("결과 = %+v", r)
	}
}

func TestPlayChickenInvalidPick(t *testing.T) {
	_, err := PlayChicken([]string{"a", "b"},
		scripted(t, 10, 10, 10, 10, 10, 10, 10, 10),
		func(n int) int { return n }) // 범위 밖
	if !errors.Is(err, ErrInvalidPick) {
		t.Errorf("err = %v, want ErrInvalidPick", err)
	}
}

func TestPlayChickenPlayerErrors(t *testing.T) {
	tests := []struct {
		name    string
		players []string
		want    error
	}{
		{"nil", nil, ErrNotEnoughPlayers},
		{"0명", []string{}, ErrNotEnoughPlayers},
		{"1명", []string{"a"}, ErrNotEnoughPlayers},
		{"중복", []string{"a", "b", "a"}, ErrDuplicatePlayer},
		{"2명 모두 같은 사람", []string{"a", "a"}, ErrDuplicatePlayer},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rolled := false
			roll := func() int { rolled = true; return 1 }
			_, err := PlayChicken(tc.players, roll, pickFirst)
			if !errors.Is(err, tc.want) {
				t.Errorf("err = %v, want %v", err, tc.want)
			}
			if rolled {
				t.Error("에러인데 주사위를 굴렸다")
			}
		})
	}
}

func TestPlayChickenDoesNotMutateInput(t *testing.T) {
	players := []string{"a", "b", "c"}
	_, err := PlayChicken(players, scripted(t, 20, 10, 15), neverPick(t))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(players, []string{"a", "b", "c"}) {
		t.Errorf("입력 슬라이스가 바뀌었다: %v", players)
	}
}
