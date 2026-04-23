package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"lucor.dev/beebuzz/internal/secure"
)

// ProfileStore manages profile config persistence on local disk.
type ProfileStore struct {
	basePath func() (string, error)
}

func newProfileStore(basePath func() (string, error)) *ProfileStore {
	return &ProfileStore{basePath: basePath}
}

var configBasePath = configBaseDirPath
var defaultProfileStore = newProfileStore(func() (string, error) {
	return configBasePath()
})

// configBaseDirPath returns the base config directory path.
func configBaseDirPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}

	return filepath.Join(homeDir, configDirName), nil
}

// defaultProfilePath returns the path to the default profile file.
func (s *ProfileStore) defaultProfilePath() (string, error) {
	base, err := s.basePath()
	if err != nil {
		return "", err
	}

	return filepath.Join(base, defaultProfileFileName), nil
}

// profilesDirPath returns the path to the profiles directory.
func (s *ProfileStore) profilesDirPath() (string, error) {
	base, err := s.basePath()
	if err != nil {
		return "", err
	}

	return filepath.Join(base, profilesDirName), nil
}

// profileDirPath returns the path to a profile's directory.
func (s *ProfileStore) profileDirPath(profile string) (string, error) {
	profilesDir, err := s.profilesDirPath()
	if err != nil {
		return "", err
	}

	return filepath.Join(profilesDir, profile), nil
}

// profileConfigPath returns the path to a profile's config file.
func (s *ProfileStore) profileConfigPath(profile string) (string, error) {
	profilesDir, err := s.profilesDirPath()
	if err != nil {
		return "", err
	}

	return filepath.Join(profilesDir, profile, configFileName), nil
}

// loadConfigFromPath reads a config file from a specific path.
func (s *ProfileStore) loadConfigFromPath(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%s%s", errPrefixConfigNotFound, configPath)
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	var raw struct {
		APIURL     string      `json:"api_url"`
		APIToken   string      `json:"api_token"`
		DeviceKeys []DeviceKey `json:"device_keys"`
		LegacyKeys []string    `json:"keys"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}

	config := Config{
		APIURL:     raw.APIURL,
		APIToken:   raw.APIToken,
		DeviceKeys: raw.DeviceKeys,
	}
	if len(config.DeviceKeys) == 0 && len(raw.LegacyKeys) > 0 {
		config.DeviceKeys = legacyDeviceKeys(raw.LegacyKeys)
	}

	config.Normalize()
	return &config, nil
}

// saveConfigToPath writes a config file to a specific path.
func (s *ProfileStore) saveConfigToPath(configPath string, config *Config) error {
	if config == nil {
		return fmt.Errorf("config is required")
	}

	config.Normalize()

	if err := os.MkdirAll(filepath.Dir(configPath), configDirPerm); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}

	if err := os.WriteFile(configPath, append(data, '\n'), configFilePerm); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}

func legacyDeviceKeys(keys []string) []DeviceKey {
	deviceKeys := make([]DeviceKey, 0, len(keys))
	for _, key := range keys {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		deviceKeys = append(deviceKeys, DeviceKey{
			AgeRecipient:            trimmedKey,
			AgeRecipientFingerprint: secure.Fingerprint(trimmedKey),
		})
	}

	return deviceKeys
}

func (s *ProfileStore) saveConfig(profile string, config *Config) error {
	if err := validateProfileName(profile); err != nil {
		return err
	}

	configPath, err := s.profileConfigPath(profile)
	if err != nil {
		return err
	}

	return s.saveConfigToPath(configPath, config)
}

// loadDefaultProfileName reads the default profile name from file.
func (s *ProfileStore) loadDefaultProfileName() (string, error) {
	path, err := s.defaultProfilePath()
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read default profile: %w", err)
	}

	name := strings.TrimSpace(string(data))
	if name == "" {
		return "", nil
	}

	return name, nil
}

func (s *ProfileStore) saveDefaultProfileName(name string) error {
	path, err := s.defaultProfilePath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), configDirPerm); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	return os.WriteFile(path, []byte(name+"\n"), configFilePerm)
}

func (s *ProfileStore) listProfileNames() ([]string, error) {
	path, err := s.profilesDirPath()
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(path)
	if os.IsNotExist(err) {
		return []string{fallbackProfileName}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read profiles dir: %w", err)
	}

	var profiles []string
	for _, entry := range entries {
		if entry.IsDir() {
			profiles = append(profiles, entry.Name())
		}
	}

	if len(profiles) == 0 {
		profiles = []string{fallbackProfileName}
	}

	return profiles, nil
}

func (s *ProfileStore) profileExists(name string) (bool, error) {
	profilePath, err := s.profileConfigPath(name)
	if err != nil {
		return false, err
	}

	if _, err := os.Stat(profilePath); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func (s *ProfileStore) deleteProfileDir(name string) error {
	profileDir, err := s.profileDirPath(name)
	if err != nil {
		return err
	}

	if err := os.RemoveAll(profileDir); err != nil {
		return fmt.Errorf("delete profile: %w", err)
	}

	return nil
}
