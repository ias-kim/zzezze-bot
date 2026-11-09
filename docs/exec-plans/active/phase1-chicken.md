# 실행 계획: 1차 배포 "치킨 먼저"

- 브랜치: `feat/phase1-chicken`
- 설계: [docs/design/six-member.md](../../design/six-member.md)
- 시작: 2026-09-23

## 목표

~~"치킨 시킬까?" → `[🎲 야추로 정하기]`로 치킨값 담당 정하기~~ → [ADR 0007](../../adr/0007-yacht-separate-from-food.md)로 바뀜: "야추겜 ㄱㄱ"로 야추 빠른 판을 따로 열고, 음식·시간 얘기에는 약속 시간 투표를 제안한다. 쮀쮀이야 밈 자동 반응도 같이 내보낸다. **DB는 쓰지 않는다**(2차에서 붙인다).

## 완료 기준

- [x] `make check`가 통과한다 (검사 항목은 Makefile `check` 타깃이 원천)
- [x] `main.go`와 `features/` 밖의 패키지가 discordgo를 import하지 않는다 (`make layers`로 강제, 테스트 파일 포함)
- [ ] nudge 트리거 테스트 데이터(`features/testdata/nudge_hit.txt`, `nudge_miss.txt`)가 실제 2번방 문장으로 채워져 있고 테이블 테스트로 고정돼 있다 (테이블 테스트 ✅, 문장은 아직 예시)
- [x] ~~1번방에서는 nudge가 반응하지 않고, 자동 반응은 mild 톤으로만 나온다~~ → 방 구분 폐지 ([ADR 0006](../../adr/0006-no-room-split.md)). 대신: 모든 채널에서 같은 기능, full 톤
- [x] 야추 빠른 판("야추겜 ㄱㄱ"): 참가자가 2명 미만이면 취소되고, 동점은 최대 3라운드 재굴림한 뒤 무작위로 정한다
- [x] 시간 투표: 18~23시 중 지나지 않은 정각으로 기본 투표(복수 선택)가 올라가고, 모두 지났으면 내일 시간대로 올라간다 (KST 22:30과 23:30 경계를 테스트)
- [x] `[🔕 오늘 그만]`을 누르면 KST 자정까지 해당 트리거가 멈춘다
- [x] SIGTERM을 받으면 진행 중인 판의 버튼이 꺼지고 재시작 안내가 표시된다 (코드·단위 테스트까지. 실서버 확인은 아래 수동 확인 항목에 포함)
- [ ] 테스트 서버에서 위 흐름을 손으로 한 번 확인했다
- [ ] EC2에 원자적 배포 스크립트로 배포했다 (서버 준비·등록은 사용자가 AWS 에이전트로 한다. 절차는 [docs/deploy.md](../../deploy.md))

## 단계

| # | 작업 | 상태 |
|---|---|---|
| 0-b | discordgo Poll 지원 확인, modernc sqlite 호환 확인 | ✅ discordgo v0.29.0으로 업그레이드. sqlite는 Go 1.25 이상 필요 → 2차에서 올림 ([ADR 0002](../../adr/0002-discordgo-v029-poll.md)) |
| 1 | `config` 타입화(`Feature`, `Tone`, `Enabled`), `interactions` custom_id codec | ✅ 테스트 통과 |
| 2 | `voice/lines.go` 대사집 + 주사위 눈 | ✅ 테스트 통과 (대사는 초안, 사용자가 다듬을 예정) |
| H | 하네스 보강 (설계를 레포로, `make check`, 계층 검사, CI `-race`, ADR) | ✅ 계층 검사가 위반을 잡는지 일부러 어겨서 확인함 |
| H2 | reviewer 감사 23건 반영: README 재작성, 계층 검사 강화(테스트 import, 새 패키지 자동 포함), `tidy`/`agents` 검사 추가, voice 키 목록 테스트, main.go `config.Lookup` 사용, profile 모순 해소(ADR 0005), handoff 도입 | ✅ 2026-10-15 |
| 3 | `session/registry.go` (`Touch`, `Finish` 1회 보장, `CloseAll`, 유저 역인덱스, `-race` 테스트) + `session/throttle.go` (쿨다운, KST 자정 멈춤) | ✅ 2026-11-09 |
| 4 | 밈 자동 반응 (`features/triggers.go` `IsMeme`, 채널별 쿨다운 3분) | ✅ 2026-11-09 |
| 5 | `yacht/chicken.go` 순수 로직 + 치킨 흐름(`features/bot.go`, `chicken.go`) + `features/testdata/nudge_*.txt` | ✅ 코드 완료. testdata는 예시 문장 → 사용자가 실제 문장으로 교체 필요 |
| 6 | 시간 투표 (`features/timepoll.go`, 기본 투표 게시, KST 경계) | ✅ 2026-11-09 |
| 7 | `main.go` 연결, 순서 있는 종료, 배포 파일(`deploy/`, `scripts/install.sh`, `make dist`, `docs/deploy.md`) | ✅ 코드 완료. 실제 배포는 사용자(AWS 에이전트) |

## 결정 로그

- 2026-09-23: 치킨 먼저 배포하고 야추 대결과 원장은 2차로 미룸 ([ADR 0004](../../adr/0004-chicken-first-rollout.md))
- 2026-09-23: 시간 고르기는 직접 만들지 않고 디스코드 기본 투표를 쓴다. 이를 위해 discordgo v0.29로 올림 ([ADR 0002](../../adr/0002-discordgo-v029-poll.md))
- 2026-09-23: 규칙은 순수 패키지에 두고, 판 상태는 공용 session.Registry로 관리한다 ([ADR 0003](../../adr/0003-pure-core-shared-session.md))
- 2026-09-28: 1·2단계를 exec-plan 없이 먼저 시작한 것을 발견함 (AGENTS.md 작업 시작 규칙 위반). 이 문서를 소급해서 작성함
- 2026-10-15: 배포 대상을 t4g.nano로 확정 (월 약 $8). Codex 무료 플랜이라 검증은 Claude 서브에이전트가 맡고, 서브에이전트에는 가벼운 모델을 지정함 (profile.md)
- 2026-11-09: 야추를 "야추겜 ㄱㄱ"로 분리하고 음식 얘기는 약속 시간 투표로 바꿈 ([ADR 0007](../../adr/0007-yacht-separate-from-food.md))
- 2026-11-09: 음식 트리거를 치킨 → 음식 목록(곱창, 양꼬치 등)과 "뭐 시킬까"로 넓히고 대사에 음식 이름을 넣음. 밈 반응에 "줴줴이야", "조줴줴이야", "최재희야" 추가 (사용자 요청)
- 2026-11-09: 비핵심 인원이 모두 강퇴되어 방 구분을 없앰 ([ADR 0006](../../adr/0006-no-room-split.md)). EC2 서버 등록은 사용자가 AWS 에이전트로 하고, 레포는 바이너리·systemd 유닛·설치 스크립트·런북만 제공한다
- 2026-10-15: 하루 마무리 때 `docs/handoff.md`를 갱신하고 push 함 (AGENTS.md 작업 규칙)
