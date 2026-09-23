#!/bin/sh
set -eu

domain="${BEEBUZZ_DOMAIN:-}"

case "$domain" in
	''|.*|*.|*..*|*[!A-Za-z0-9.-]*)
		echo "BEEBUZZ_DOMAIN must be a valid hostname" >&2
		exit 1
		;;
esac

for app in dashboard hive; do
	printf "window.__BEEBUZZ_CONFIG__ = { domain: '%s' };\n" "$domain" > "/srv/build/$app/config.js"
done

exec /usr/bin/caddy run --config /etc/caddy/Caddyfile --adapter caddyfile
