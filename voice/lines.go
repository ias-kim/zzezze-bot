// Package voice 는 봇이 하는 모든 말을 한 곳에 모은 대사집이다.
//
// 로직 코드에는 한국어 문자열을 직접 쓰지 않고 Say(키, 톤, ...) 로만 가져온다.
// 새 밈이 생기면 이 파일의 lines 표만 고치면 된다.
//
//	full: 원정 멤버 전용 방(2번방). 방 밈 말투 풀파워.
//	mild: 비핵심 인원이 있는 방(1번방). 가벼운 밈만.
package voice

import (
	"slices"
	"strings"

	"zzezze-bot/config"
)

// Key 는 대사 상황 이름이다.
type Key string

const (
	MemeReply       Key = "meme.reply"       // 쮀쮀이야 밈에 자동 반응
	NudgeChicken    Key = "nudge.chicken"    // "치킨 시킬까?" → 야추로 정하기 제안
	NudgeTime       Key = "nudge.time"       // "몇시?" → 시간 투표 제안
	NudgeMuted      Key = "nudge.muted"      // [오늘 그만] 눌렀을 때
	ChickenOpen     Key = "chicken.open"     // 참가 모집 시작
	ChickenJoined   Key = "chicken.joined"   // {user} 참가
	ChickenCancel   Key = "chicken.cancel"   // 참가자 부족으로 취소
	ChickenTie      Key = "chicken.tie"      // 동점 재굴림
	ChickenLoser    Key = "chicken.loser"    // {user} 당첨 (치킨값 담당)
	TimePollTitle   Key = "time.poll.title"  // 시간 투표 질문
	SessionExpired  Key = "session.expired"  // 만료된 버튼을 눌렀을 때
	SessionRestart  Key = "session.restart"  // 봇 재시작으로 판 취소
	NotYourButton   Key = "button.notyours"  // 남의 판 버튼을 눌렀을 때
	WrongChannel    Key = "channel.wrong"    // 기능이 꺼진 채널에서 커맨드 사용
	AlreadyInGame   Key = "session.busy"     // 이미 진행 중인 판이 있을 때
	ChickenBtnLabel Key = "chicken.btn.play" // [🎲 야추로 정하기] 버튼 라벨
	TimeBtnLabel    Key = "time.btn.poll"    // [⏰ 시간 투표] 버튼 라벨
	MuteBtnLabel    Key = "nudge.btn.mute"   // [🔕 오늘 그만] 버튼 라벨
	JoinBtnLabel    Key = "chicken.btn.join" // [🙋 참가] 버튼 라벨
)

// keys 는 선언된 모든 대사 키 목록이다. 새 키를 만들면 여기에도 추가한다.
// 테스트(TestKeyListMatchesTable)가 이 목록과 lines 표가 같은 집합인지 확인한다.
var keys = []Key{
	MemeReply, NudgeChicken, NudgeTime, NudgeMuted,
	ChickenOpen, ChickenJoined, ChickenCancel, ChickenTie, ChickenLoser,
	TimePollTitle, SessionExpired, SessionRestart, NotYourButton, WrongChannel, AlreadyInGame,
	ChickenBtnLabel, TimeBtnLabel, MuteBtnLabel, JoinBtnLabel,
}

type toneLines map[config.Tone][]string

var lines = map[Key]toneLines{
	MemeReply: {
		config.ToneFull: {"쮀쮀이야 티~원~ 🙌", "아.. 예?.. 쮀쮀이야~", "쮀쮀쮀쮀 쮀쮀이야!!", "줴~랄~ 쮀쮀이야 ㅋㅋ"},
		config.ToneMild: {"쮀쮀이야! 🙌", "쮀쮀이야~"},
	},
	NudgeChicken: {
		config.ToneFull: {"아.. 예?.. 치킨이요? 줴~랄~ 누가 쏠지 야추로 정하시죠 🎲", "치킨 얘기 나왔다 쮀쮀이야~ 꼴찌가 쏘는 걸로 ㅋㅋ"},
		config.ToneMild: {"치킨 누가 살지 주사위로 정해볼까요? 🎲"},
	},
	NudgeTime: {
		config.ToneFull: {"아.. 예?.. 몇시냐고요? 투표로 정하시죠 줴~랄~ ⏰", "또 시간 못 정하고 있네 ㅋㅋ 투표 ㄱㄱ ⏰"},
		config.ToneMild: {"시간 투표 올려드릴까요? ⏰"},
	},
	NudgeMuted: {
		config.ToneFull: {"아.. 예.. 오늘은 조용히 있겠습니다 🤐", "넵 오늘은 입 닫습니다 쮀쮀.."},
		config.ToneMild: {"오늘은 조용히 있을게요 🤐"},
	},
	ChickenOpen: {
		config.ToneFull: {"🍗 치킨값 걸린 한 판! 60초 안에 [참가] 누르세요. 꼴찌가 쏩니다 줴~랄~", "🍗 쫄리면 뒤지시든가~ 60초 안에 참가 ㄱㄱ"},
		config.ToneMild: {"🍗 주사위 한 판! 60초 안에 [참가]를 눌러주세요. 최저점이 당첨!"},
	},
	ChickenJoined: {
		config.ToneFull: {"{user} 참가 ㅋㅋ 쮀쮀이야~", "{user} 들어왔다 아.. 예?.."},
		config.ToneMild: {"{user} 참가!"},
	},
	ChickenCancel: {
		config.ToneFull: {"아.. 예?.. 혼자 하실 거예요? 줴~랄~ 취소입니다", "참가자 부족 ㅋㅋ 다들 쫄았네 취소~"},
		config.ToneMild: {"참가자가 부족해서 취소됐어요."},
	},
	ChickenTie: {
		config.ToneFull: {"동점?? 줴~랄~ 한 번 더 굴립니다 🎲", "아.. 예?.. 동점이네요 재대결 ㄱㄱ"},
		config.ToneMild: {"동점! 한 번 더 굴립니다 🎲"},
	},
	ChickenLoser: {
		config.ToneFull: {"🍗 오늘의 치킨 스폰서: {user} ㅋㅋㅋ 쮀쮀이야 티~원~", "아.. 예?.. {user} 님이 쏘신답니다 줴~랄~ 🍗"},
		config.ToneMild: {"🍗 당첨: {user}! 잘 먹겠습니다 ㅎㅎ"},
	},
	TimePollTitle: {
		config.ToneFull: {"몇시에 봄? 되는 시간 다 찍어라 (복수 선택)", "아.. 예?.. 몇시요? 가능한 시간 다 고르세요"},
		config.ToneMild: {"가능한 시간을 모두 골라주세요"},
	},
	SessionExpired: {
		config.ToneFull: {"아.. 예?.. 이미 끝난 판입니다 줴~랄~"},
		config.ToneMild: {"이미 끝난 판이에요."},
	},
	SessionRestart: {
		config.ToneFull: {"⚠️ 봇 재시작으로 판 취소 (포인트 이동 없음) 아.. 예.."},
		config.ToneMild: {"⚠️ 봇 재시작으로 취소됐어요 (포인트 이동 없음)."},
	},
	NotYourButton: {
		config.ToneFull: {"아.. 예?.. 님 판 아닌데요 줴~랄~"},
		config.ToneMild: {"이 버튼은 다른 분 차례예요."},
	},
	WrongChannel: {
		config.ToneFull: {"여기선 안 됩니다 아.. 예.."},
		config.ToneMild: {"이 방에서는 쓸 수 없어요."},
	},
	AlreadyInGame: {
		config.ToneFull: {"이미 판 하나 하고 있잖아요 줴~랄~"},
		config.ToneMild: {"이미 진행 중인 판이 있어요."},
	},
	ChickenBtnLabel: {
		config.ToneFull: {"야추로 정하기"},
		config.ToneMild: {"주사위로 정하기"},
	},
	TimeBtnLabel: {
		config.ToneFull: {"시간 투표"},
		config.ToneMild: {"시간 투표"},
	},
	MuteBtnLabel: {
		config.ToneFull: {"오늘 그만"},
		config.ToneMild: {"오늘 그만"},
	},
	JoinBtnLabel: {
		config.ToneFull: {"참가"},
		config.ToneMild: {"참가"},
	},
}

// AllKeys 는 선언된 모든 대사 키를 반환한다 (테스트와 점검용).
func AllKeys() []Key {
	return slices.Clone(keys)
}

// Picker 는 변형 n개 중 하나의 인덱스를 고른다. 운영에서는 rand.IntN, 테스트에서는 고정값.
type Picker func(n int) int

// Say 는 키와 톤에 맞는 대사 하나를 골라 치환자({user} 등)를 채워 돌려준다.
// repl 은 strings.NewReplacer 형식의 (old, new) 쌍이다.
// 모르는 키면 빈 문자열, 모르는 톤이면 mild 로 대체한다.
func Say(k Key, tone config.Tone, pick Picker, repl ...string) string {
	tl, ok := lines[k]
	if !ok {
		return ""
	}
	variants := tl[tone]
	if len(variants) == 0 {
		variants = tl[config.ToneMild]
	}
	if len(variants) == 0 {
		return ""
	}
	i := pick(len(variants))
	if i < 0 || i >= len(variants) {
		i = 0
	}
	s := variants[i]
	if len(repl) >= 2 {
		s = strings.NewReplacer(repl...).Replace(s)
	}
	return s
}

// 주사위 눈 텍스트. 서버 커스텀 이모지를 올리면 버튼용 이모지는 2차에서 추가한다.
var dieFaces = [...]string{"⚀", "⚁", "⚂", "⚃", "⚄", "⚅"}

// Die 는 1~6 주사위 눈을 문자로 돌려준다. 범위 밖이면 "?".
func Die(n int) string {
	if n < 1 || n > 6 {
		return "?"
	}
	return dieFaces[n-1]
}
