# 쮀쮀이야 봇

BM 증바람 서버용 디스코드 봇. 봇 1개가 채널 ID를 보고 방마다 다른 기능을 켠다.
2번방에서 "치킨 시킬까?"가 나오면 주사위로 치킨값 담당을 정해 주는, 방에 같이 있는 **여섯 번째 멤버**가 목표다.

| 알고 싶은 것 | 볼 곳 |
|---|---|
| 무엇을 왜 만드나 | [docs/design/six-member.md](docs/design/six-member.md) |
| 지금 어디까지 했나, 다음은 뭔가 | [docs/handoff.md](docs/handoff.md), [docs/exec-plans/active/](docs/exec-plans/active/) |
| 명령어, 폴더 규칙, 작업 규칙 | [AGENTS.md](AGENTS.md) |
| 어떤 사실을 어디서 관리하나 | [docs/sot.md](docs/sot.md) |

## 처음 실행하기

1. Go 설치. 필요한 버전은 [go.mod](go.mod)의 `go` 줄을 따른다.
2. Discord Developer Portal에서 할 일
   - 애플리케이션 → Bot → Reset Token으로 **봇 토큰**을 받는다.
   - 같은 Bot 탭에서 **MESSAGE CONTENT INTENT**를 켠다 (자동 반응에 필요).
3. 토큰을 설정한다. 필요한 변수 목록은 [.env.example](.env.example)이 원천이다.
   ```bash
   cp .env.example .env   # .env 를 열어 값을 채운다. .env 는 커밋되지 않는다.
   ```
4. 검사와 실행
   ```bash
   make check   # 포맷, 테스트, 의존 방향 검사 등 전체
   make run
   ```

정상이면 콘솔에 `로그인 완료: ...`가 뜨고, 등록된 방에 메시지가 올라올 때마다 방 이름과 켜진 기능이 로그에 찍힌다. 등록되지 않은 채널은 무시한다.

## 방 추가/변경

[config/channels.go](config/channels.go)의 `Channels` 맵만 고친다. 거기 있는 기존 항목을 그대로 따라 쓰면 된다(기능은 `config.Feature` 상수, 말투는 `Tone`). `make check`가 이름이나 톤이 빠진 항목을 잡아낸다.

채널 ID는 디스코드 설정 → 고급 → 개발자 모드 ON → 채널 우클릭 → "채널 ID 복사"로 얻는다.

## 배포

EC2 t4g.nano(arm64)에서 systemd로 돌린다. 빌드 명령과 배포 절차는 [설계 문서의 "배포" 절](docs/design/six-member.md#배포)이 원천이다.
