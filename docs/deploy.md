# 배포 런북 (EC2 t4g.nano)

서버 만들기와 등록은 사용자가 **AWS 에이전트**에게 맡긴다. 이 문서는 그 에이전트(또는 사람)에게 그대로 넘기는 지시서다. 레포가 주는 것은 아래 세 파일뿐이다.

| 파일 | 역할 |
|---|---|
| `make dist` → `dist/zzezze-bot` | linux/arm64 정적 바이너리 (CGO 없음) |
| [`deploy/zzezze-bot.service`](../deploy/zzezze-bot.service) | systemd 유닛 |
| [`scripts/install.sh`](../scripts/install.sh) | 서버에서 바이너리를 원자적으로 바꾸고 재시작 |

## 서버 조건

- 인스턴스: t4g.nano (arm64), 리전 ap-northeast-2(서울), Amazon Linux 2023 또는 Ubuntu 24.04 arm64.
- 들어오는 포트: **없음**. 봇은 디스코드로 나가는 HTTPS(443)만 쓴다. 보안 그룹 인바운드는 비워 둔다.
- 접속: SSH 키 대신 **SSM Session Manager**를 쓴다. 인스턴스 역할에 `AmazonSSMManagedInstanceCore` 정책을 붙이면 SSM 에이전트로 명령을 보낼 수 있다(Amazon Linux 2023/Ubuntu AMI에는 SSM 에이전트가 기본 설치돼 있다).
- 비용 근거: [docs/harness/profile.md](harness/profile.md) "비용".

## 처음 한 번 (서버에서 root)

```bash
useradd --system --home /opt/zzezze-bot --shell /usr/sbin/nologin zzezze
install -d -o zzezze -g zzezze -m 0750 /opt/zzezze-bot
```

`.env`를 만든다. 변수 목록의 원천은 [`.env.example`](../.env.example)이다. **토큰은 채팅·로그·커밋에 남기지 않는다.** 가능하면 SSM Parameter Store(SecureString)에 넣어 두고 서버에서 꺼내 쓴다.

```bash
install -o zzezze -g zzezze -m 0600 /dev/null /opt/zzezze-bot/.env
# 편집기로 DISCORD_TOKEN=... 을 채운다
```

systemd 유닛을 설치한다(레포의 `deploy/zzezze-bot.service`를 서버로 옮긴 뒤).

```bash
install -m 0644 zzezze-bot.service /etc/systemd/system/zzezze-bot.service
systemctl daemon-reload
systemctl enable zzezze-bot
```

## 배포할 때마다

1. 로컬에서 검사하고 빌드한다.
   ```bash
   make check && make dist
   ```
2. `dist/zzezze-bot`과 `scripts/install.sh`를 서버의 `/tmp/`로 옮긴다(S3 경유 + SSM Run Command, 또는 `scp`).
3. 서버에서 실행한다.
   ```bash
   sudo bash /tmp/install.sh /tmp/zzezze-bot
   ```
   `zzezze-bot.new`로 복사한 뒤 `mv`로 바꿔 끼우고 `systemctl restart` 한다. 재시작할 때 진행 중인 야추 판은 버튼이 꺼지고 재시작 안내가 뜬다.

## 확인

```bash
systemctl status zzezze-bot
journalctl -u zzezze-bot -f
```

로그에 `로그인 완료: ...`가 보이면 된다. 디스코드에서 "야추겜 ㄱㄱ"와 "치킨 먹자"를 쳐서 판과 버튼이 뜨는지 본다.

## 디스코드 쪽 준비 (한 번)

- Developer Portal → Bot → **MESSAGE CONTENT INTENT** 켜기.
- 봇 초대 권한: 메시지 보기, 메시지 보내기, 메시지 기록 보기, 투표 보내기(Send Polls).

## 되돌리기

바로 전 바이너리를 다시 `install.sh`로 설치한다. 1차는 DB가 없어서 되돌릴 데이터가 없다.
