module go.beebuzz.app/beebuzz

go 1.25.0

require (
	filippo.io/age v1.3.1
	github.com/SherClockHolmes/webpush-go v1.4.0
	github.com/getsentry/sentry-go v0.47.0
	github.com/go-chi/chi/v5 v5.3.0
	github.com/go-chi/httprate v0.15.0
	github.com/golang-migrate/migrate/v4 v4.19.1
	github.com/google/uuid v1.6.0
	github.com/jmoiron/sqlx v1.4.0
	github.com/joho/godotenv v1.5.1
	github.com/tidwall/gjson v1.19.0
	golang.org/x/crypto v0.53.0
	modernc.org/sqlite v1.52.0
	rsc.io/qr v0.2.0
)

require (
	filippo.io/hpke v0.4.0 // indirect
	github.com/BurntSushi/toml v1.4.1-0.20240526193622-a339e1f7089c // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/golang-jwt/jwt/v5 v5.3.1 // indirect
	github.com/klauspost/cpuid/v2 v2.3.0 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-sqlite3 v1.14.34 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.0 // indirect
	github.com/zeebo/xxh3 v1.1.0 // indirect
	golang.org/x/exp/typeparams v0.0.0-20231108232855-2478ac86f678 // indirect
	golang.org/x/mod v0.36.0 // indirect
	golang.org/x/sync v0.21.0 // indirect
	golang.org/x/sys v0.46.0 // indirect
	golang.org/x/telemetry v0.0.0-20260508192327-42602be52be6 // indirect
	golang.org/x/text v0.38.0 // indirect
	golang.org/x/tools v0.45.0 // indirect
	golang.org/x/vuln v1.3.0 // indirect
	honnef.co/go/tools v0.7.0 // indirect
	modernc.org/libc v1.72.3 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.11.0 // indirect
)

tool (
	golang.org/x/tools/cmd/goimports
	golang.org/x/vuln/cmd/govulncheck
	honnef.co/go/tools/cmd/staticcheck
)
