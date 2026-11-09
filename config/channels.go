package config

import "slices"

// Feature 는 채널에서 켤 수 있는 기능 이름이다.
// 문자열 대신 상수를 쓰면 오타가 컴파일 에러가 된다.
type Feature string

const (
	AutoReply Feature = "autoReply" // 쮀쮀이야 계열 밈 자동 반응
	Yacht     Feature = "yacht"     // "야추겜 ㄱㄱ" 빠른 판 (1차), /야추 대결·/포인트·/랭킹 (2차)
	Nudge     Feature = "nudge"     // 음식/시간 얘기 → 시간 투표 제안
)

// Tone 은 봇 대사의 말투 강도다.
type Tone string

const (
	ToneMild Tone = "mild" // 가벼운 밈만 (지금은 Overrides 로만 쓸 수 있다)
	ToneFull Tone = "full" // 방 밈 말투 풀파워 (기본값)
)

// ChannelConfig 은 한 채널에서 켜질 기능 목록, 표시 이름, 말투를 담는다.
type ChannelConfig struct {
	Name     string
	Features []Feature
	Tone     Tone
}

// Default 는 모든 채널에 적용되는 기본 설정이다.
// 비핵심 인원이 모두 강퇴되어 방마다 나눌 필요가 없어졌다 (docs/adr/0006-no-room-split.md).
var Default = ChannelConfig{
	Name:     "기본",
	Features: []Feature{AutoReply, Yacht, Nudge},
	Tone:     ToneFull,
}

// Overrides 는 기본 설정과 다르게 동작해야 하는 채널만 적는다 (채널 ID -> 설정).
// 지금은 비어 있다. 특정 방을 조용하게 만들고 싶으면 여기에 항목을 추가한다.
//
// 채널 ID 확인 방법: 디스코드 설정 -> 고급 -> 개발자 모드 ON,
// 채널 우클릭 -> "채널 ID 복사".
var Overrides = map[string]ChannelConfig{}

// Has 는 채널 설정에 특정 기능이 켜져 있는지 확인한다.
func (c ChannelConfig) Has(f Feature) bool {
	return slices.Contains(c.Features, f)
}

// Lookup 은 채널 ID로 설정을 찾는다. override 가 없으면 Default 를 돌려준다.
// 채널 ID 가 비어 있으면 ok=false.
func Lookup(channelID string) (ChannelConfig, bool) {
	if channelID == "" {
		return ChannelConfig{}, false
	}
	if cfg, ok := Overrides[channelID]; ok {
		return cfg, true
	}
	return Default, true
}

// Enabled 는 해당 채널에서 기능이 켜져 있는지 확인한다.
// 메시지 이벤트와 버튼/커맨드 이벤트가 같은 규칙을 쓰도록 이 함수 하나만 사용한다.
func Enabled(channelID string, f Feature) bool {
	cfg, ok := Lookup(channelID)
	return ok && cfg.Has(f)
}
