#!/bin/sh
set -eu

domain="${BEEBUZZ_DOMAIN:-}"

case "$domain" in
	''|.*|*.|*..*|*[!A-Za-z0-9.-]*)
		echo "BEEBUZZ_DOMAIN must be a valid hostname" >&2
		exit 1
		;;
esac

printf "window.__BEEBUZZ_CONFIG__ = { domain: '%s' };\n" "$domain" > /srv/build/config.js

exec /usr/bin/caddy run --config /etc/caddy/Caddyfile --adapter caddyfile
