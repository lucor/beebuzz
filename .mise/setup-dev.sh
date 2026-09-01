#!/usr/bin/env bash
# setup-dev.sh — Pre-dev setup for local HTTPS development.
# Meant to be sourced (not executed) so env vars propagate to the caller.
#
# What it does:
#   1. Validates that .env exists and required keys are configured
#   2. Validates the lancert.dev hostname configured in BEEBUZZ_DOMAIN
#   3. Resolves the certificate and private key managed by the local Lancert client
#
# The Lancert client stores real Let's Encrypt wildcard certificates locally.
# The configured certificate covers BEEBUZZ_DOMAIN and its BeeBuzz subdomains.
#
# Requirements: bash, lancert certificate issued for BEEBUZZ_DOMAIN
# Supports: macOS, Linux
set -euo pipefail

# --- Validate .env is configured ---
if [ ! -f .env ]; then
  echo "ERROR: .env not found. Run 'mise run setup' first." >&2
  return 1
fi

if [ -z "${BEEBUZZ_VAPID_PRIVATE_KEY:-}" ] || [ -z "${BEEBUZZ_VAPID_PUBLIC_KEY:-}" ]; then
  echo "ERROR: VAPID keys not configured. Run 'mise run setup' first." >&2
  return 1
fi

if [ -z "${BEEBUZZ_BOOTSTRAP_ADMIN_EMAIL:-}" ]; then
  echo "ERROR: BEEBUZZ_BOOTSTRAP_ADMIN_EMAIL not set. Run 'mise run setup' first." >&2
  return 1
fi

DOMAIN="${BEEBUZZ_DOMAIN:-}"
if [ -z "$DOMAIN" ] || [[ "$DOMAIN" != *.lancert.dev ]]; then
  echo "ERROR: BEEBUZZ_DOMAIN must be the hostname issued by the Lancert client (*.lancert.dev)." >&2
  return 1
fi

export BEEBUZZ_DOMAIN="$DOMAIN"
export VITE_BEEBUZZ_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "dev")
export COMMIT_SHA="$VITE_BEEBUZZ_COMMIT"
echo "[setup-dev] Domain: $DOMAIN"

# --- Resolve locally managed Lancert certificate ---
if [ -n "${LANCERT_CONFIG_DIR:-}" ]; then
  CONFIG_DIR="$LANCERT_CONFIG_DIR"
else
  case "$(uname -s)" in
    Darwin)
      CONFIG_DIR="${HOME}/Library/Application Support/lancert"
      ;;
    Linux)
      CONFIG_DIR="${XDG_CONFIG_HOME:-${HOME}/.config}/lancert"
      ;;
    *)
      echo "ERROR: unsupported platform; set LANCERT_CONFIG_DIR explicitly." >&2
      return 1
      ;;
  esac
fi

CERT_DIR="${CONFIG_DIR}/certs/${DOMAIN}"
export BEEBUZZ_TLS_CERT_FILE="${CERT_DIR}/fullchain.pem"
export BEEBUZZ_TLS_KEY_FILE="${CERT_DIR}/privkey.pem"

if [ ! -r "$BEEBUZZ_TLS_CERT_FILE" ] || [ ! -r "$BEEBUZZ_TLS_KEY_FILE" ]; then
  echo "ERROR: Lancert certificate not found for $DOMAIN." >&2
  echo "Run the Lancert client for this machine, then retry. Expected files in: $CERT_DIR" >&2
  return 1
fi

echo "[setup-dev] Lancert certificate: $CERT_DIR"
