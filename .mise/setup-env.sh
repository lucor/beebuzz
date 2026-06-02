#!/usr/bin/env bash
# setup-env.sh — One-time environment setup run by `mise run setup`.
# Idempotent: safe to run multiple times.
#
# What it does:
#   1. Creates .env from .env.example if missing
#   2. Generates VAPID keys if missing
#   3. Prompts for bootstrap admin email if missing
#
# Requirements: bash
# Supports: macOS, Linux
set -euo pipefail

# --- Create .env from template if missing ---
if [ ! -f .env ]; then
  if [ ! -f .env.example ]; then
    echo "ERROR: .env.example not found" >&2
    exit 1
  fi
  cp .env.example .env
  echo "[setup] Created .env from .env.example"
fi

# --- Generate VAPID keys if missing ---
VAPID_PRIVATE=$(grep "^BEEBUZZ_VAPID_PRIVATE_KEY=" .env 2>/dev/null | cut -d= -f2 || echo "")
VAPID_PUBLIC=$(grep "^BEEBUZZ_VAPID_PUBLIC_KEY=" .env 2>/dev/null | cut -d= -f2 || echo "")

if [ -z "$VAPID_PRIVATE" ] || [ -z "$VAPID_PUBLIC" ]; then
  echo "[setup] Generating VAPID keys..."
  VAPID_KEYS=$(go run . vapid generate)
  echo "$VAPID_KEYS" >> .env
  echo "[setup] VAPID keys saved to .env"
else
  echo "[setup] VAPID keys already present"
fi

# --- Prompt for bootstrap admin email if missing ---
BOOTSTRAP_EMAIL=$(grep "^BEEBUZZ_BOOTSTRAP_ADMIN_EMAIL=" .env 2>/dev/null | cut -d= -f2 || echo "")
DEFAULT_ADMIN_EMAIL="admin@beebuzz.local"

if [ -z "$BOOTSTRAP_EMAIL" ]; then
  read -rp "[setup] Bootstrap admin email [${DEFAULT_ADMIN_EMAIL}] " answer
  BOOTSTRAP_EMAIL="${answer:-${DEFAULT_ADMIN_EMAIL}}"
  echo "BEEBUZZ_BOOTSTRAP_ADMIN_EMAIL=${BOOTSTRAP_EMAIL}" >> .env
  echo "[setup] Set BEEBUZZ_BOOTSTRAP_ADMIN_EMAIL=${BOOTSTRAP_EMAIL} in .env"
else
  echo "[setup] BEEBUZZ_BOOTSTRAP_ADMIN_EMAIL=${BOOTSTRAP_EMAIL}"
fi
