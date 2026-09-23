package config

// ChannelConfig 은 한 채널에서 켜질 기능 목록과 표시 이름을 담는다.
type ChannelConfig struct {
	Name     string
	Features []string
}

// Channels 는 채널 ID -> 설정 매핑이다.
// 방이 늘어나면 여기에 항목만 추가하면 된다 (기능 코드는 건드리지 않는다).
//
// 채널 ID 확인 방법: 디스코드 설정 -> 고급 -> 개발자 모드 ON,
// 채널 우클릭 -> "채널 ID 복사".
var Channels = map[string]ChannelConfig{
	"1482655053646135440": { // #채팅방 (1번방)
		Name:     "채팅방",
		Features: []string{"autoReply", "moderation", "yacht"},
	},
	"1538222697501298782": { // #개꿀이즤 (2번방)
		Name:     "개꿀이즤",
		Features: []string{"autoReply", "moderation", "schedule", "yacht"},
	},
}

// Has 는 채널 설정에 특정 기능이 켜져 있는지 확인한다.
func (c ChannelConfig) Has(feature string) bool {
	for _, f := range c.Features {
		if f == feature {
			return true
		}
	}
	return false
}
