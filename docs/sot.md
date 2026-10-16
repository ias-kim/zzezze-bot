# Source of Truth 표

사실 유형별로 원천을 하나만 정한다. 다른 문서는 여기로 링크만 걸고 값을 복사하지 않는다. "일치 점검"이 `make check`이면 기계가, "수동"이면 GC 체크리스트에서 사람이 확인한다.

| 사실 유형 | SoT | 파생 (값을 복사하는 곳) | 일치 점검 |
|---|---|---|---|
| 채널 ↔ 기능 매핑, 방별 말투 | `config/channels.go` | 설계 문서 "채널별 기능" 표 (설명용) | 코드 자체는 `config/channels_test.go`, 설계 표는 수동 |
| 봇 대사, 대사 키, 주사위 눈 | `voice/lines.go` | 없음 | `make check` (`keys` 목록 ↔ `lines` 표 집합 일치, 키마다 full/mild 존재) |
| 버튼 custom_id 형식 | `interactions/customid.go` | 설계 문서, AGENTS.md 설명 | `make check` (codec 테스트) |
| 제품 설계, 게임 규칙(족보 표 등) | `docs/design/six-member.md` | 2차에서 `yacht/` 테이블 테스트로 옮김 | 옮긴 뒤에는 테스트 |
| 의존 방향 규칙 | `scripts/check-layers.sh` | AGENTS.md, 설계 문서, ADR 0003 설명 | `make check` (`layers`) |
| 서브에이전트 지시문 | `.claude/agents/*.md` 본문 | `.codex/agents/*.toml` | `make check` (`agents`) |
| Go 버전, 의존성 버전 | `go.mod` (+ `go.sum`) | CI(`go-version-file`)가 직접 읽음 | `make check` (`tidy`) |
| 게이트 1 검사 항목 | `Makefile` (`check` 타깃) | AGENTS.md 명령어 절, CI | 수동 |
| DB 스키마 (2차) | `store/schema.sql` (예정) | 없음 | store 테스트가 스키마를 적용해서 실행 |
| 환경 변수 목록 | `.env.example` | 없음 (README는 링크만) | - |
| 설계 결정 | `docs/adr/*.md` | AGENTS.md, profile.md는 링크만 | 수동 |
| 하네스 설정, 비용 방침 | `docs/harness/profile.md` | - | - |
| 진행 중 작업, 완료 기준 | `docs/exec-plans/active/*.md` | 완료 시 `completed/`로 이동 | 수동 |
| 오늘 상태 / 다음 할 일 | `docs/handoff.md` | - | 하루 마무리 때 갱신 |
| 재발 방지 교훈 | `docs/solutions/*.md` (필요할 때 생성) | - | 수동 |
| 기술 부채 | `docs/exec-plans/tech-debt-tracker.md` | - | 수동 |

## 해당 없음

- API 계약: 없음 (Discord 게이트웨이 이벤트 수신만, 자체 API 미노출)
