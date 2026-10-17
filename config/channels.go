package config

import "slices"

// Feature 는 채널에서 켤 수 있는 기능 이름이다.
// 문자열 대신 상수를 쓰면 오타가 컴파일 에러가 된다.
type Feature string

const (
	AutoReply Feature = "autoReply" // 쮀쮀이야 계열 밈 자동 반응
	Yacht     Feature = "yacht"     // /야추, /포인트, /랭킹
	Nudge     Feature = "nudge"     // 약속 대화 트리거 → 약속 카드 (2번방 전용)
)

// Tone 은 봇 대사의 말투 강도다.
type Tone string

const (
	ToneMild Tone = "mild" // 가벼운 밈만 (비핵심 인원이 있는 방)
	ToneFull Tone = "full" // 방 밈 말투 풀파워 (원정 멤버 전용 방)
)

// ChannelConfig 은 한 채널에서 켜질 기능 목록, 표시 이름, 말투를 담는다.
type ChannelConfig struct {
	Name     string
	Features []Feature
	Tone     Tone
}

// Channels 는 채널 ID -> 설정 매핑이다.
// 방이 늘어나면 여기에 항목만 추가하면 된다 (기능 코드는 건드리지 않는다).
//
// 채널 ID 확인 방법: 디스코드 설정 -> 고급 -> 개발자 모드 ON,
// 채널 우클릭 -> "채널 ID 복사".
var Channels = map[string]ChannelConfig{
	"1482655053646135440": { // #채팅방 (1번방)
		Name:     "채팅방",
		Features: []Feature{AutoReply, Yacht},
		Tone:     ToneMild,
	},
	"1538222697501298782": { // #개꿀이즤 (2번방)
		Name:     "개꿀이즤",
		Features: []Feature{AutoReply, Yacht, Nudge},
		Tone:     ToneFull,
	},
}

// Has 는 채널 설정에 특정 기능이 켜져 있는지 확인한다.
func (c ChannelConfig) Has(f Feature) bool {
	return slices.Contains(c.Features, f)
}

// Lookup 은 채널 ID로 설정을 찾는다. 등록되지 않은 채널이면 ok=false.
func Lookup(channelID string) (ChannelConfig, bool) {
	cfg, ok := Channels[channelID]
	return cfg, ok
}

// Enabled 는 해당 채널에서 기능이 켜져 있는지 확인한다.
// 메시지 이벤트와 버튼/커맨드 이벤트가 같은 규칙을 쓰도록 이 함수 하나만 사용한다.
func Enabled(channelID string, f Feature) bool {
	cfg, ok := Lookup(channelID)
	return ok && cfg.Has(f)
}
