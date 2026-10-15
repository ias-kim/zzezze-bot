// Package interactions 는 버튼 custom_id 형식("기능:동작[:인자]")을 담당한다.
// 버튼 → 핸들러 라우팅은 discordgo 가 필요하므로 features/ 쪽에 둔다.
// Feature 필드는 string 이다. config.Feature 와 이름 공간이 다를 수 있어 일부러 묶지 않았다.
package interactions

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// 디스코드는 custom_id 를 최대 100자로 제한한다.
const maxLen = 100

const sep = ":"

// ID 는 버튼 custom_id 의 구조화된 형태다. 형식: "기능:동작[:인자]".
// 판 상태는 session.Registry(메시지 ID 키)에 있으므로 여기에는 담지 않는다.
type ID struct {
	Feature string // 예: "yacht", "nudge"
	Action  string // 예: "hold", "accept", "chicken"
	Arg     string // 선택. 예: "3", "join"
}

// ErrInvalid 는 custom_id 형식이 잘못되었을 때 반환된다.
var ErrInvalid = errors.New("invalid custom_id")

// Encode 는 ID 를 custom_id 문자열로 만든다.
func Encode(id ID) (string, error) {
	if id.Feature == "" || id.Action == "" {
		return "", fmt.Errorf("%w: feature/action required: %+v", ErrInvalid, id)
	}
	for _, part := range []string{id.Feature, id.Action, id.Arg} {
		if strings.Contains(part, sep) {
			return "", fmt.Errorf("%w: %q contains %q", ErrInvalid, part, sep)
		}
	}
	s := id.Feature + sep + id.Action
	if id.Arg != "" {
		s += sep + id.Arg
	}
	if len(s) > maxLen {
		return "", fmt.Errorf("%w: %d chars > %d", ErrInvalid, len(s), maxLen)
	}
	return s, nil
}

// Decode 는 custom_id 문자열을 ID 로 해석한다.
func Decode(s string) (ID, error) {
	if s == "" || len(s) > maxLen {
		return ID{}, fmt.Errorf("%w: length %d", ErrInvalid, len(s))
	}
	parts := strings.Split(s, sep)
	if len(parts) < 2 || len(parts) > 3 {
		return ID{}, fmt.Errorf("%w: %q", ErrInvalid, s)
	}
	if slices.Contains(parts, "") {
		return ID{}, fmt.Errorf("%w: empty part in %q", ErrInvalid, s)
	}
	id := ID{Feature: parts[0], Action: parts[1]}
	if len(parts) == 3 {
		id.Arg = parts[2]
	}
	return id, nil
}
