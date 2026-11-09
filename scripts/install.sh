#!/usr/bin/env bash
# EC2 서버에서 root 로 실행한다. 새 바이너리를 원자적으로 바꿔 끼우고 재시작한다.
#   sudo bash install.sh /tmp/zzezze-bot
# 처음 설치(유저·폴더·.env·systemd 유닛)는 docs/deploy.md "처음 한 번" 절을 먼저 한다.
set -euo pipefail

src="${1:?사용법: install.sh <새 바이너리 경로>}"
dir=/opt/zzezze-bot

[[ -f "$dir/.env" ]] || { echo "✘ $dir/.env 가 없음. docs/deploy.md '처음 한 번'을 먼저 하세요."; exit 1; }

# 같은 파일시스템 안에서 mv 해야 원자적으로 바뀐다.
install -o zzezze -g zzezze -m 0755 "$src" "$dir/zzezze-bot.new"
mv -f "$dir/zzezze-bot.new" "$dir/zzezze-bot"

systemctl restart zzezze-bot
sleep 3
if systemctl is-active --quiet zzezze-bot; then
	echo "✔ zzezze-bot 실행 중"
	journalctl -u zzezze-bot -n 5 --no-pager
else
	echo "✘ zzezze-bot 이 뜨지 않음"
	journalctl -u zzezze-bot -n 30 --no-pager
	exit 1
fi
