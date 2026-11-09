package features

import (
	"strconv"
	"time"

	"zzezze-bot/config"
	"zzezze-bot/session"
	"zzezze-bot/voice"
)

// 시간 투표 선택지 범위(KST, 정각). 마지막 시각은 포함한다.
const (
	firstSlotHour = 18
	lastSlotHour  = 23
	pollHours     = 3 // 투표 마감까지 시간 (디스코드 기본 투표는 시간 단위)
)

// TimeSlots 는 오늘 18~23시 중 아직 지나지 않은 정각을 돌려준다.
// 모두 지났으면 내일 18~23시를 돌려준다. 정각과 같은 시각은 지난 것으로 본다.
func TimeSlots(now time.Time) []time.Time {
	n := now.In(session.KST)
	slots := daySlots(n, 0)
	var out []time.Time
	for _, s := range slots {
		if s.After(n) {
			out = append(out, s)
		}
	}
	if len(out) == 0 {
		return daySlots(n, 1)
	}
	return out
}

func daySlots(n time.Time, addDays int) []time.Time {
	var out []time.Time
	for h := firstSlotHour; h <= lastSlotHour; h++ {
		out = append(out, time.Date(n.Year(), n.Month(), n.Day()+addDays, h, 0, 0, 0, session.KST))
	}
	return out
}

// SlotLabel 은 투표 선택지 문구다. 오늘이 아니면 내일 문구를 쓴다.
func SlotLabel(slot, now time.Time, tone config.Tone, pick voice.Picker) string {
	s, n := slot.In(session.KST), now.In(session.KST)
	key := voice.TimeSlotToday
	if s.YearDay() != n.YearDay() || s.Year() != n.Year() {
		key = voice.TimeSlotNextDay
	}
	return voice.Say(key, tone, pick, "{hour}", strconv.Itoa(s.Hour()))
}
