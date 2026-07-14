package config

import (
	"errors"
	"fmt"
	"net/netip"
	"os"
	"strconv"

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
	envDeploymentMode      = "BEEBUZZ_DEPLOYMENT_MODE"
	envBillingProvider     = "BEEBUZZ_BILLING_PROVIDER"
	envBillingSuccessURL   = "BEEBUZZ_BILLING_SUCCESS_URL"
	envBillingGraceDays    = "BEEBUZZ_BILLING_GRACE_PERIOD_DAYS"
	envCreemAPIKey         = "BEEBUZZ_BILLING_CREEM_API_KEY"
	envCreemProductID      = "BEEBUZZ_BILLING_CREEM_PRODUCT_ID"
	envCreemWebhookSecret  = "BEEBUZZ_BILLING_CREEM_WEBHOOK_SECRET"
	envCreemAPIBaseURL     = "BEEBUZZ_BILLING_CREEM_API_BASE_URL"
	envBootstrapAdminEmail = "BEEBUZZ_BOOTSTRAP_ADMIN_EMAIL"
	envEnv                 = "BEEBUZZ_ENV"
	envProxySubnet         = "BEEBUZZ_PROXY_SUBNET"
	envIPHashSalt          = "BEEBUZZ_IP_HASH_SALT"
	envVAPIDPublicKey      = "BEEBUZZ_VAPID_PUBLIC_KEY"
	envVAPIDPrivateKey     = "BEEBUZZ_VAPID_PRIVATE_KEY"
	envRequestIDHeader     = "BEEBUZZ_REQUEST_ID_HEADER"
	envFreeMaxMessagesDay  = "BEEBUZZ_FREE_MAX_MESSAGES_PER_DAY"
	envFreeMaxMessagesMo   = "BEEBUZZ_FREE_MAX_MESSAGES_PER_MONTH"
	envHostedFairUseMsgMo  = "BEEBUZZ_HOSTED_FAIR_USE_MESSAGES_PER_MONTH"
	envMailerSMTPAddress   = "BEEBUZZ_MAILER_SMTP_ADDRESS"
	envMailerSMTPUser      = "BEEBUZZ_MAILER_SMTP_USER"
	envMailerSMTPPassword  = "BEEBUZZ_MAILER_SMTP_PASSWORD"
	envMailerResendAPIKey  = "BEEBUZZ_MAILER_RESEND_API_KEY"
	envMailerSender        = "BEEBUZZ_MAILER_SENDER"
	envMailerReplyTo       = "BEEBUZZ_MAILER_REPLY_TO"
	envSentryDSN           = "BEEBUZZ_SENTRY_DSN"
	envPushStub            = "BEEBUZZ_PUSH_STUB"
)

// defaults
const (
	defaultPort               = "8899"
	defaultDBDir              = "./data/db"
	defaultAttachmentsDir     = "./data/attachments"
	defaultDomain             = "example.com"
	defaultEnv                = EnvDevelopment
	defaultDeploymentMode     = DeploymentModeSelfHosted
	defaultBillingProvider    = ""
	defaultBillingGraceDays   = 7
	defaultBillingSuccessPath = "/account/billing?checkout=success"
	defaultCreemAPIBaseURL    = "https://test-api.creem.io"
	defaultFreeMsgDay         = 50
	defaultFreeMsgMonth       = 500
	defaultHostedMsgMonth     = 100_000
	defaultMailerSender       = "noreply@example.com"
	defaultMailerReplyTo      = "support@example.com"
)

// Deployment mode constants.
const (
	DeploymentModeSelfHosted = "self_hosted"
	DeploymentModeHosted     = "hosted"
)

// Billing provider constants.
const (
	BillingProviderCreem = "creem"
)

// Mailer holds mailer configuration.
type Mailer struct {
	SMTPAddress  string // SMTP server address (host:port)
	SMTPUser     string // SMTP username
	SMTPPassword string // SMTP password
	ResendAPIKey string // Resend API key
	Sender       string // Email sender address
	ReplyTo      string // Reply-To address
	SiteURL      string // Dashboard base URL for email links
}

// Config holds the application configuration loaded from environment variables.
type Config struct {
	Domain                     string       // Base domain (e.g. "example.com")
	Port                       string       // HTTP server port
	DBDir                      string       // Directory for the SQLite database file
	AttachmentsDir             string       // Directory for attachment file storage
	URL                        string       // API base URL (https://api.{domain})
	SiteURL                    string       // Dashboard base URL (https://dashboard.{domain})
	DeploymentMode             string       // Deployment mode: self_hosted or hosted
	BootstrapAdminEmail        string       // Optional bootstrap admin identity promoted after OTP verification
	Env                        string       // Environment (dev, staging, prod)
	ProxySubnet                netip.Prefix // CIDR of the trusted reverse proxy (zero value = no proxy)
	IPHashSalt                 string       // Secret salt for hashing client IPs (required in production)
	VAPIDPublicKey             string       // VAPID public key used for Web Push
	VAPIDPrivateKey            string       // VAPID private key used for Web Push
	CookieDomain               string       // Domain attribute for session cookies (e.g. ".example.com")
	AllowedOrigins             []string     // CORS allowed origins
	Mailer                     *Mailer      // Mailer configuration
	VAPIDSubject               string       // VAPID subject (https://{domain}) per RFC 8292
	RequestIDHeader            string       // HTTP header name for request ID propagation (default: X-Request-ID)
	BillingProvider            string       // Billing provider adapter (empty = disabled)
	BillingSuccessURL          string       // Return URL after provider checkout
	BillingGracePeriodDays     int          // Hosted grace period after payment issues
	CreemAPIKey                string       // Creem API key
	CreemProductID             string       // Creem Hosted plan product ID
	CreemWebhookSecret         string       // Creem webhook signing secret
	CreemAPIBaseURL            string       // Creem API base URL
	FreeMaxMessagesDay         int          // Hosted Free hard message cap per UTC day
	FreeMaxMessagesMonth       int          // Hosted Free hard message cap per UTC calendar month
	HostedFairUseMessagesMonth int          // Hosted paid fair-use message threshold per UTC calendar month
	HiveURL                    string       // Base URL of the Hive PWA (https://hive.{domain})
	PushURL                    string       // Base URL of the push endpoint (https://push.{domain})
	HookURL                    string       // Base URL of the webhook endpoint (https://hook.{domain})
	SentryDSN                  string       // Sentry/GlitchTip DSN (empty = disabled)
	PushStub                   bool         // Enable push-stub mode for local/dev testing (NEVER in production)
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

	freeMaxMessagesDay, err := getEnvInt(envFreeMaxMessagesDay, defaultFreeMsgDay)
	if err != nil {
		return nil, err
	}
	freeMaxMessagesMo, err := getEnvInt(envFreeMaxMessagesMo, defaultFreeMsgMonth)
	if err != nil {
		return nil, err
	}
	hostedFairUseMsgMo, err := getEnvInt(envHostedFairUseMsgMo, defaultHostedMsgMonth)
	if err != nil {
		return nil, err
	}
	billingGracePeriodDays, err := getEnvInt(envBillingGraceDays, defaultBillingGraceDays)
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		Domain:                     domain,
		Port:                       getEnv(envPort, defaultPort),
		DBDir:                      getEnv(envDBDir, defaultDBDir),
		AttachmentsDir:             getEnv(envAttachmentsDir, defaultAttachmentsDir),
		URL:                        "https://api." + domain,
		SiteURL:                    "https://dashboard." + domain,
		HiveURL:                    "https://hive." + domain,
		PushURL:                    "https://push." + domain,
		HookURL:                    "https://hook." + domain,
		CookieDomain:               "." + domain,
		AllowedOrigins:             []string{"https://dashboard." + domain, "https://hive." + domain},
		VAPIDSubject:               "https://" + domain,
		ProxySubnet:                proxySubnet,
		IPHashSalt:                 getEnv(envIPHashSalt, ""),
		VAPIDPublicKey:             getEnv(envVAPIDPublicKey, ""),
		VAPIDPrivateKey:            getEnv(envVAPIDPrivateKey, ""),
		RequestIDHeader:            getEnv(envRequestIDHeader, ""),
		BillingProvider:            getEnv(envBillingProvider, defaultBillingProvider),
		BillingSuccessURL:          getEnv(envBillingSuccessURL, "https://dashboard."+domain+defaultBillingSuccessPath),
		BillingGracePeriodDays:     billingGracePeriodDays,
		CreemAPIKey:                getEnv(envCreemAPIKey, ""),
		CreemProductID:             getEnv(envCreemProductID, ""),
		CreemWebhookSecret:         getEnv(envCreemWebhookSecret, ""),
		CreemAPIBaseURL:            getEnv(envCreemAPIBaseURL, defaultCreemAPIBaseURL),
		FreeMaxMessagesDay:         freeMaxMessagesDay,
		FreeMaxMessagesMonth:       freeMaxMessagesMo,
		HostedFairUseMessagesMonth: hostedFairUseMsgMo,
		DeploymentMode:             getEnv(envDeploymentMode, defaultDeploymentMode),
		BootstrapAdminEmail:        getEnv(envBootstrapAdminEmail, ""),
		Env:                        getEnv(envEnv, defaultEnv),
		SentryDSN:                  getEnv(envSentryDSN, ""),
		PushStub:                   getEnvBool(envPushStub, false),
		Mailer: &Mailer{
			SMTPAddress:  getEnv(envMailerSMTPAddress, ""),
			SMTPUser:     getEnv(envMailerSMTPUser, ""),
			SMTPPassword: getEnv(envMailerSMTPPassword, ""),
			ResendAPIKey: getEnv(envMailerResendAPIKey, ""),
			Sender:       getEnv(envMailerSender, defaultMailerSender),
			ReplyTo:      getEnv(envMailerReplyTo, defaultMailerReplyTo),
			SiteURL:      "https://dashboard." + domain,
		},
	}

	if !isValidDeploymentMode(cfg.DeploymentMode) {
		return nil, fmt.Errorf("invalid %s: must be one of %s, %s", envDeploymentMode, DeploymentModeSelfHosted, DeploymentModeHosted)
	}
	if !isValidBillingProvider(cfg.BillingProvider) {
		return nil, fmt.Errorf("invalid %s: must be empty or creem", envBillingProvider)
	}
	if cfg.BillingProvider != "" && !cfg.IsHosted() {
		return nil, fmt.Errorf("invalid %s: requires %s=%s", envBillingProvider, envDeploymentMode, DeploymentModeHosted)
	}
	if cfg.FreeMaxMessagesDay < 0 {
		return nil, fmt.Errorf("invalid %s: must be >= 0", envFreeMaxMessagesDay)
	}
	if cfg.FreeMaxMessagesMonth < 0 {
		return nil, fmt.Errorf("invalid %s: must be >= 0", envFreeMaxMessagesMo)
	}
	if cfg.HostedFairUseMessagesMonth < 0 {
		return nil, fmt.Errorf("invalid %s: must be >= 0", envHostedFairUseMsgMo)
	}
	if cfg.BillingGracePeriodDays < 0 {
		return nil, fmt.Errorf("invalid %s: must be >= 0", envBillingGraceDays)
	}

	// Ensure storage directories exist. The DB directory holds session,
	// API-key, and webhook secret hashes plus user identifying data, so
	// it must not be readable by other local users. database.New
	// re-applies these permissions at startup to tighten directories
	// that were created with the previous looser mode.
	if err := os.MkdirAll(cfg.DBDir, 0o700); err != nil {
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

// IsHosted reports whether this instance is BeeBuzz-operated hosted service.
func (c *Config) IsHosted() bool {
	return c.DeploymentMode == DeploymentModeHosted
}

func isValidDeploymentMode(mode string) bool {
	switch mode {
	case DeploymentModeSelfHosted, DeploymentModeHosted:
		return true
	}
	return false
}

func isValidBillingProvider(provider string) bool {
	switch provider {
	case "", BillingProviderCreem:
		return true
	}
	return false
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

// getEnvInt retrieves an integer environment variable with a fallback default value.
func getEnvInt(key string, defaultValue int) (int, error) {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: must be an integer", key)
	}
	return parsed, nil
}
