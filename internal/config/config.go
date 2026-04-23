package config

import (
	"errors"
	"fmt"
	"net/netip"
	"os"

	"github.com/joho/godotenv"
)

// Environment constants (public)
const (
	EnvDevelopment = "development"
	EnvTest        = "test"
	EnvProduction  = "production"
)

// Env vars
const (
	envPort                = "BEEBUZZ_PORT"
	envDBDir               = "BEEBUZZ_DB_DIR"
	envAttachmentsDir      = "BEEBUZZ_ATTACHMENTS_DIR"
	envDomain              = "BEEBUZZ_DOMAIN"
	envPrivateBeta         = "BEEBUZZ_PRIVATE_BETA"
	envBootstrapAdminEmail = "BEEBUZZ_BOOTSTRAP_ADMIN_EMAIL"
	envEnv                 = "BEEBUZZ_ENV"
	envProxySubnet         = "BEEBUZZ_PROXY_SUBNET"
	envIPHashSalt          = "BEEBUZZ_IP_HASH_SALT"
	envVAPIDPublicKey      = "BEEBUZZ_VAPID_PUBLIC_KEY"
	envVAPIDPrivateKey     = "BEEBUZZ_VAPID_PRIVATE_KEY"
	envRequestIDHeader     = "BEEBUZZ_REQUEST_ID_HEADER"
	envMailerSMTPAddress   = "BEEBUZZ_MAILER_SMTP_ADDRESS"
	envMailerSMTPUser      = "BEEBUZZ_MAILER_SMTP_USER"
	envMailerSMTPPassword  = "BEEBUZZ_MAILER_SMTP_PASSWORD"
	envMailerResendAPIKey  = "BEEBUZZ_MAILER_RESEND_API_KEY"
	envMailerSender        = "BEEBUZZ_MAILER_SENDER"
	envMailerReplyTo       = "BEEBUZZ_MAILER_REPLY_TO"
	envSentryDSN           = "BEEBUZZ_SENTRY_DSN"
)

// defaults
const (
	defaultPort           = "8899"
	defaultDBDir          = "./data/db"
	defaultAttachmentsDir = "./data/attachments"
	defaultDomain         = "example.com"
	defaultEnv            = EnvDevelopment
	defaultMailerSender   = "noreply@example.com"
	defaultMailerReplyTo  = "support@example.com"
)

// Mailer holds mailer configuration.
type Mailer struct {
	SMTPAddress  string // SMTP server address (host:port)
	SMTPUser     string // SMTP username
	SMTPPassword string // SMTP password
	ResendAPIKey string // Resend API key
	Sender       string // Email sender address
	ReplyTo      string // Reply-To address
	SiteURL      string // Site base URL for email links
}

// Config holds the application configuration loaded from environment variables.
type Config struct {
	Domain              string       // Base domain (e.g. "example.com")
	Port                string       // HTTP server port
	DBDir               string       // Directory for the SQLite database file
	AttachmentsDir      string       // Directory for attachment file storage
	URL                 string       // API base URL (https://api.{domain})
	SiteURL             string       // Site base URL (https://{domain})
	PrivateBeta         bool         // Enable private beta mode
	BootstrapAdminEmail string       // Optional bootstrap admin identity promoted after OTP verification
	Env                 string       // Environment (dev, staging, prod)
	ProxySubnet         netip.Prefix // CIDR of the trusted reverse proxy (zero value = no proxy)
	IPHashSalt          string       // Secret salt for hashing client IPs (required in production)
	VAPIDPublicKey      string       // VAPID public key used for Web Push
	VAPIDPrivateKey     string       // VAPID private key used for Web Push
	CookieDomain        string       // Domain attribute for session cookies (e.g. ".example.com")
	AllowedOrigins      []string     // CORS allowed origins
	Mailer              *Mailer      // Mailer configuration
	VAPIDSubject        string       // VAPID subject (https://{domain}) per RFC 8292
	RequestIDHeader     string       // HTTP header name for request ID propagation (default: X-Request-ID)
	HiveURL             string       // Base URL of the Hive PWA (https://hive.{domain})
	PushURL             string       // Base URL of the push endpoint (https://push.{domain})
	HookURL             string       // Base URL of the webhook endpoint (https://hook.{domain})
	SentryDSN           string       // Sentry/GlitchTip DSN (empty = disabled)
}

// Load reads the .env file (if present) and loads configuration from environment variables.
// Returns a Config struct with sensible defaults if variables are not set.
func Load() (*Config, error) {
	// Load .env file if it exists
	_ = godotenv.Load()

	domain := getEnv(envDomain, defaultDomain)

	var proxySubnet netip.Prefix
	if raw := os.Getenv(envProxySubnet); raw != "" {
		var err error
		proxySubnet, err = netip.ParsePrefix(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid %s: %w", envProxySubnet, err)
		}
	}

	cfg := &Config{
		Domain:              domain,
		Port:                getEnv(envPort, defaultPort),
		DBDir:               getEnv(envDBDir, defaultDBDir),
		AttachmentsDir:      getEnv(envAttachmentsDir, defaultAttachmentsDir),
		URL:                 "https://api." + domain,
		SiteURL:             "https://" + domain,
		HiveURL:             "https://hive." + domain,
		PushURL:             "https://push." + domain,
		HookURL:             "https://hook." + domain,
		CookieDomain:        "." + domain,
		AllowedOrigins:      []string{"https://" + domain, "https://hive." + domain},
		VAPIDSubject:        "https://" + domain,
		ProxySubnet:         proxySubnet,
		IPHashSalt:          getEnv(envIPHashSalt, ""),
		VAPIDPublicKey:      getEnv(envVAPIDPublicKey, ""),
		VAPIDPrivateKey:     getEnv(envVAPIDPrivateKey, ""),
		RequestIDHeader:     getEnv(envRequestIDHeader, ""),
		PrivateBeta:         getEnvBool(envPrivateBeta, true),
		BootstrapAdminEmail: getEnv(envBootstrapAdminEmail, ""),
		Env:                 getEnv(envEnv, defaultEnv),
		SentryDSN:           getEnv(envSentryDSN, ""),
		Mailer: &Mailer{
			SMTPAddress:  getEnv(envMailerSMTPAddress, ""),
			SMTPUser:     getEnv(envMailerSMTPUser, ""),
			SMTPPassword: getEnv(envMailerSMTPPassword, ""),
			ResendAPIKey: getEnv(envMailerResendAPIKey, ""),
			Sender:       getEnv(envMailerSender, defaultMailerSender),
			ReplyTo:      getEnv(envMailerReplyTo, defaultMailerReplyTo),
			SiteURL:      "https://" + domain,
		},
	}

	// Ensure storage directories exist
	if err := os.MkdirAll(cfg.DBDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create DB directory: %w", err)
	}
	if err := os.MkdirAll(cfg.AttachmentsDir, 0o700); err != nil {
		return nil, fmt.Errorf("failed to create attachments directory: %w", err)
	}

	// In production, require critical env vars to be explicitly set
	if cfg.Env == EnvProduction {
		if err := cfg.validateProduction(); err != nil {
			return nil, fmt.Errorf("production config validation failed: %w", err)
		}
	}

	return cfg, nil
}

// validateProduction ensures all critical environment variables are explicitly set in production.
// Returns an error listing all missing variables.
func (c *Config) validateProduction() error {
	var errs []error

	if os.Getenv(envDomain) == "" {
		errs = append(errs, fmt.Errorf("%s is required", envDomain))
	}
	if os.Getenv(envMailerSender) == "" {
		errs = append(errs, fmt.Errorf("%s is required", envMailerSender))
	}
	if os.Getenv(envMailerReplyTo) == "" {
		errs = append(errs, fmt.Errorf("%s is required", envMailerReplyTo))
	}
	if os.Getenv(envMailerSMTPAddress) == "" && os.Getenv(envMailerResendAPIKey) == "" {
		errs = append(errs, fmt.Errorf("either %s or %s is required", envMailerSMTPAddress, envMailerResendAPIKey))
	}
	if os.Getenv(envIPHashSalt) == "" {
		errs = append(errs, fmt.Errorf("%s is required", envIPHashSalt))
	}
	if c.VAPIDPublicKey == "" || c.VAPIDPrivateKey == "" {
		errs = append(errs, fmt.Errorf("%s and %s are required", envVAPIDPublicKey, envVAPIDPrivateKey))
	}

	return errors.Join(errs...)
}

// getEnv retrieves an environment variable with a fallback default value.
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// getEnvBool retrieves a boolean environment variable with a fallback default value.
func getEnvBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value == "true" || value == "1" || value == "yes"
}
