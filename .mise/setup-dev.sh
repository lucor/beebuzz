#!/usr/bin/env bash
# setup-dev.sh — Pre-dev setup for local HTTPS development.
# Meant to be sourced (not executed) so env vars propagate to the caller.
#
# What it does:
#   1. Detects LAN IP and derives a lancert.dev domain (e.g. 192-168-1-50.lancert.dev)
#   2. Optionally persists BEEBUZZ_DOMAIN to .env (asks for confirmation)
#   3. Fetches TLS certs from lancert.dev if not cached locally
#
# lancert.dev provides real Let's Encrypt wildcard certs for RFC 1918 IPs,
# so *.192-168-1-50.lancert.dev covers all subdomains (api., hive., push., hook.).
#
# Cert refresh: delete .mise/certs/ and re-run "mise run dev".
#
# Requirements: bash, curl
# Supports: macOS, Linux
set -euo pipefail

LANCERT_URL="https://lancert.dev"
CERT_DIR=".mise/certs"

# --- Detect LAN IP (Linux + macOS) ---
detect_local_ip() {
  if command -v ip &>/dev/null; then
    ip route get 1.1.1.1 | awk '{for(i=1;i<=NF;i++) if($i=="src") {print $(i+1); exit}}'
  else
    local iface
    iface=$(route -n get default 2>/dev/null | awk '/interface:/{print $2}')
    ipconfig getifaddr "$iface" 2>/dev/null
  fi
}

# --- Resolve domain ---
DOMAIN="${BEEBUZZ_DOMAIN:-}"

if [ -z "$DOMAIN" ] || [[ "$DOMAIN" != *.lancert.dev ]]; then
  LOCAL_IP=$(detect_local_ip)
  if [ -z "${LOCAL_IP:-}" ]; then
    echo "ERROR: could not detect LAN IP" >&2
    return 1
  fi
  LABEL=$(echo "$LOCAL_IP" | tr '.' '-')
  DOMAIN="${LABEL}.lancert.dev"

  echo "[setup-dev] Detected LAN IP: $LOCAL_IP"
  echo "[setup-dev] BEEBUZZ_DOMAIN=$DOMAIN"

  # Ask before writing to .env
  CURRENT=$(grep "^BEEBUZZ_DOMAIN=" .env 2>/dev/null | cut -d= -f2 || echo "")
  if [ "$CURRENT" != "$DOMAIN" ]; then
    read -rp "[setup-dev] Update BEEBUZZ_DOMAIN in .env? (current: ${CURRENT:-<unset>}) [Y/n] " answer
    if [[ "${answer:-Y}" =~ ^[Yy]$ ]]; then
      if [ -f .env ] && grep -q "^BEEBUZZ_DOMAIN=" .env; then
        sed -i.bak "s/^BEEBUZZ_DOMAIN=.*/BEEBUZZ_DOMAIN=${DOMAIN}/" .env && rm -f .env.bak
      else
        echo "BEEBUZZ_DOMAIN=${DOMAIN}" >> .env
      fi
      echo "[setup-dev] Updated .env"
    else
      echo "[setup-dev] Skipped .env update (using $DOMAIN for this session only)"
    fi
  fi
fi

IP=$(echo "${DOMAIN%.lancert.dev}" | tr '-' '.')
export BEEBUZZ_DOMAIN="$DOMAIN"
export VITE_BEEBUZZ_DOMAIN="$DOMAIN"

echo "[setup-dev] Domain: $DOMAIN"

# --- Fetch certs if not cached locally ---
mkdir -p "$CERT_DIR"

if [ -f "$CERT_DIR/fullchain.pem" ] && [ -f "$CERT_DIR/privkey.pem" ]; then
  echo "[setup-dev] Certs already present, skipping fetch. Delete $CERT_DIR to force refresh."
else
  # Check if cert exists on lancert
  STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$LANCERT_URL/certs/$IP/fullchain.pem")

  if [ "$STATUS" = "404" ]; then
    echo "[setup-dev] Requesting cert for $IP from lancert.dev..."
    curl -s -o /dev/null -X POST "$LANCERT_URL/certs/$IP"

    echo "[setup-dev] Waiting for certificate issuance..."
    for _ in $(seq 1 60); do
      STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$LANCERT_URL/certs/$IP/fullchain.pem")
      if [ "$STATUS" = "200" ]; then
        break
      fi
      printf "."
      sleep 10
    done
    echo ""
  fi

  if [ "$STATUS" != "200" ]; then
    echo "ERROR: could not fetch certs from lancert.dev (status: $STATUS)" >&2
    return 1
  fi

  curl -sf "$LANCERT_URL/certs/$IP/fullchain.pem" -o "$CERT_DIR/fullchain.pem"
  curl -sf "$LANCERT_URL/certs/$IP/privkey.pem" -o "$CERT_DIR/privkey.pem"
  echo "[setup-dev] Certs saved to $CERT_DIR/"
fi
