# 실행 계획: 1차 배포 "치킨 먼저"

- 브랜치: `feat/phase1-chicken`
- 설계: [docs/design/six-member.md](../../design/six-member.md)
- 시작: 2026-09-23

## 목표

2번방에서 "치킨 시킬까?"라고 하면 봇이 `[🎲 야추로 정하기]` 버튼을 띄우고, 누른 사람끼리 한 판 해서 치킨값 담당을 정한다. 쮀쮀이야 밈 자동 반응과 시간 투표도 같이 내보낸다. **DB는 쓰지 않는다**(2차에서 붙인다).

## 완료 기준

- [x] `make check`가 통과한다 (검사 항목은 Makefile `check` 타깃이 원천)
- [x] `main.go`와 `features/` 밖의 패키지가 discordgo를 import하지 않는다 (`make layers`로 강제, 테스트 파일 포함)
- [~] nudge 트리거 테스트 데이터(`features/testdata/nudge_hit.txt`, `nudge_miss.txt`)가 실제 2번방 문장으로 채워져 있고 테이블 테스트로 고정돼 있다 → 초안으로 고정(2026-11-19). 실제 문장은 배포 후 행으로 추가
- [x] 1번방에서는 nudge가 반응하지 않고, 자동 반응은 mild 톤으로만 나온다
- [x] 치킨 판: 참가자가 2명 미만이면 취소되고, 동점은 최대 3라운드 재굴림한 뒤 무작위로 정한다
- [x] 시간 투표: 18~23시 중 지나지 않은 정각으로 기본 투표(복수 선택)가 올라가고, 모두 지났으면 내일 시간대로 올라간다 (KST 22:30과 23:30 경계를 테스트)
- [x] `[🔕 오늘 그만]`을 누르면 KST 자정까지 해당 트리거가 멈춘다
- [x] SIGTERM을 받으면 진행 중인 판의 버튼이 꺼지고 재시작 안내가 표시된다
- [ ] 테스트 서버에서 위 흐름을 손으로 한 번 확인했다
- [ ] EC2에 원자적 배포 스크립트로 배포했다

## 단계

| # | 작업 | 상태 |
|---|---|---|
| 0-b | discordgo Poll 지원 확인, modernc sqlite 호환 확인 | ✅ discordgo v0.29.0으로 업그레이드. sqlite는 Go 1.25 이상 필요 → 2차에서 올림 ([ADR 0002](../../adr/0002-discordgo-v029-poll.md)) |
| 1 | `config` 타입화(`Feature`, `Tone`, `Enabled`), `interactions` custom_id codec | ✅ 테스트 통과 |
| 2 | `voice/lines.go` 대사집 + 주사위 눈 | ✅ 테스트 통과 (대사는 초안, 사용자가 다듬을 예정) |
| H | 하네스 보강 (설계를 레포로, `make check`, 계층 검사, CI `-race`, ADR) | ✅ 계층 검사가 위반을 잡는지 일부러 어겨서 확인함 |
| H2 | reviewer 감사 23건 반영: README 재작성, 계층 검사 강화(테스트 import, 새 패키지 자동 포함), `tidy`/`agents` 검사 추가, voice 키 목록 테스트, main.go `config.Lookup` 사용, profile 모순 해소(ADR 0005), handoff 도입 | ✅ 2026-10-15 |
| 3 | `session/registry.go` (`Touch`, `Finish` 1회 보장, `CloseAll`, 유저 역인덱스, `-race` 테스트) | ✅ 2026-10-16 (아래 3단계 완료 기준 참고) |
| 4 | `features/autoreply.go` (밈 반응, 방별 쿨다운, 테스트 안의 표) | ✅ 2026-10-16 (수동 확인은 7단계 테스트 서버 확인 때 함께) |
| 5 | `yacht/chicken.go` 순수 로직 + `features/nudge.go` 치킨 흐름 + `features/testdata/nudge_*.txt` | ✅ 2026-11-19 (testdata는 초안, 실제 문장은 배포 후 추가) |
| 6 | 시간 투표 (기본 투표 게시, KST 경계) | ✅ 2026-11-19 |
| 7 | `main.go` 연결, 순서 있는 종료, 배포 스크립트, 1차 배포 | 🟨 연결·종료·배포 파일 ✅ 2026-11-19 / 테스트 서버 확인(U2)·AWS 승인(U3)·배포 ⬜ |

## 3단계 완료 기준: `session.Registry`

인스턴스는 **하나**다(ADR 0003 "Registry 하나로 공유"). 판 상태는 `any`로 담고 꺼내는 쪽(features)이 타입을 단언한다. 그래서 "한 유저 동시에 판 1개"는 기능을 가로질러 적용된다(치킨 판에 들어가 있으면 대결도 못 건다).

`go test -race ./session/`로 아래를 증명한다. 타이머는 가짜를 주입하고 sleep에 기대지 않는다. CI가 go.mod의 Go 1.23을 쓰므로 `testing/synctest` 등 1.24 이후 API는 쓰지 않는다.

- [x] `Start`는 원자적이다. 유저 중 한 명이라도 다른 판에 있으면 실패하고, 나머지 유저도 인덱스에 남지 않는다. 목록 안의 중복 유저, 이미 있는 msgID도 거절한다.
- [x] `Join`: 다른 판에 있는 유저는 `ErrUserBusy`, 같은 판에 이미 있으면 `ErrAlreadyJoined`, 없는 판이면 `ErrNotFound`.
- [x] `Finish`를 N개 고루틴이 동시에 불러도 true는 정확히 1번이다. 판과 그 판을 가리키는 역인덱스 항목만 지운다.
- [x] 타이머 만료도 같은 종료 경로를 거친다. 만료와 `Finish`가 경합하면 "onExpire 호출"과 "`Finish` true" 중 정확히 하나만 일어난다.
- [x] `Touch` 전에 걸린 타이머가 늦게 발화해도 판이 끝나지 않는다(세대 검사). 방치 시간은 마지막 `Touch`부터 센다.
- [x] onExpire는 모든 락 밖에서 호출된다(콜백 안에서 Registry를 불러도 데드락이 없다).
- [x] `CloseAll`은 남은 판(msgID, 상태)을 돌려주고, 이후 발화하는 타이머는 onExpire를 부르지 않으며, 이후 `Start`는 `ErrClosed`로 실패한다.
- [x] `Do(msgID, fn)`는 판별 락 안에서 fn을 실행한다. fn 안에서 `Touch`/`Finish`를 불러도 데드락이 없다.

## 4단계 완료 기준: 밈 자동 반응

판단(순수)과 디스코드 연결(얇음)을 나눈다. 쿨다운은 새 순수 패키지 `quiet/`에 두어 5단계 nudge가 재사용한다("오늘 그만"은 5단계에서 `quiet/`에 추가). `now`를 인자로 받아 sleep 없이 테스트한다.

- [x] 트리거 표 테스트: `[쮀줴제체재쩨졔]` 두 글자 + `이야`/`희야`(쮀쮀이야, 줴줴이야, 체재희야, 제제이야), "주 ㅔ" 같은 자모 분리 표기, "쮀쮀"/"줴줴" 단독에 반응한다. "줘줘", "졸졸", "이제 재희야", "쮀" 한 글자에는 반응하지 않는다.
- [x] 봇 메시지, `/` 또는 `!`로 시작하는 메시지, AutoReply가 꺼진(또는 등록 안 된) 채널에는 반응하지 않는다.
- [x] 대사는 채널 톤으로 `voice.MemeReply`에서 나온다(1번방 mild, 2번방 full).
- [x] `quiet.Gate`: 채널+트리거별 쿨다운 30분. 29분 59초 뒤에는 막고 30분 뒤에는 연다. 다른 채널/트리거는 서로 막지 않는다. 막힌 시도는 쿨다운을 늘리지 않는다. `-race`에서 동시 호출이 안전하다.
- [ ] 반응은 원문에 답장(reply)으로 보내고 멘션 알림은 끈다. `main.go`의 메시지 핸들러에 연결한다(화면은 수동 확인). → 코드와 연결 완료, 테스트 서버 수동 확인 남음

## 5·6단계 순수 로직 완료 기준 (2026-11-19, 문장 수집과 무관한 부분 먼저)

nudge 트리거(`features/nudge.go`, testdata)와 디스코드 연결은 2번방 문장 수집 뒤로 미룬다. 아래 세 가지는 discordgo 없이 서로 다른 패키지에서 병렬로 진행한다. 모두 `go test -race`로 검증하고 sleep에 기대지 않는다.

**5-a `yacht/chicken.go` 치킨 판**
- [x] 참가자가 2명 미만이면 `ErrNotEnoughPlayers`를 돌려주고 굴리지 않는다. 중복 참가자는 거절한다.
- [x] 참가자마다 주사위 5개를 굴려 합계가 가장 낮은 사람이 당첨이다. 주사위 함수와 최종 선택 함수는 주입받는다(테스트는 고정값).
- [x] 최저점 동점이면 동점자끼리만 다시 굴린다. 재굴림은 최대 3라운드이고, 그래도 같으면 주입받은 선택 함수로 동점자 중 한 명을 고른다.
- [x] 결과에는 라운드별 참가자 주사위와 합계가 남아 있어 화면을 한 번만 그릴 수 있다.

**6-a 시간 투표 선택지 (새 순수 패키지 `timevote/`)**
- [x] KST 기준 오늘 18~23시 정각 중 `now`보다 뒤인 것만 선택지로 낸다. 모두 지났으면 내일 18~23시 6개를 낸다.
- [x] 경계: KST 22:30 → 23시 하나, 23:30 → 내일 18~23시, 22:00 정각 → 23시 하나(같은 시각은 지난 것으로 본다), 17:59 → 오늘 18~23시.
- [x] `now`가 UTC 등 다른 시간대로 들어와도 KST로 바꿔 계산한다(`time/tzdata` 포함). 마감 시간 3시간은 상수로 둔다.

**5-b `quiet` "오늘 그만"**
- [x] `Gate.MuteToday(channelID, trigger, now)`를 부르면 그 채널+트리거는 다음 KST 자정 전까지 `Allow`가 false다. 자정이 지나면 다시 열린다.
- [x] 다른 채널·트리거는 영향받지 않는다. 음소거 중 막힌 시도는 쿨다운 기록을 바꾸지 않는다. `-race`에서 동시 호출이 안전하다.

**진단 반영 (2026-11-19 마피아MCP 검토)**
- [x] `main.go`가 메시지 원문과 작성자를 로그에 남기지 않는다(운영 서버 로그에 대화가 쌓이지 않게).

## 5~7단계 연결 계약 (2026-11-19, W1 에이전트 공통 입력)

오케스트레이션: W0(계약, 이 절) → W1 병렬(A nudge, B 치킨, C 시간 투표, F 배포 파일) → W2 통합 → W3 리뷰 → U2 테스트 서버 → U3 AWS 승인 → W4 배포 → U4 병합. A·B·C는 같은 `features` 패키지라 워크트리로 격리한다. nudge 테스트 데이터는 **초안으로 먼저** 만든다(사용자 결정 2026-11-19, 실제 2번방 문장은 배포 뒤 행으로 추가).

### 흐름

1. 2번방 메시지 → **A** `Nudge.Handle`: 트리거(chicken/time) 판정 + 트리거별 쿨다운 → 원문에 답장으로 제안 메시지와 버튼을 단다.
   - chicken: `NudgeChicken` + [`chicken:open` `ChickenBtnLabel`] [`nudge:mute:chicken` `MuteBtnLabel`]
   - time: `NudgeTime` + [`timevote:open` `TimeBtnLabel`] [`nudge:mute:time` `MuteBtnLabel`]
2. `nudge:mute:<trigger>` (**A**): `gate.MuteToday` → `updateMessage(NudgeMuted, 버튼 없음)`.
3. `chicken:open` (**B**): 제안 메시지를 그대로 판 메시지로 바꾼다. `Registry.Start(i.Message.ID, Users=[누른 사람], Idle=60s, OnExpire=정산)`. `ErrUserBusy`면 ephemeral `AlreadyInGame`. 성공하면 `updateMessage(ChickenOpen + 참가자 목록, [chicken:join JoinBtnLabel])`.
4. `chicken:join` (**B**): `Registry.Join`. `ErrNotFound`면 ephemeral `SessionExpired`, `ErrUserBusy`·`ErrAlreadyJoined`면 ephemeral `AlreadyInGame`. 성공하면 `Do`로 참가자를 추가하고 `updateMessage`. **Join은 Touch하지 않는다**(모집 60초 고정).
5. 만료 `OnExpire` (**B**): `yacht.PlayChicken`. `ErrNotEnoughPlayers`면 `editMessage(ChickenCancel)`. 아니면 라운드별 주사위·합계(재굴림 라운드 앞에 `ChickenTie`)와 `ChickenLoser`를 그려 `editMessage` **한 번**, 버튼 없음.
6. `timevote:open` (**C**): `updateMessage`로 제안 메시지의 버튼을 없앤 뒤 `ChannelMessageSendComplex`로 기본 투표를 올린다. 실패하면 로그와 채널에 `TimePollFailed`.
7. 종료 (**W2**): `CloseAll` → 상태가 `closable`이면 `editMessage(SessionRestart, 버튼 없음)`, 전체 5초 제한 → `dg.Close`.

### 공용 (W0에서 커밋. W1 에이전트는 고치지 않는다)

- `features/respond.go`: `replyEphemeral`, `updateMessage`, `editMessage`(타이머·종료용), `userID`, `mention`, `noMentions`
- `voice` 키: 기존 키 + `TimeOptionToday`/`TimeOptionTomorrow`(`{hour}` 치환), `TimePollFailed`. 새 대사가 필요하면 만들지 말고 보고한다.
- 쓰는 순수 패키지: `interactions.Encode/Decode`, `session.Registry`, `quiet.Gate`(`Allow`, `MuteToday`), `yacht.PlayChicken`, `timevote.Options`/`Duration`
- 종료 인터페이스 (W2가 `features/router.go`에 선언): `type closable interface{ closeInfo() (channelID string, tone config.Tone) }`. B의 판 상태는 이 메서드를 구현한다.

### 공개 API (W2 라우터가 이 이름으로 연결)

| 소유 | 생성자 | 메서드 | custom_id feature |
|---|---|---|---|
| A `features/nudge.go` | `NewNudge(gate *quiet.Gate) *Nudge`, `const NudgeCooldown = 30 * time.Minute` | `Handle(s, *discordgo.MessageCreate)`, `HandleComponent(s, *discordgo.InteractionCreate, interactions.ID)` | `nudge` |
| B `features/chicken.go` | `NewChicken(reg *session.Registry) *Chicken` | `HandleComponent(...)` | `chicken` |
| C `features/timepoll.go` | `NewTimeVote() *TimeVote` | `HandleComponent(...)` | `timevote` |

- 시간·난수는 공개 필드로 주입한다: `Now func() time.Time`, `Pick voice.Picker`, B는 `Roll func() int` 추가. 기본값은 `time.Now`, `rand.IntN`.
- 같은 패키지 이름 충돌 방지: 패키지 수준 비공개 이름에는 접두어를 붙인다(`nudge…`, `chicken…`, `timeVote…`). testdata 파일도 접두어로 구분한다.
- 판단과 화면 문자열은 순수 함수로 테스트한다(디스코드 호출은 얇게).

### 완료 기준 (W1~W3)

**A nudge**
- [x] `features/testdata/nudge_hit.txt`, `nudge_miss.txt`(초안)를 읽는 테이블 테스트로 판정을 고정한다
- [x] 치킨은 "치킨"과 의도(`시킬|시켜|먹을|ㄱ|뭐먹|?`)가 함께일 때만, 시간은 `몇\s*시|언제\s*(봄|볼|만나)`
- [x] Nudge가 꺼진 채널(1번방), 봇 메시지, `/`·`!` 메시지는 무시한다. 트리거별 쿨다운 30분, 음소거 중에는 반응하지 않는다
- [x] mute 버튼은 `MuteToday` 후 메시지를 갱신한다

**B 치킨 판**
- [x] open: 누른 사람이 첫 참가자다. 이미 다른 판에 있으면 거절한다
- [x] join: 모집 60초 고정. 중복·다른 판·만료를 처리한다
- [x] 정산: 2명 미만이면 취소, 결과는 메시지를 한 번 수정, 결과 문자열은 순수 함수로 테스트(재굴림 라운드 표시 포함)

**C 시간 투표**
- [x] 투표: 복수 선택, 선택지는 `timevote.Options`의 라벨(오늘/내일 `{hour}`시), **`Poll.Duration == 3`(시간 단위)를 테스트로 고정**
- [x] 게시에 실패하면 로그를 남기고 `TimePollFailed`를 보낸다 (권한 리스크)

**F 배포 파일**
- [x] `scripts/deploy.sh`: arm64 빌드 → `.new` 업로드 → 기존 바이너리를 `.prev`로 보존 → `mv` → restart. `rollback` 하위 명령으로 `.prev`를 되돌린다
- [x] `deploy/zzezze-bot.service`: `Restart=always`, 전용 계정, `EnvironmentFile`(권한 600), `TimeoutStopSec` 10초 이상
- [x] `docs/deploy.md`: EC2 준비(AWS 리전·플랜 확인 포함), 첫 배포, 되돌리기

**W2 통합**
- [x] 라우터: custom_id → feature 핸들러. 디코드 실패·모르는 feature는 로그만 남기고 무시한다
- [x] 모든 핸들러를 recover 래퍼로 감싼다(패닉을 로그로 남기고 프로세스 유지). 그래도 죽으면 systemd가 재시작한다
- [x] SIGTERM: `CloseAll` → 버튼 끄기와 `SessionRestart`(5초 제한) → `dg.Close`
- [x] `make check` 통과

**W3**
- [x] reviewer 서브에이전트 지적 반영
- [x] 테스트 서버 수동 확인 목록 작성 (MESSAGE CONTENT 인텐트, Send Polls 권한, 답장·버튼·투표 화면)

### 알려진 한계 (W3 리뷰, 받아들임)

- 모집 마감 직전(59.9초) [참가] 클릭의 화면 갱신이 만료 정산보다 늦게 디스코드에 도착하면, 결과 화면이 모집 화면으로 덮이고 [참가] 버튼이 다시 보일 수 있다. 누르면 버튼이 꺼지고 만료 안내가 나온다(정산 결과 자체는 맞다). 종료 안내도 같은 경합이 있다. 확률이 낮아 지연 처리는 넣지 않았다.
- [시간 투표]를 화면이 갱신되기 전에 두 번 누르면 투표가 두 개 올라갈 수 있다.
- 메시지 전송이 실패해도 쿨다운은 소비된다(로그만 남음).
- nudge 의도 정규식 오탐: "어제 치킨 시켜 먹었는데" 같은 과거형, "몇 시야 지금" 같은 현재 시각 질문. 실제 2번방 문장을 모은 뒤 testdata 와 패턴을 다듬는다.
- 배포 실패 뒤 rollback 없이 다시 배포하면 `.prev`가 실패한 바이너리로 바뀐다(직전 한 개만 보존, `docs/deploy.md`).

## 테스트 서버 확인 목록 (U2, 사용자가 직접)

자동 테스트로 확인할 수 없는 디스코드 화면과 권한을 본다. 테스트 서버의 채널 ID를 `config/channels.go`에 임시로 넣거나, 실제 서버에서 조용한 시간에 한다. `make run`으로 띄운다.

**사전 설정**
- [ ] Developer Portal → Bot → MESSAGE CONTENT INTENT 켜짐 (안 켜면 아무 반응이 없다)
- [ ] 봇 역할 권한: 메시지 보내기, 메시지 기록 보기, **투표 보내기(Send Polls)**

**밈 자동 반응 (4단계)**
- [ ] 2번방 "쮀쮀이야" → 원문에 답장으로 full 톤 대사, 알림(멘션) 없음
- [ ] 1번방 "쮀쮀이야" → mild 톤 대사
- [ ] 30분 안에 다시 쳐도 반응 없음

**치킨 (5단계)**
- [ ] 2번방 "치킨 시킬까?" → 답장으로 제안 + [야추로 정하기] [오늘 그만]
- [ ] 1번방 "치킨 시킬까?" → 반응 없음
- [ ] [야추로 정하기] → 같은 메시지가 모집 화면으로 바뀌고 누른 사람이 첫 참가자
- [ ] 다른 사람 [참가] → 목록에 추가. 같은 사람이 다시 누르면 본인에게만 안내
- [ ] 60초 뒤 결과가 메시지 한 번 수정으로 나오고 버튼이 사라짐 (2명 이상)
- [ ] 혼자만 참가하면 60초 뒤 취소 안내
- [ ] 끝난 판의 버튼을 누르면 본인에게만 만료 안내

**시간 투표 (6단계)**
- [ ] "몇 시에 봄?" → 제안 + [시간 투표] [오늘 그만]
- [ ] [시간 투표] → 제안 버튼이 사라지고 기본 투표가 올라옴: 복수 선택, 선택지 "오늘/내일 N시", **마감 3시간**
- [ ] (권한을 일부러 빼고) 투표 실패 안내가 나오는지 — 선택

**오늘 그만**
- [ ] [오늘 그만] → 안내로 바뀌고 버튼 사라짐. 같은 트리거 문장에 반응 없음, 다른 트리거는 반응함

**종료 (7단계)**
- [ ] 치킨 모집 중에 Ctrl+C → 그 메시지의 버튼이 사라지고 재시작 안내가 뜬 뒤 프로세스 종료

## 결정 로그

- 2026-09-23: 치킨 먼저 배포하고 야추 대결과 원장은 2차로 미룸 ([ADR 0004](../../adr/0004-chicken-first-rollout.md))
- 2026-09-23: 시간 고르기는 직접 만들지 않고 디스코드 기본 투표를 쓴다. 이를 위해 discordgo v0.29로 올림 ([ADR 0002](../../adr/0002-discordgo-v029-poll.md))
- 2026-09-23: 규칙은 순수 패키지에 두고, 판 상태는 공용 session.Registry로 관리한다 ([ADR 0003](../../adr/0003-pure-core-shared-session.md))
- 2026-09-28: 1·2단계를 exec-plan 없이 먼저 시작한 것을 발견함 (AGENTS.md 작업 시작 규칙 위반). 이 문서를 소급해서 작성함
- 2026-10-15: 배포 대상을 t4g.nano로 확정 (월 약 $8). Codex 무료 플랜이라 검증은 Claude 서브에이전트가 맡고, 서브에이전트에는 가벼운 모델을 지정함 (profile.md)
- 2026-10-15: 하루 마무리 때 `docs/handoff.md`를 갱신하고 push 함 (AGENTS.md 작업 규칙)
- 2026-10-16: `session.Registry`는 인스턴스 하나에 상태를 `any`로 담는다. 유저 배타성은 기능을 가로질러 적용된다 (ADR 0003 해석, 3단계 완료 기준)
- 2026-10-16: 방치 시간은 `Start` 때 고정하고 `Touch(msgID)`는 인자로 시간을 받지 않는다. 대결(2차)에서 수락 대기와 본게임의 방치 시간이 달라야 하면 그때 추가한다
- 2026-10-16: 밈 트리거를 변형(줴줴이야, 체재희야, 제제이야, 자모 분리)까지 넓힘. 오작동 여부는 1차 배포 후 관찰. 쿨다운은 순수 패키지 `quiet/`로 분리 (사용자 결정)
- 2026-11-19: 마피아MCP 검토 반영. (1) 메시지 원문 로그 제거. (2) 하네스·문서는 지금 규모(6명, 운영 코드 약 800줄)에 비해 충분하므로 **새 절차·검사를 더 늘리지 않는다**. 그래서 ADR 0005의 store PR 템플릿 체크박스는 만들지 않는다(리뷰 요청만으로 충분, 기술 부채 줄 삭제). (3) 문장 수집에 막히지 않은 5·6단계 순수 로직을 먼저 진행한다
- 2026-11-19: 리스크 검토 반영. profile의 사실과 다른 두 줄 수정(토큰 검사 없음, 구현 병렬화 조건). 투표 시간 단위·권한, recover/Restart, `.prev` 되돌리기, EC2 토큰 파일, AWS 리전 확인을 완료 기준에 넣음. nudge 테스트 데이터는 초안으로 먼저 진행(사용자 결정)
- 2026-11-19: W0~W3 완료. 병렬 에이전트 4개(A·B·C 워크트리, F 공유) → 통합 → reviewer 리뷰. 리뷰 반영: [야추로 정하기] 동시 클릭은 참가로 처리, 응답 없는 버튼 경로는 조용히 ack, 끝난 판 버튼은 내용은 두고 버튼만 끔, 배포 재시작 확인을 크래시 루프까지 잡도록, `.env` 업로드를 처음부터 600으로. 나머지는 "알려진 한계"로 기록
