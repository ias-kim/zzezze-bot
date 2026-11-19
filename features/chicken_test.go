package features

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"

	"zzezze-bot/config"
	"zzezze-bot/interactions"
	"zzezze-bot/session"
	"zzezze-bot/voice"
	"zzezze-bot/yacht"
)

var _ interface{ closeInfo() (string, config.Tone) } = (*chickenState)(nil)

func chickenFirst(n int) int { return 0 }

// chickenSeq 는 주어진 눈을 순서대로 돌려주는 Roll 이다.
func chickenSeq(vals ...int) func() int {
	i := 0
	return func() int {
		v := vals[i%len(vals)]
		i++
		return v
	}
}

func TestChickenJoinWindow(t *testing.T) {
	if ChickenJoinWindow != 60*time.Second {
		t.Fatalf("ChickenJoinWindow = %v, want 60s", ChickenJoinWindow)
	}
}

func TestChickenOpen_첫참가자는_누른사람(t *testing.T) {
	reg := session.New()
	text, err := chickenOpen(reg, "m1", "ch1", "u1", config.ToneFull, chickenFirst, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, voice.Say(voice.ChickenOpen, config.ToneFull, chickenFirst)) {
		t.Errorf("모집 화면에 ChickenOpen 대사가 없다: %q", text)
	}
	if !strings.Contains(text, mention("u1")) {
		t.Errorf("모집 화면에 참가자가 없다: %q", text)
	}
	if got, ok := reg.UserGame("u1"); !ok || got != "m1" {
		t.Errorf("UserGame = %q,%v", got, ok)
	}
	var st *chickenState
	_ = reg.Do("m1", func(s any) { st = s.(*chickenState) })
	ch, tone := st.closeInfo()
	if ch != "ch1" || tone != config.ToneFull {
		t.Errorf("closeInfo = %q,%q", ch, tone)
	}
}

func TestChickenOpen_다른판에_있으면_거절(t *testing.T) {
	reg := session.New()
	if err := reg.Start("other", session.Game{Idle: time.Minute, Users: []string{"u1"}}); err != nil {
		t.Fatal(err)
	}
	_, err := chickenOpen(reg, "m1", "ch1", "u1", config.ToneMild, chickenFirst, nil)
	if !errors.Is(err, session.ErrUserBusy) {
		t.Fatalf("err = %v, want ErrUserBusy", err)
	}
	if k := chickenRejectKey(err); k != voice.AlreadyInGame {
		t.Errorf("reject key = %v", k)
	}
}

func TestChickenOpen_이미_판인_메시지(t *testing.T) {
	reg := session.New()
	if _, err := chickenOpen(reg, "m1", "ch1", "u1", config.ToneMild, chickenFirst, nil); err != nil {
		t.Fatal(err)
	}
	_, err := chickenOpen(reg, "m1", "ch1", "u2", config.ToneMild, chickenFirst, nil)
	if !errors.Is(err, session.ErrExists) {
		t.Fatalf("err = %v, want ErrExists", err)
	}
	if k := chickenRejectKey(err); k != voice.AlreadyInGame {
		t.Errorf("reject key = %v", k)
	}
	if _, ok := reg.UserGame("u2"); ok {
		t.Error("거절된 유저가 등록됐다")
	}
}

func TestChickenJoin(t *testing.T) {
	reg := session.New()
	if _, err := chickenOpen(reg, "m1", "ch1", "u1", config.ToneFull, chickenFirst, nil); err != nil {
		t.Fatal(err)
	}
	text, err := chickenJoin(reg, "m1", "u2", chickenFirst)
	if err != nil {
		t.Fatal(err)
	}
	for _, u := range []string{"u1", "u2"} {
		if !strings.Contains(text, mention(u)) {
			t.Errorf("화면에 %s 가 없다: %q", u, text)
		}
	}
	if strings.Index(text, mention("u1")) > strings.Index(text, mention("u2")) {
		t.Errorf("참가 순서가 아니다: %q", text)
	}
	var players []string
	_ = reg.Do("m1", func(s any) { players = append(players, s.(*chickenState).players...) })
	if len(players) != 2 || players[0] != "u1" || players[1] != "u2" {
		t.Errorf("players = %v", players)
	}
}

func TestChickenJoin_에러_매핑(t *testing.T) {
	reg := session.New()
	if _, err := chickenOpen(reg, "m1", "ch1", "u1", config.ToneMild, chickenFirst, nil); err != nil {
		t.Fatal(err)
	}
	if err := reg.Start("other", session.Game{Idle: time.Minute, Users: []string{"u3"}}); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, msg, user string
		wantErr         error
		wantKey         voice.Key
	}{
		{"만료", "none", "u2", session.ErrNotFound, voice.SessionExpired},
		{"중복", "m1", "u1", session.ErrAlreadyJoined, voice.AlreadyInGame},
		{"다른 판", "m1", "u3", session.ErrUserBusy, voice.AlreadyInGame},
	}
	for _, c := range cases {
		_, err := chickenJoin(reg, c.msg, c.user, chickenFirst)
		if !errors.Is(err, c.wantErr) {
			t.Errorf("%s: err = %v, want %v", c.name, err, c.wantErr)
		}
		if k := chickenRejectKey(err); k != c.wantKey {
			t.Errorf("%s: key = %v, want %v", c.name, k, c.wantKey)
		}
	}
	var n int
	_ = reg.Do("m1", func(s any) { n = len(s.(*chickenState).players) })
	if n != 1 {
		t.Errorf("거절 뒤 참가자 수 = %d, want 1", n)
	}
}

func TestChickenResultText_재굴림없음(t *testing.T) {
	res := yacht.ChickenResult{
		Rounds: []yacht.Round{{Rolls: []yacht.PlayerRoll{
			{Player: "a", Dice: [5]int{1, 2, 3, 4, 5}, Sum: 15},
			{Player: "b", Dice: [5]int{6, 6, 6, 6, 6}, Sum: 30},
		}}},
		Loser: "a",
	}
	text := chickenResultText(res, config.ToneFull, chickenFirst)
	lines := strings.Split(text, "\n")
	if len(lines) != 3 {
		t.Fatalf("줄 수 = %d, want 3 (참가자 2 + 당첨): %q", len(lines), text)
	}
	wantA := mention("a") + " " + voice.Die(1) + voice.Die(2) + voice.Die(3) + voice.Die(4) + voice.Die(5) + " = 15"
	if lines[0] != wantA {
		t.Errorf("line0 = %q, want %q", lines[0], wantA)
	}
	if !strings.HasSuffix(lines[1], " = 30") || !strings.HasPrefix(lines[1], mention("b")) {
		t.Errorf("line1 = %q", lines[1])
	}
	wantLoser := voice.Say(voice.ChickenLoser, config.ToneFull, chickenFirst, "{user}", mention("a"))
	if lines[2] != wantLoser {
		t.Errorf("line2 = %q, want %q", lines[2], wantLoser)
	}
	tie := voice.Say(voice.ChickenTie, config.ToneFull, chickenFirst)
	if strings.Contains(text, tie) {
		t.Error("첫 라운드에는 동점 문구가 없어야 한다")
	}
}

func TestChickenResultText_재굴림_라운드는_동점문구가_앞에(t *testing.T) {
	res := yacht.ChickenResult{
		Rounds: []yacht.Round{
			{Rolls: []yacht.PlayerRoll{
				{Player: "a", Dice: [5]int{1, 1, 1, 1, 1}, Sum: 5},
				{Player: "b", Dice: [5]int{1, 1, 1, 1, 1}, Sum: 5},
				{Player: "c", Dice: [5]int{6, 6, 6, 6, 6}, Sum: 30},
			}},
			{Rolls: []yacht.PlayerRoll{
				{Player: "a", Dice: [5]int{2, 2, 2, 2, 2}, Sum: 10},
				{Player: "b", Dice: [5]int{3, 3, 3, 3, 3}, Sum: 15},
			}},
		},
		Loser: "a",
	}
	text := chickenResultText(res, config.ToneMild, chickenFirst)
	lines := strings.Split(text, "\n")
	// 3 + (동점 1 + 2) + 당첨 1
	if len(lines) != 7 {
		t.Fatalf("줄 수 = %d, want 7: %q", len(lines), text)
	}
	tie := voice.Say(voice.ChickenTie, config.ToneMild, chickenFirst)
	if lines[3] != tie {
		t.Errorf("line3 = %q, want 동점 문구 %q", lines[3], tie)
	}
	if !strings.HasPrefix(lines[4], mention("a")) || !strings.HasPrefix(lines[5], mention("b")) {
		t.Errorf("재굴림 라운드 줄이 이상하다: %q", lines[4:6])
	}
	if strings.Count(text, tie) != 1 {
		t.Errorf("동점 문구 횟수 = %d, want 1", strings.Count(text, tie))
	}
}

func TestChickenSettle_2명미만은_취소(t *testing.T) {
	got := chickenSettle([]string{"a"}, config.ToneFull, chickenSeq(3), chickenFirst)
	want := voice.Say(voice.ChickenCancel, config.ToneFull, chickenFirst)
	if got != want {
		t.Errorf("settle = %q, want %q", got, want)
	}
}

func TestChickenSettle_정상(t *testing.T) {
	// a 는 1 만, b 는 6 만 나오는 Roll: 호출 순서 a×5, b×5
	roll := chickenSeq(1, 1, 1, 1, 1, 6, 6, 6, 6, 6)
	got := chickenSettle([]string{"a", "b"}, config.ToneMild, roll, chickenFirst)
	if !strings.Contains(got, voice.Say(voice.ChickenLoser, config.ToneMild, chickenFirst, "{user}", mention("a"))) {
		t.Errorf("a 가 당첨이어야 한다: %q", got)
	}
}

func TestChickenJoinRow(t *testing.T) {
	row := chickenJoinRow(config.ToneFull, chickenFirst)
	if len(row) != 1 {
		t.Fatalf("row 수 = %d", len(row))
	}
	ar, ok := row[0].(discordgo.ActionsRow)
	if !ok || len(ar.Components) != 1 {
		t.Fatalf("row = %#v", row[0])
	}
	btn, ok := ar.Components[0].(discordgo.Button)
	if !ok {
		t.Fatalf("component = %#v", ar.Components[0])
	}
	id, err := interactions.Decode(btn.CustomID)
	if err != nil || id.Feature != "chicken" || id.Action != "join" {
		t.Errorf("custom_id = %q (%v)", btn.CustomID, err)
	}
	if btn.Label != voice.Say(voice.JoinBtnLabel, config.ToneFull, chickenFirst) {
		t.Errorf("label = %q", btn.Label)
	}
}

// 같은 제안 메시지의 [야추로 정하기]를 여러 명이 거의 동시에 누르면, 첫 번째는 판을 열고 나머지는 참가한다.
func TestChickenOpenOrJoin_동시클릭은_참가로(t *testing.T) {
	reg := session.New()
	if _, err := chickenOpenOrJoin(reg, "m1", "ch1", "u1", config.ToneMild, chickenFirst, nil); err != nil {
		t.Fatalf("첫 클릭: %v", err)
	}
	text, err := chickenOpenOrJoin(reg, "m1", "ch1", "u2", config.ToneMild, chickenFirst, nil)
	if err != nil {
		t.Fatalf("두 번째 클릭은 참가여야 한다: %v", err)
	}
	if !strings.Contains(text, mention("u2")) {
		t.Errorf("모집 화면에 u2 가 없다: %q", text)
	}
	if _, err := chickenOpenOrJoin(reg, "m1", "ch1", "u1", config.ToneMild, chickenFirst, nil); !errors.Is(err, session.ErrAlreadyJoined) {
		t.Errorf("이미 참가한 사람의 재클릭 = %v, want ErrAlreadyJoined", err)
	}
	reg.CloseAll()
}
