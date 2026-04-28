#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

ENV_FILE="${ENV_FILE:-.env.quickstart-demo}"

if [ ! -f "$ENV_FILE" ]; then
  echo "missing $ENV_FILE" >&2
  exit 1
fi

set -a
# shellcheck disable=SC1090
source "$ENV_FILE"
set +a

detect_local_ip() {
  if command -v ip >/dev/null 2>&1; then
    ip route get 1.1.1.1 | awk '{for(i=1;i<=NF;i++) if($i=="src") {print $(i+1); exit}}'
    return
  fi

  local iface
  iface="$(route -n get default 2>/dev/null | awk '/interface:/{print $2}')"
  ipconfig getifaddr "$iface" 2>/dev/null
}

if [ -z "${BEEBUZZ_DOMAIN:-}" ] || [[ "$BEEBUZZ_DOMAIN" != *.lancert.dev ]]; then
  LOCAL_IP="$(detect_local_ip)"
  if [ -z "${LOCAL_IP:-}" ]; then
    echo "could not detect LAN IP for lancert.dev domain" >&2
    exit 1
  fi
  export BEEBUZZ_DOMAIN="$(echo "$LOCAL_IP" | tr '.' '-').lancert.dev"
fi

export VITE_BEEBUZZ_DOMAIN="$BEEBUZZ_DOMAIN"

if [ -z "${BEEBUZZ_VAPID_PRIVATE_KEY:-}" ] || [ -z "${BEEBUZZ_VAPID_PUBLIC_KEY:-}" ]; then
  eval "$(mise x -- go run ./cmd/beebuzz-server vapid generate)"
  export BEEBUZZ_VAPID_PRIVATE_KEY
  export BEEBUZZ_VAPID_PUBLIC_KEY
fi

rm -rf "${BEEBUZZ_DB_DIR:?}" "${BEEBUZZ_ATTACHMENTS_DIR:?}"
mkdir -p "$BEEBUZZ_DB_DIR" "$BEEBUZZ_ATTACHMENTS_DIR" "${DEMO_OUTPUT_DIR:-docs/assets/readme}"

# setup-dev.sh fetches/reuses lancert.dev certificates and exports Caddy/Vite domain env.
# It will not prompt because BEEBUZZ_DOMAIN is already a lancert.dev host.
# shellcheck disable=SC1091
source .mise/setup-dev.sh

cleanup() {
  if [ -n "${STACK_PID:-}" ]; then
    kill "$STACK_PID" >/dev/null 2>&1 || true
    wait "$STACK_PID" >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT

(
  cd .mise
  mise x -- goreman -f Procfile start
) &
STACK_PID=$!

SITE_URL="https://${BEEBUZZ_DOMAIN}"
HIVE_URL="https://hive.${BEEBUZZ_DOMAIN}"
API_URL="https://api.${BEEBUZZ_DOMAIN}"

echo "[quickstart-demo] site: $SITE_URL"
echo "[quickstart-demo] hive: $HIVE_URL"
echo "[quickstart-demo] api:  $API_URL"

mise x -- node web/tests/demo/quickstart-demo.mjs
