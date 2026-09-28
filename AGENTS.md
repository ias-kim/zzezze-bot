# AGENTS.md

BM 증바람 서버용 디스코드 봇. 봇 1개가 채널 ID를 인식해 방마다 다른 기능을 실행한다. 설계는 `docs/design/six-member.md`, 지금 진행 중인 작업은 `docs/exec-plans/active/`를 본다.

## 명령어

```bash
make check       # 게이트 1 전체: gofmt, vet, build, test -race, 계층 검사, arm64 크로스 빌드
make test        # go test -race ./...
make layers      # 의존 방향 검사만 (scripts/check-layers.sh)
make run         # 로컬 실행 (.env 필요, .env.example 참고)
go mod tidy      # 의존성 정리
```

PR을 올리기 전과 "고쳤다"고 말하기 전에 `make check`를 통과시킨다. CI(`.github/workflows/ci.yml`)도 같은 `make check`를 실행한다.

## 계층/의존 방향 규칙

discordgo는 `main.go`와 `features/`에서만 import한다. 나머지는 디스코드 없이 `go test`로 검증되는 순수 패키지다. 이 규칙은 `make layers`가 강제한다. 근거는 `docs/adr/0003-pure-core-shared-session.md`에 있다.

- `config/`: 채널 → 기능(`Feature` 상수), 말투(`Tone`) 매핑만 둔다. 기능이 켜져 있는지는 `config.Enabled()` 하나로만 확인한다.
- `interactions/`: 버튼 custom_id 형식(`기능:동작[:인자]`)을 인코딩/디코딩한다.
- `voice/`: 봇 대사집(full/mild)과 주사위 눈. 로직 코드에 한국어 대사를 직접 쓰지 않고 `voice.Say()`로 가져온다.
- `session/`: 버튼 달린 판의 메모리 상태, 타이머, 1회 종료 보장.
- `yacht/`: 게임 규칙을 순수 함수로 둔다. 무작위성은 주입받는다(`Roller`, `Picker`).
- `store/` (2차): SQLite. 모든 함수가 `*sql.Tx`를 받는다.
- `features/`: 디스코드 이벤트 ↔ 순수 로직 변환과 화면 그리기만 한다.
- `main.go`: 세션 생성, 인텐트, 핸들러 등록, 종료 순서만 담당한다.
- 순수 패키지는 `features/`와 `store/`를 import하지 않는다.

## 사실의 원천

이 문서에 값을 복사하지 않는다. 사실은 `docs/sot.md`를 본다.

## 작업 시작 규칙

기능 단위 작업을 시작하기 전에 `docs/exec-plans/active/<slug>.md`에 완료 기준을 먼저 적는다(**코드보다 먼저**). 구현은 TDD로 한다. 테스트를 먼저 쓰고 실패(RED)를 확인한 다음 구현한다. 끝나면 `docs/exec-plans/completed/`로 옮긴다. 설계 결정이 생기면 `docs/adr/`에 짧게 기록한다.

## 코딩 에이전트별 메모

- Claude Code: `CLAUDE.md` 참고 (이 문서를 따른다는 내용 + Claude Code 전용 메모만 있음).
- Codex: 이 문서(`AGENTS.md`)를 그대로 따른다. 별도 메모 없음.

## 하네스 설정

이 프로젝트의 하네스 레벨/위험 영역 등은 `docs/harness/profile.md` 참고.
