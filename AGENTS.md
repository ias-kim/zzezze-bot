# AGENTS.md

BM 증바람 서버용 디스코드 봇. 봇 1개가 채널 ID를 보고 방마다 다른 기능을 실행한다.

**세션을 시작하면 먼저 [docs/handoff.md](docs/handoff.md)를 읽는다.** 마지막으로 어디까지 했고 다음에 뭘 할지가 적혀 있다. 설계는 `docs/design/six-member.md`, 진행 중인 작업은 `docs/exec-plans/active/`를 본다.

## 명령어

```bash
make check       # 게이트 1 전체 (검사 항목은 Makefile 의 check 타깃이 원천)
make test        # go test -race ./...
make layers      # 의존 방향 검사만 (scripts/check-layers.sh)
make agents      # .claude/agents 와 .codex/agents 지시문이 같은지 (scripts/check-agents-sync.sh)
make run         # 로컬 실행 (.env 필요, .env.example 참고)
```

PR을 올리기 전과 "고쳤다"고 말하기 전에 `make check`를 통과시킨다. CI(`.github/workflows/ci.yml`)도 같은 `make check`를 실행한다.

## 계층/의존 방향 규칙

discordgo는 `main.go`와 `features/`에서만 import한다(`_test.go` 포함). 나머지는 디스코드 없이 `go test`로 검증되는 순수 패키지다. `features/`와 `store/`를 import할 수 있는 쪽은 `main.go`와 `features/`뿐이다(순수 패키지는 이 둘을 모른다). 이 규칙은 `make layers`가 **새 패키지까지 자동으로** 강제한다. 근거는 `docs/adr/0003-pure-core-shared-session.md`에 있다.

있는 것:
- `config/`: 채널 → 기능(`Feature` 상수), 말투(`Tone`) 매핑만 둔다. 기능이 켜져 있는지는 `config.Enabled()`로, 채널 설정은 `config.Lookup()`으로만 조회한다(맵을 직접 열지 않는다).
- `interactions/`: 버튼 custom_id 형식(`기능:동작[:인자]`)을 인코딩/디코딩한다.
- `voice/`: 봇 대사집(full/mild)과 주사위 눈. 로직 코드에 한국어 대사를 직접 쓰지 않고 `voice.Say()`로 가져온다. 새 대사 키는 상수, `keys` 목록, `lines` 표 세 곳에 넣는다(테스트가 어긋남을 잡는다).
- `session/`: 버튼 달린 판의 메모리 상태, 방치 타이머, 1회 종료 보장, 선택적 유저 역인덱스(`session.Registry`).
- `main.go`: 세션 생성, 인텐트, 핸들러 등록, 종료 순서만 담당한다.

예정 (exec-plan 단계 표기):
- `features/` (1차 4~7단계): 디스코드 이벤트 ↔ 순수 로직 변환과 화면 그리기만 한다.
- `meetup/` (1차 5단계): 약속 트리거 판정, 메시지 묶기, 날짜 해석, 약속 카드 상태를 순수 함수로 둔다. 현재 시각은 주입받는다.
- `yacht/` (2차 대결): 게임 규칙을 순수 함수로 둔다. 무작위성은 주입받는다(이름은 구현할 때 정한다).
- `store/` (2차): SQLite. 모든 함수가 `*sql.Tx`를 받는다.

## 사실의 원천

이 문서에 값(버전, 기능 목록, 채널 ID 등)을 복사하지 않는다. 사실은 `docs/sot.md`를 본다.

## 작업 규칙

- **시작**: 기능 단위 작업을 시작하기 전에 `docs/exec-plans/active/<slug>.md`에 완료 기준을 먼저 적는다(**코드보다 먼저**).
- **구현**: TDD로 한다. 테스트를 먼저 쓰고 실패(RED)를 확인한 다음 구현한다.
- **완료**: 끝난 계획은 `docs/exec-plans/completed/`로 옮긴다. 설계 결정이 생기거나 뒤집히면 `docs/adr/`에 짧게 기록한다.
- **교훈**: 검증과 리뷰가 끝난 작업에서, 코드·테스트·문서만으로는 알기 어려워 같은 실수나 조사를 반복할 위험이 있는 교훈이 생기면 `docs/solutions/<주제>.md`에 남긴다. 일상적인 수정이면 남기지 않는다. 작성 방법은 아래 "에이전트별 메모"를 본다.
- **하루 마무리**: 사용자가 작업 종료를 알리면 `docs/handoff.md`를 덮어써서 갱신(오늘 한 것 / 다음 할 것 / 막힌 것 / 새 결정)하고, 커밋한 뒤 `git push`한다. 날짜별 기록은 git 히스토리(`git log -p docs/handoff.md`)가 대신한다.

## 코딩 에이전트별 메모

- **Claude Code**: `CLAUDE.md`를 본다. 서브에이전트는 `.claude/agents/`(explorer=haiku, reviewer=sonnet)에 있다. 교훈 기록 도구(ce-compound)는 설치돼 있지 않으므로 `docs/solutions/`에 직접 작성한다(제목, 증상, 원인, 해결, 재발 방지 다섯 절).
- **Codex**: 이 문서를 그대로 따른다. 서브에이전트는 `.codex/agents/`에 있다(지시문은 `.claude/agents/`와 같아야 하며 `make agents`가 검사한다). 교훈 기록은 `/ce-compound mode:non-interactive depth:lightweight`를 쓴다(compound-engineering 플러그인).

## 하네스 설정

이 프로젝트의 하네스 레벨, 위험 영역, 비용 방침은 `docs/harness/profile.md`를 본다.
