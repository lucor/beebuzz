package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveConfigToPathAndLoadConfigFromPath(t *testing.T) {
	store := newProfileStore(func() (string, error) { return t.TempDir(), nil })
	configPath := filepath.Join(t.TempDir(), ".beebuzz", "config.json")
	config := &Config{
		APIURL:   " https://api.example.com/ ",
		APIToken: " beebuzz_token ",
		DeviceKeys: []DeviceKey{
			{AgeRecipient: "age1abc"},
		},
	}

	if err := store.saveConfigToPath(configPath, config); err != nil {
		t.Fatalf("saveConfigToPath: %v", err)
	}

	loadedConfig, err := store.loadConfigFromPath(configPath)
	if err != nil {
		t.Fatalf("loadConfigFromPath: %v", err)
	}

	if loadedConfig.APIURL != "https://api.example.com" {
		t.Fatalf("APIURL: got %q, want %q", loadedConfig.APIURL, "https://api.example.com")
	}
	if loadedConfig.APIToken != "beebuzz_token" {
		t.Fatalf("APIToken: got %q, want %q", loadedConfig.APIToken, "beebuzz_token")
	}
	if len(loadedConfig.DeviceKeys) != 1 || loadedConfig.DeviceKeys[0].AgeRecipient != "age1abc" {
		t.Fatalf("DeviceKeys: got %#v", loadedConfig.DeviceKeys)
	}
}

func TestLoadConfigFromPathMissingFile(t *testing.T) {
	store := newProfileStore(func() (string, error) { return t.TempDir(), nil })
	configPath := filepath.Join(t.TempDir(), ".beebuzz", "config.json")

	_, err := store.loadConfigFromPath(configPath)
	if err == nil {
		t.Fatal("expected error for missing config")
	}
}

func TestLoadProfileConfigAppliesEnvOverrides(t *testing.T) {
	t.Setenv(envBeeBuzzAPIURL, " https://env.example.com/ ")
	t.Setenv(envBeeBuzzAPIToken, " env_token ")

	baseDir := t.TempDir()
	setIsolatedConfigBase(t, baseDir)
	store := newProfileStore(func() (string, error) { return configBasePath() })
	service := newProfileService(store, getenvFromT(t))

	configPath := filepath.Join(baseDir, profilesDirName, fallbackProfileName, configFileName)
	config := &Config{
		APIURL:   "https://file.example.com",
		APIToken: "file_token",
		DeviceKeys: []DeviceKey{
			{AgeRecipient: "age1abc"},
		},
	}

	if err := store.saveConfigToPath(configPath, config); err != nil {
		t.Fatalf("saveConfigToPath: %v", err)
	}

	loadedConfig, err := service.loadConfig(fallbackProfileName)
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}

	if loadedConfig.APIURL != "https://env.example.com" {
		t.Fatalf("APIURL: got %q, want %q", loadedConfig.APIURL, "https://env.example.com")
	}
	if loadedConfig.APIToken != "env_token" {
		t.Fatalf("APIToken: got %q, want %q", loadedConfig.APIToken, "env_token")
	}
	if len(loadedConfig.DeviceKeys) != 1 || loadedConfig.DeviceKeys[0].AgeRecipient != "age1abc" {
		t.Fatalf("DeviceKeys: got %#v", loadedConfig.DeviceKeys)
	}
}

func TestLoadProfileConfigUsesEnvWithoutConfigFile(t *testing.T) {
	t.Setenv(envBeeBuzzAPIURL, "https://env.example.com/")
	t.Setenv(envBeeBuzzAPIToken, "env_token")

	setIsolatedConfigBase(t, t.TempDir())
	store := newProfileStore(func() (string, error) { return configBasePath() })
	service := newProfileService(store, getenvFromT(t))

	loadedConfig, err := service.loadConfig(fallbackProfileName)
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}

	if loadedConfig.APIURL != "https://env.example.com" {
		t.Fatalf("APIURL: got %q, want %q", loadedConfig.APIURL, "https://env.example.com")
	}
	if loadedConfig.APIToken != "env_token" {
		t.Fatalf("APIToken: got %q, want %q", loadedConfig.APIToken, "env_token")
	}
	if len(loadedConfig.DeviceKeys) != 0 {
		t.Fatalf("DeviceKeys: got %#v, want empty", loadedConfig.DeviceKeys)
	}
}

func TestResolveConfigRejectsCrossHostAPIURLOverride(t *testing.T) {
	baseDir := t.TempDir()
	setIsolatedConfigBase(t, baseDir)
	store := newProfileStore(func() (string, error) { return configBasePath() })
	service := newProfileService(store, getenvFromT(t))

	configPath := filepath.Join(baseDir, profilesDirName, fallbackProfileName, configFileName)
	config := &Config{
		APIURL:   "https://api.example.com",
		APIToken: "token",
	}

	if err := store.saveConfigToPath(configPath, config); err != nil {
		t.Fatalf("saveConfigToPath: %v", err)
	}

	_, err := service.resolveConfig(fallbackProfileName, "https://evil.example.com")
	if err == nil {
		t.Fatal("expected cross-host override error")
	}
}

func TestResolveConfigAllowsSameHostAPIURLOverride(t *testing.T) {
	baseDir := t.TempDir()
	setIsolatedConfigBase(t, baseDir)
	store := newProfileStore(func() (string, error) { return configBasePath() })
	service := newProfileService(store, getenvFromT(t))

	configPath := filepath.Join(baseDir, profilesDirName, fallbackProfileName, configFileName)
	config := &Config{
		APIURL:   "https://api.example.com",
		APIToken: "token",
	}

	if err := store.saveConfigToPath(configPath, config); err != nil {
		t.Fatalf("saveConfigToPath: %v", err)
	}

	resolvedConfig, err := service.resolveConfig(fallbackProfileName, "https://api.example.com/v1")
	if err != nil {
		t.Fatalf("resolveConfig: %v", err)
	}

	if resolvedConfig.APIURL != "https://api.example.com/v1" {
		t.Fatalf("APIURL: got %q, want %q", resolvedConfig.APIURL, "https://api.example.com/v1")
	}
}

// setIsolatedConfigBase overrides configBasePath to return the given directory,
// and restores it when the test finishes.
func setIsolatedConfigBase(t *testing.T, baseDir string) {
	t.Helper()

	original := configBasePath
	configBasePath = func() (string, error) { return baseDir, nil }
	t.Cleanup(func() {
		configBasePath = original
	})
}

func getenvFromT(t *testing.T) func(string) string {
	t.Helper()

	return func(key string) string {
		return os.Getenv(key)
	}
}
