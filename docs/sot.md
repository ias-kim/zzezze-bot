# Source of Truth 표

사실 유형별로 원천을 하나만 정한다. `AGENTS.md`/README는 여기로 링크만 걸고 값을 복사하지 않는다.

| 사실 유형 | SoT | 파생 | 일치 점검 방법 |
|---|---|---|---|
| 채널 ↔ 기능 매핑, 방별 말투 | `config/channels.go` | 설계 문서의 "채널별 기능" 표(설명용) | `config/channels_test.go` |
| 봇 대사, 주사위 눈 | `voice/lines.go` | 없음 | `voice/lines_test.go` (모든 키에 full/mild가 있는지) |
| 버튼 custom_id 형식 | `interactions/customid.go` | 없음 | `interactions/customid_test.go` |
| 제품 설계, 게임 규칙(족보 표 등) | `docs/design/six-member.md` | 2차에서 `yacht/score.go` 테이블 테스트로 옮김 | 테스트 |
| 의존 방향 규칙 | `scripts/check-layers.sh` | `AGENTS.md` 설명 | `make layers` |
| Go 버전 | `go.mod` | CI (`go-version-file`) | CI가 직접 읽음 |
| DB 스키마 (2차) | `store/schema.sql` (예정) | 없음 | store 테스트가 스키마를 적용해서 실행 |
| 환경 설정 | `.env.example` | README "준비물/로컬 실행" 절 | 수동 확인 (변수 추가 시 둘 다 갱신) |
| 설계 결정 | `docs/adr/*.md` | `AGENTS.md`는 링크만 | - |
| 진행 중 작업/완료 기준 | `docs/exec-plans/active/*.md` | 완료 시 `docs/exec-plans/completed/`로 이동 | - |
| 기술 부채 | `docs/exec-plans/tech-debt-tracker.md` | - | - |

## 해당 없음

- API 계약: 없음 (Discord 게이트웨이 이벤트 수신만, 자체 API 미노출)
