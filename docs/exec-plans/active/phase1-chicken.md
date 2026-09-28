# 실행 계획: 1차 배포 "치킨 먼저"

- 브랜치: `feat/phase1-chicken`
- 설계: [docs/design/six-member.md](../../design/six-member.md)
- 시작: 2026-09-23

## 목표

2번방에서 "치킨 시킬까?"라고 하면 봇이 `[🎲 야추로 정하기]` 버튼을 띄우고, 누른 사람끼리 한 판 해서 치킨값 담당을 정한다. 쮀쮀이야 밈 자동 반응과 시간 투표도 같이 내보낸다. **DB는 쓰지 않는다**(2차에서 붙인다).

## 완료 기준

- [ ] `make check`가 통과한다 (gofmt, vet, build, `test -race`, 계층 검사, arm64 크로스 빌드)
- [ ] 순수 패키지(`config`, `interactions`, `voice`, `session`, `yacht`)가 discordgo를 import하지 않는다 (계층 검사로 강제)
- [ ] 트리거 테스트 데이터(`testdata/triggers_hit.txt`, `triggers_miss.txt`)가 실제 2번방 문장으로 채워져 있고 테이블 테스트로 고정돼 있다
- [ ] 1번방에서는 nudge가 반응하지 않고, 자동 반응은 mild 톤으로만 나온다
- [ ] 치킨 판: 참가자가 2명 미만이면 취소되고, 동점은 최대 3라운드 재굴림한 뒤 무작위로 정한다
- [ ] 시간 투표: 18~23시 중 지나지 않은 정각으로 기본 투표(복수 선택)가 올라가고, 모두 지났으면 내일 시간대로 올라간다 (KST 22:30과 23:30 경계를 테스트)
- [ ] `[🔕 오늘 그만]`을 누르면 KST 자정까지 해당 트리거가 멈춘다
- [ ] SIGTERM을 받으면 진행 중인 판의 버튼이 꺼지고 재시작 안내가 표시된다
- [ ] 테스트 서버에서 위 흐름을 손으로 한 번 확인했다
- [ ] EC2에 원자적 배포 스크립트로 배포했다

## 단계

| # | 작업 | 상태 |
|---|---|---|
| 0-b | discordgo Poll 지원 확인, modernc sqlite 호환 확인 | ✅ discordgo v0.29.0으로 업그레이드. sqlite는 Go 1.25 이상 필요 → 2차에서 올림 ([ADR 0002](../../adr/0002-discordgo-v029-poll.md)) |
| 1 | `config` 타입화(`Feature`, `Tone`, `Enabled`), `interactions` custom_id codec | ✅ 테스트 통과 |
| 2 | `voice/lines.go` 대사집 + 주사위 눈 | ✅ 테스트 통과 (대사는 초안, 사용자가 다듬을 예정) |
| H | 하네스 보강 (설계를 레포로, `make check`, 계층 검사, CI `-race`, ADR) | ✅ 계층 검사가 위반을 잡는지 일부러 어겨서 확인함 |
| 3 | `session/registry.go` (`Touch`, `Finish` 1회 보장, `CloseAll`, 유저 역인덱스, `-race` 테스트) | ⏳ 다음 |
| 4 | `features/autoreply.go` + 트리거 테스트 데이터 | ⬜ (사용자 과제: 2번방 문장 수집 필요) |
| 5 | `yacht/chicken.go` 순수 로직 + `features/nudge.go` 치킨 흐름 | ⬜ |
| 6 | 시간 투표 (기본 투표 게시, KST 경계) | ⬜ |
| 7 | `main.go` 연결, 순서 있는 종료, 배포 스크립트, 1차 배포 | ⬜ |

## 결정 로그

- 2026-09-23: 치킨 먼저 배포하고 야추 대결과 원장은 2차로 미룸 ([ADR 0004](../../adr/0004-chicken-first-rollout.md))
- 2026-09-23: 시간 고르기는 직접 만들지 않고 디스코드 기본 투표를 쓴다. 이를 위해 discordgo v0.29로 올림 ([ADR 0002](../../adr/0002-discordgo-v029-poll.md))
- 2026-09-23: 규칙은 순수 패키지에 두고, 판 상태는 공용 session.Registry로 관리한다 ([ADR 0003](../../adr/0003-pure-core-shared-session.md))
- 2026-09-28: 1·2단계를 exec-plan 없이 먼저 시작한 것을 발견함 (AGENTS.md 작업 시작 규칙 위반). 이 문서를 소급해서 작성함
