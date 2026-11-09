# Handoff

> 하루 작업이 끝날 때마다 이 파일을 **덮어써서** 갱신한다. 지난 기록은 `git log -p docs/handoff.md`로 본다.
> 새 세션은 이 파일 → `docs/exec-plans/active/` 순서로 읽고 시작한다.

- 마지막 갱신: 2026-11-09
- 브랜치: `claude/great-albattani-dtj07o` (PR #1은 main에 병합됨)
- 진행 중 계획: [phase1-chicken.md](exec-plans/active/phase1-chicken.md) — 코드는 끝, 수동 확인·배포만 남음

## 오늘 한 것

- **방 구분 폐지** ([ADR 0006](adr/0006-no-room-split.md)): 비핵심 인원이 강퇴되어 모든 채널에서 같은 기능을 full 톤으로 켠다. `config.Default` + 빈 `config.Overrides`.
- 1차 3~7단계 코드를 TDD로 끝냈다.
  - `session/`: `Registry`(Start/Join/Touch/Finish 1회/CloseAll/유저 역인덱스, `-race` 경합 테스트), `Throttle`(쿨다운, KST 자정까지 멈춤)
  - `yacht/quick.go`: 최저점 꼴찌, 최저 동점자만 재굴림 최대 3라운드, 그다음 제비뽑기
  - `features/`: 트리거 판정, 시간 슬롯(22:30/23:30 경계 테스트), 야추 판 화면, 버튼 처리, 기본 투표 게시, 종료 정리
  - `main.go`: 핸들러 연결, SIGTERM → 판 정리(5초) → 게이트웨이 종료
- 배포 준비: `make dist`(arm64 바이너리), `deploy/zzezze-bot.service`, `scripts/install.sh`(원자적 교체), [docs/deploy.md](deploy.md) 런북.
- **야추 분리** ([ADR 0007](adr/0007-yacht-separate-from-food.md)): "야추겜 ㄱㄱ"로 바로 야추 판을 연다. 음식 얘기(치킨·곱창·양꼬치 등 `features.Foods`)에는 "약속 언제 잡을래?" + 시간 투표로 반응하고, 투표 제목에 음식 이름이 들어간다.
- 밈 반응 확장: "줴줴이야", "조줴줴이야", "최재희야".
- 롤 파티 모집(5인큐, 증바람 등) 아이디어를 설계 문서 "나중에" 절에 적었다.

## 다음 할 것

1. **서버 등록과 배포**: AWS 에이전트에게 [docs/deploy.md](deploy.md)를 넘겨 EC2를 만들고 설치한다.
2. 디스코드에서 손으로 확인: 밈 반응, "야추겜 ㄱㄱ" 판(혼자면 취소 / 2명 이상), 음식 얘기 → 약속 투표, 시간 투표, `[🔕 오늘 그만]`, 재시작 시 판 정리.
3. 1주 관찰 뒤 2차(야추 대결·포인트) 또는 롤 파티 모집 중 우선순위를 정한다. 어느 쪽이든 exec-plan부터 쓴다.

## 막힌 것 / 사용자에게 필요한 것

- `features/testdata/nudge_hit.txt`, `nudge_miss.txt`는 예시 문장이다. 실제 방 문장으로 바꿔야 한다.
- `voice/lines.go` 대사는 여전히 초안이다.
- 봇 초대 권한에 **투표 보내기(Send Polls)** 가 있어야 시간 투표가 올라간다.

## 새로 내린 결정

- 방 구분 폐지 (ADR 0006)
- 야추는 "야추겜 ㄱㄱ"로 분리, 음식 얘기는 약속 잡기 (ADR 0007)
- EC2 서버 등록은 사용자가 AWS 에이전트로 하고, 레포는 바이너리·유닛·설치 스크립트·런북만 제공한다.
- 메뉴는 `features/triggers.go`의 `Foods` 목록 한 곳에서 관리한다. 빠진 메뉴가 있으면 거기에 추가한다.
- 밈 자동 반응 쿨다운은 채널별 3분으로 시작한다(대화 트리거는 설계대로 30분).
