#!/usr/bin/env bash
set -euo pipefail

for endpoint in "https://api.${BEEBUZZ_DOMAIN}/health" "https://dashboard.${BEEBUZZ_DOMAIN}/auth" "http://127.0.0.1:8025/api/v1/messages"; do
	ready=0
	for _ in $(seq 1 30); do
		if curl --silent --show-error --fail --insecure "$endpoint" >/dev/null 2>&1; then
			ready=1
			break
		fi
		sleep 1
	done
	if [ "$ready" -ne 1 ]; then
		echo "ERROR: billing service did not become ready: $endpoint" >&2
		exit 1
	fi
done

pnpm -C web exec playwright test tests/e2e/billing-checkout.spec.ts

# The E2E process is part of the Goreman Procfile. Stop the process group after
# a successful run so the single task returns instead of leaving the stack up.
kill -TERM "$PPID" 2>/dev/null || true
