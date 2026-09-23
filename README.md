# 쮀쮀이야 봇 (1단계 골격)

BM 증바람 서버용 디스코드 봇. 봇 1개로 채널 ID를 인식해 방마다 다른 기능을 실행하는 구조.
이 단계의 목표는 **봇이 켜지고, 메시지가 어느 방에서 왔는지 인식해 콘솔에 찍는 것**까지다.

## 구성

```
zzezze-bot/
├─ main.go               # 진입점: 세션 생성, 인텐트 설정, 이벤트 핸들러
├─ config/
│  └─ channels.go        # 채널 ID ↔ 기능 매핑 (여기만 고치면 방 추가 가능)
├─ .env.example          # DISCORD_TOKEN 예시
├─ .gitignore
└─ go.mod
```

## 준비물

1. **Go 1.23+** 설치
2. **봇 토큰** — Discord Developer Portal → 애플리케이션 → Bot → Reset Token
3. Developer Portal → Bot 탭에서 **MESSAGE CONTENT INTENT** 켜기 (자동 반응에 필요)

## 로컬 실행

```bash
# 1. 의존성 받기 (discordgo, godotenv + go.sum 생성)
go mod tidy

# 2. 토큰 설정
cp .env.example .env
#   .env 파일을 열어 DISCORD_TOKEN 값을 채운다

# 3. 실행
go run .
```

정상이면 콘솔에 `로그인 완료: 쮀쮀이야 ...` 가 뜨고,
등록된 두 방(#채팅방, #개꿀이즤)에 누가 메시지를 보낼 때마다 이렇게 찍힌다:

```
[채팅방] 니얼굴디크: "쮀쮀이야 ㅋㅋ"  (features=[autoReply moderation yacht])
```

설정에 없는 채널의 메시지는 무시된다.

## 채널 추가/변경

`config/channels.go` 의 `Channels` 맵에 항목만 추가하면 된다. 기능 코드는 건드리지 않는다.

```go
"채널ID": {
    Name:     "표시이름",
    Features: []string{"autoReply", "moderation"},
},
```

채널 ID는 디스코드 설정 → 고급 → 개발자 모드 ON 후, 채널 우클릭 → "채널 ID 복사".

## EC2 배포 (요약)

Go는 단일 바이너리로 컴파일되므로 배포가 간단하다.

```bash
# 1. ARM(t4g) 대상 크로스 컴파일
GOOS=linux GOARCH=arm64 go build -o zzezze-bot .

# 2. EC2로 전송
scp zzezze-bot .env ec2-user@<EC2_IP>:~/zzezze-bot/

# 3. EC2에서 systemd 서비스로 상시 구동 (예: /etc/systemd/system/zzezze-bot.service)
#    [Service]
#    ExecStart=/home/ec2-user/zzezze-bot/zzezze-bot
#    WorkingDirectory=/home/ec2-user/zzezze-bot
#    Restart=always
#    User=ec2-user
#    그 뒤: sudo systemctl enable --now zzezze-bot
```

## 다음 단계

2단계부터 `features/` 아래에 기능 모듈을 추가하고, `main.go` 의 `onMessageCreate` 안
TODO 자리에서 `cfg.Has("autoReply")` 처럼 켜진 기능만 디스패치한다.
순서: 키워드 자동 반응 → 채팅/공지 관리 → 약속 조율(2번방) → 야추 게임.
