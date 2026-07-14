package config

import (
	"slices"
	"testing"
)

func TestLoadDerivesDashboardAndHiveOrigins(t *testing.T) {
	t.Setenv(envDomain, "example.com")
	t.Setenv(envEnv, EnvDevelopment)

	// Isolate billing/deployment settings so this test verifies defaults
	// regardless of any values leaked from the developer's environment.
	for _, key := range []string{
		envDeploymentMode,
		envBillingProvider,
		envBillingSuccessURL,
		envBillingGraceDays,
		envCreemAPIKey,
		envCreemProductID,
		envCreemWebhookSecret,
		envCreemAPIBaseURL,
		envFreeMaxMessagesDay,
		envFreeMaxMessagesMo,
		envHostedFairUseMsgMo,
	} {
		t.Setenv(key, "")
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.SiteURL != "https://dashboard.example.com" {
		t.Fatalf("SiteURL = %q, want dashboard URL", cfg.SiteURL)
	}
	if cfg.Mailer.SiteURL != cfg.SiteURL {
		t.Fatalf("Mailer.SiteURL = %q, want %q", cfg.Mailer.SiteURL, cfg.SiteURL)
	}
	if !slices.Contains(cfg.AllowedOrigins, "https://dashboard.example.com") {
		t.Fatalf("AllowedOrigins missing dashboard origin: %#v", cfg.AllowedOrigins)
	}
	if !slices.Contains(cfg.AllowedOrigins, "https://hive.example.com") {
		t.Fatalf("AllowedOrigins missing hive origin: %#v", cfg.AllowedOrigins)
	}
	if slices.Contains(cfg.AllowedOrigins, "https://example.com") {
		t.Fatalf("AllowedOrigins contains root site origin: %#v", cfg.AllowedOrigins)
	}
	if cfg.DeploymentMode != DeploymentModeSelfHosted {
		t.Fatalf("DeploymentMode = %q, want %q", cfg.DeploymentMode, DeploymentModeSelfHosted)
	}
	if cfg.FreeMaxMessagesDay != defaultFreeMsgDay {
		t.Fatalf("FreeMaxMessagesDay = %d, want %d", cfg.FreeMaxMessagesDay, defaultFreeMsgDay)
	}
	if cfg.FreeMaxMessagesMonth != defaultFreeMsgMonth {
		t.Fatalf("FreeMaxMessagesMonth = %d, want %d", cfg.FreeMaxMessagesMonth, defaultFreeMsgMonth)
	}
	if cfg.HostedFairUseMessagesMonth != defaultHostedMsgMonth {
		t.Fatalf("HostedFairUseMessagesMonth = %d, want %d", cfg.HostedFairUseMessagesMonth, defaultHostedMsgMonth)
	}
	if cfg.BillingProvider != "" {
		t.Fatalf("BillingProvider = %q, want empty", cfg.BillingProvider)
	}
	if cfg.BillingSuccessURL != "https://dashboard.example.com/account/billing?checkout=success" {
		t.Fatalf("BillingSuccessURL = %q, want dashboard billing confirmation URL", cfg.BillingSuccessURL)
	}
	if cfg.BillingGracePeriodDays != defaultBillingGraceDays {
		t.Fatalf("BillingGracePeriodDays = %d, want %d", cfg.BillingGracePeriodDays, defaultBillingGraceDays)
	}
	if cfg.CreemAPIBaseURL != defaultCreemAPIBaseURL {
		t.Fatalf("CreemAPIBaseURL = %q, want %q", cfg.CreemAPIBaseURL, defaultCreemAPIBaseURL)
	}
}

func TestLoadDeploymentMode(t *testing.T) {
	t.Run("accepts hosted", func(t *testing.T) {
		t.Setenv(envDeploymentMode, DeploymentModeHosted)

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if cfg.DeploymentMode != DeploymentModeHosted {
			t.Fatalf("DeploymentMode = %q, want %q", cfg.DeploymentMode, DeploymentModeHosted)
		}
		if !cfg.IsHosted() {
			t.Fatal("IsHosted() = false, want true")
		}
	})

	t.Run("rejects invalid", func(t *testing.T) {
		t.Setenv(envDeploymentMode, "cloud")

		if _, err := Load(); err == nil {
			t.Fatal("Load() error = nil, want invalid deployment mode")
		}
	})
}

func TestLoadMessageLimits(t *testing.T) {
	t.Run("accepts overrides", func(t *testing.T) {
		t.Setenv(envFreeMaxMessagesDay, "25")
		t.Setenv(envFreeMaxMessagesMo, "250")
		t.Setenv(envHostedFairUseMsgMo, "50000")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if cfg.FreeMaxMessagesDay != 25 {
			t.Fatalf("FreeMaxMessagesDay = %d, want 25", cfg.FreeMaxMessagesDay)
		}
		if cfg.FreeMaxMessagesMonth != 250 {
			t.Fatalf("FreeMaxMessagesMonth = %d, want 250", cfg.FreeMaxMessagesMonth)
		}
		if cfg.HostedFairUseMessagesMonth != 50000 {
			t.Fatalf("HostedFairUseMessagesMonth = %d, want 50000", cfg.HostedFairUseMessagesMonth)
		}
	})

	t.Run("rejects negative", func(t *testing.T) {
		t.Setenv(envFreeMaxMessagesDay, "-1")

		if _, err := Load(); err == nil {
			t.Fatal("Load() error = nil, want invalid limit")
		}
	})

	t.Run("rejects non-integer", func(t *testing.T) {
		t.Setenv(envFreeMaxMessagesMo, "many")

		if _, err := Load(); err == nil {
			t.Fatal("Load() error = nil, want invalid limit")
		}
	})
}

func TestLoadBillingProvider(t *testing.T) {
	t.Run("accepts creem", func(t *testing.T) {
		t.Setenv(envDeploymentMode, DeploymentModeHosted)
		t.Setenv(envBillingProvider, BillingProviderCreem)
		t.Setenv(envBillingSuccessURL, "https://dashboard.example.com/thanks")
		t.Setenv(envBillingGraceDays, "3")
		t.Setenv(envCreemAPIKey, "creem_test_key")
		t.Setenv(envCreemProductID, "prod_123")
		t.Setenv(envCreemWebhookSecret, "secret")
		t.Setenv(envCreemAPIBaseURL, "https://test-api.example.com")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if cfg.BillingProvider != BillingProviderCreem {
			t.Fatalf("BillingProvider = %q, want %q", cfg.BillingProvider, BillingProviderCreem)
		}
		if cfg.BillingSuccessURL != "https://dashboard.example.com/thanks" {
			t.Fatalf("BillingSuccessURL = %q, want override", cfg.BillingSuccessURL)
		}
		if cfg.BillingGracePeriodDays != 3 {
			t.Fatalf("BillingGracePeriodDays = %d, want 3", cfg.BillingGracePeriodDays)
		}
		if cfg.CreemAPIKey != "creem_test_key" {
			t.Fatal("CreemAPIKey override not loaded")
		}
		if cfg.CreemProductID != "prod_123" {
			t.Fatal("CreemProductID override not loaded")
		}
		if cfg.CreemWebhookSecret != "secret" {
			t.Fatal("CreemWebhookSecret override not loaded")
		}
		if cfg.CreemAPIBaseURL != "https://test-api.example.com" {
			t.Fatalf("CreemAPIBaseURL = %q, want override", cfg.CreemAPIBaseURL)
		}
	})

	t.Run("rejects invalid", func(t *testing.T) {
		t.Setenv(envDeploymentMode, DeploymentModeHosted)
		t.Setenv(envBillingProvider, "stripe")

		if _, err := Load(); err == nil {
			t.Fatal("Load() error = nil, want invalid billing provider")
		}
	})

	t.Run("rejects billing provider in self-hosted mode", func(t *testing.T) {
		t.Setenv(envDeploymentMode, DeploymentModeSelfHosted)
		t.Setenv(envBillingProvider, BillingProviderCreem)

		if _, err := Load(); err == nil {
			t.Fatal("Load() error = nil, want billing provider to require hosted mode")
		}
	})

	t.Run("rejects negative grace period", func(t *testing.T) {
		t.Setenv(envBillingGraceDays, "-1")

		if _, err := Load(); err == nil {
			t.Fatal("Load() error = nil, want invalid grace period")
		}
	})
}

func TestValidateProductionRequiresVAPIDKeys(t *testing.T) {
	t.Setenv(envDomain, "example.com")
	t.Setenv(envMailerSender, defaultMailerSender)
	t.Setenv(envMailerReplyTo, defaultMailerReplyTo)
	t.Setenv(envMailerSMTPAddress, "smtp.example.com:25")
	t.Setenv(envMailerResendAPIKey, "")
	t.Setenv(envIPHashSalt, "secret-salt")

	t.Run("rejects missing VAPID keys", func(t *testing.T) {
		t.Setenv(envVAPIDPublicKey, "")
		t.Setenv(envVAPIDPrivateKey, "")

		cfg := &Config{Env: EnvProduction}
		if err := cfg.validateProduction(); err == nil {
			t.Fatal("validateProduction() error = nil, want missing VAPID keys")
		}
	})

	t.Run("accepts present VAPID keys", func(t *testing.T) {
		t.Setenv(envVAPIDPublicKey, "public-key")
		t.Setenv(envVAPIDPrivateKey, "private-key")

		cfg := &Config{
			Env:             EnvProduction,
			VAPIDPublicKey:  "public-key",
			VAPIDPrivateKey: "private-key",
		}
		if err := cfg.validateProduction(); err != nil {
			t.Fatalf("validateProduction() error = %v", err)
		}
	})
}
