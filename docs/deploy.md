# 배포

대상: EC2 t4g.nano (arm64, Amazon Linux 2023), systemd. 구성 근거와 비용은 [harness/profile.md](harness/profile.md), 종료 순서는 [design/six-member.md](design/six-member.md)의 "배포"와 "공통 규칙".

## 1. 사전 확인

- AWS Settings > View all projects > Overview > Additional Info > Region 에서 프로젝트 리전을 확인한다. 이 계정은 리전 하나로 고정돼 있고(계정 주소로 정해짐) 다른 리전에는 만들 수 없다. 설계는 서울(ap-northeast-2)을 가정하므로 다르면 비용·지연을 다시 본다. 플랜(무료/유료)과 지출 한도도 확인한다.
- 인스턴스: t4g.nano, Amazon Linux 2023 arm64.
- 보안 그룹: 인바운드는 SSH(내 IP)만 연다. 봇은 아웃바운드(디스코드 접속)만 필요하다.

## 2. 서버 1회 준비

서버에 SSH 로 들어가서 실행한다. 아래 `ec2-user` 는 배포용 SSH 계정이다.

```bash
# 로그인 불가 전용 계정
sudo useradd --system --no-create-home --shell /usr/sbin/nologin zzezze

# 배포 디렉터리: 배포 계정(ec2-user)이 바이너리를 올리고, zzezze 는 읽고 실행만 한다.
sudo mkdir -p /opt/zzezze-bot
sudo chown ec2-user:zzezze /opt/zzezze-bot
sudo chmod 755 /opt/zzezze-bot
```

로컬에서 `.env` 를 올린다([.env.example](../.env.example) 참고). 토큰은 systemd 가 root 로 읽으므로 서비스 계정이 읽을 필요는 없다.

```bash
# umask 077 로 처음부터 600 으로 만든다(scp 후 chmod 하면 잠깐 다른 계정이 읽을 수 있다).
ssh <ssh-host> 'umask 077; cat > /opt/zzezze-bot/.env' < .env
```

unit 설치와 enable (로컬 레포에서):

```bash
scp deploy/zzezze-bot.service <ssh-host>:/tmp/
ssh <ssh-host> 'sudo install -o root -g root -m 644 /tmp/zzezze-bot.service /etc/systemd/system/ && rm /tmp/zzezze-bot.service && sudo systemctl daemon-reload && sudo systemctl enable zzezze-bot'
```

enable 만 하고 시작은 첫 배포가 한다.

## 3. 디스코드 포털 확인

Developer Portal 에서 확인한다.

- Bot > Privileged Gateway Intents: **MESSAGE CONTENT INTENT** 켜기
- 봇 권한: 메시지 보내기, 메시지 기록 보기, 투표 보내기(Send Polls)

## 4. 배포 / 되돌리기 / 로그

```bash
scripts/deploy.sh deploy <ssh-host>                 # make check -> arm64 빌드 -> 업로드 -> 교체 -> restart
scripts/deploy.sh deploy <ssh-host> --skip-check    # 검사 생략
scripts/deploy.sh deploy <ssh-host> --dry-run       # 실행할 명령만 출력
scripts/deploy.sh rollback <ssh-host>               # 직전 바이너리(.prev)로 복구
ssh <ssh-host> 'journalctl -u zzezze-bot -f'        # 로그 보기
```

배포 때 기존 바이너리는 `zzezze-bot.prev` 로 보존되고, 재시작 뒤 active 가 아니면 최근 로그와 rollback 명령이 출력된다. `.prev` 는 직전 한 개만 남는다(rollback 하면 소진된다).

## 5. 토큰이 유출됐을 때

1. Developer Portal > Bot > **Reset Token** 으로 새 토큰을 발급한다(옛 토큰은 즉시 무효).
2. 서버 `/opt/zzezze-bot/.env` 의 `DISCORD_TOKEN` 을 새 값으로 교체한다(`chmod 600` 유지).
3. `sudo systemctl restart zzezze-bot`
