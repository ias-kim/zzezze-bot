# ADR 0003: 순수 로직 패키지 + 공용 session.Registry

## 상태

승인됨 (2026-09-23, /plan-eng-review 이슈 1·4·5·7)

## 맥락

야추 대결과 치킨 판은 모두 "봇 메시지에 달린 버튼 + 메모리 상태 + 타이머 + 재시작하면 만료"라는 구조를 가진다. 이 동시성 코드를 기능마다 따로 짜면 경합 버그도 기능마다 따로 생긴다. 게임 규칙이 discordgo 핸들러 안에 섞여 있으면 진짜 디스코드 없이는 테스트할 수 없고, 그러면 TDD를 할 수 없다.

## 결정

- 게임 규칙은 discordgo를 import하지 않는 순수 패키지(`yacht/`)에 둔다. 대결은 `Duel.Apply(Event) → (Duel, Outcome, error)` 상태 머신으로 만든다.
- 판 상태 관리는 `session.Registry` 하나로 공유한다(`Touch`, `Finish` 1회 보장, `CloseAll`, 유저 역인덱스).
- 기능 키는 `config.Feature` 타입 상수로, 버튼 ID는 `interactions` codec(`기능:동작[:인자]`)으로, 대사는 `voice` 대사집으로 한 곳에 모은다.
- discordgo는 `main`과 `features/`만 import한다. 이 규칙은 `scripts/check-layers.sh`가 강제한다.

## 결과

- 게임 규칙과 동시성 코드를 `go test -race`만으로 검증할 수 있다.
- `features/`는 얇은 연결 계층이 된다. 화면 표시가 맞는지는 수동으로 확인한다.
