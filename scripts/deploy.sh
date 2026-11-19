#!/usr/bin/env bash
# zzezze-bot 배포 스크립트 (로컬 Mac 에서 실행).
#
#   scripts/deploy.sh deploy   <ssh-host> [--skip-check] [--dry-run]
#   scripts/deploy.sh rollback <ssh-host> [--dry-run]
#
# ssh-host: ~/.ssh/config 별칭 또는 user@host
# 환경변수: REMOTE_DIR(기본 /opt/zzezze-bot), SERVICE(기본 zzezze-bot)
set -euo pipefail

REMOTE_DIR="${REMOTE_DIR:-/opt/zzezze-bot}"
SERVICE="${SERVICE:-zzezze-bot}"
BIN="zzezze-bot"
NEW="zzezze-bot.new"

usage() {
  cat <<USAGE
사용법:
  $0 deploy   <ssh-host> [--skip-check] [--dry-run]
  $0 rollback <ssh-host> [--dry-run]

  --skip-check  빌드 전 make check 를 생략한다
  --dry-run     실제로 실행하지 않고 실행할 명령만 출력한다
USAGE
}

if [ $# -lt 2 ]; then
  usage
  exit 2
fi

cmd="$1"
host="$2"
shift 2

skip_check=0
dry_run=0
for arg in "$@"; do
  case "$arg" in
    --skip-check) skip_check=1 ;;
    --dry-run) dry_run=1 ;;
    *) echo "알 수 없는 옵션: $arg" >&2; usage; exit 2 ;;
  esac
done

# 레포 루트에서 실행한다.
cd "$(dirname "$0")/.."

# 실행하거나(dry-run 이면 출력만) 한다.
run() {
  if [ "$dry_run" -eq 1 ]; then
    echo "[dry-run] $*"
  else
    "$@"
  fi
}

# 원격에서 실행할 셸 스크립트를 ssh 로 보낸다. 값은 인자($1..)로 넘겨 인용 문제를 피한다.
remote() {
  local script="$1"
  if [ "$dry_run" -eq 1 ]; then
    echo "[dry-run] ssh $host bash -s -- $REMOTE_DIR $SERVICE <<'REMOTE'"
    echo "$script"
    echo "REMOTE"
  else
    ssh "$host" bash -s -- "$REMOTE_DIR" "$SERVICE" <<<"$script"
  fi
}

# 재시작 후 active 확인. 실패하면 로그와 rollback 안내를 출력하고 종료코드 1.
REMOTE_RESTART='
set -u
dir="$1"; svc="$2"
sudo systemctl restart "$svc"
sleep 3
if ! systemctl is-active --quiet "$svc"; then
  echo "서비스가 active 가 아닙니다. 최근 로그:" >&2
  sudo journalctl -u "$svc" -n 20 --no-pager >&2 || true
  exit 1
fi
echo "서비스 active"
'

rollback_hint() {
  echo "되돌리려면: $0 rollback $host" >&2
}

do_deploy() {
  if [ "$skip_check" -eq 0 ]; then
    run make check
  else
    echo "make check 생략(--skip-check)"
  fi

  # 끝나면(실패 포함) 로컬 산출물을 지운다.
  trap 'rm -f "$NEW"' EXIT

  run env GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o "$NEW" .
  run scp "$NEW" "$host:$REMOTE_DIR/$NEW"

  # 기존 바이너리 보존(.prev) -> 같은 디렉터리 안에서 mv 로 원자적 교체.
  remote '
set -eu
dir="$1"
cd "$dir"
if [ -f zzezze-bot ]; then
  cp -p zzezze-bot zzezze-bot.prev
fi
chmod 755 zzezze-bot.new
mv zzezze-bot.new zzezze-bot
'
  if ! remote "$REMOTE_RESTART"; then
    rollback_hint
    exit 1
  fi
  if [ "$dry_run" -eq 1 ]; then echo "dry-run 끝(실제 변경 없음)"; else echo "배포 완료: $host"; fi
}

do_rollback() {
  remote '
set -eu
dir="$1"
cd "$dir"
if [ ! -f zzezze-bot.prev ]; then
  echo "되돌릴 zzezze-bot.prev 가 없습니다." >&2
  exit 1
fi
mv zzezze-bot.prev zzezze-bot
'
  remote "$REMOTE_RESTART"
  if [ "$dry_run" -eq 1 ]; then echo "dry-run 끝(실제 변경 없음)"; else echo "롤백 완료: $host"; fi
}

case "$cmd" in
  deploy) do_deploy ;;
  rollback) do_rollback ;;
  *) usage; exit 2 ;;
esac
