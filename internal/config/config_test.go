package config

import (
	"slices"
	"testing"
)

func TestLoadDerivesDashboardAndHiveOrigins(t *testing.T) {
	t.Setenv(envDomain, "example.com")
	t.Setenv(envEnv, EnvDevelopment)

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
