#!/usr/bin/env bash
# setup-dev-billing.sh — Environment and prerequisite checks for Creem billing flows.
# Meant to be sourced by billing tasks so exports reach every Goreman process.
set -euo pipefail

source .mise/setup-dev.sh

if [ -z "${BEEBUZZ_BILLING_NGROK_URL:-}" ]; then
	echo "ERROR: Set BEEBUZZ_BILLING_NGROK_URL in your local .env before starting billing development." >&2
	exit 1
fi

if ! command -v ngrok >/dev/null 2>&1; then
	echo "ERROR: ngrok is required. Install it and authenticate with your ngrok account first." >&2
	exit 1
fi

case "$BEEBUZZ_BILLING_NGROK_URL" in
	https://*) ;;
	*)
		echo "ERROR: BEEBUZZ_BILLING_NGROK_URL must be an HTTPS URL." >&2
		exit 1
		;;
esac

export BEEBUZZ_LOG_FILE="${BEEBUZZ_LOG_FILE:-$PWD/data/billing-e2e.log}"
