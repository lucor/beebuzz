package config

import "testing"

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
