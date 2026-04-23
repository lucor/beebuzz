package main

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

// ProfileService resolves effective profile config from flags, env, and local storage.
type ProfileService struct {
	store  *ProfileStore
	getenv func(string) string
}

func newProfileService(store *ProfileStore, getenv func(string) string) *ProfileService {
	return &ProfileService{
		store:  store,
		getenv: getenv,
	}
}

// resolveProfile returns the given profile name if non-empty, or the default profile otherwise.
func (s *ProfileService) resolveProfile(profile string) (string, error) {
	if strings.TrimSpace(profile) != "" {
		return strings.TrimSpace(profile), nil
	}

	return s.resolveDefaultProfileName()
}

// resolveConfig returns the effective config for a profile name plus optional API URL override.
func (s *ProfileService) resolveConfig(profileName, apiURLOverride string) (*Config, error) {
	resolvedProfile, err := s.resolveProfile(profileName)
	if err != nil {
		return nil, err
	}

	config, err := s.loadConfig(resolvedProfile)
	if err != nil {
		return nil, err
	}

	if !s.hasRuntimeAPIConfigOverride() {
		if err := validateAPIURLOverrideHost(config.APIURL, apiURLOverride); err != nil {
			return nil, err
		}
	}

	s.applyAPIURLOverride(config, apiURLOverride)
	return config, nil
}

func (s *ProfileService) hasRuntimeAPIConfigOverride() bool {
	return strings.TrimSpace(s.getenv(envBeeBuzzAPIURL)) != "" || strings.TrimSpace(s.getenv(envBeeBuzzAPIToken)) != ""
}

// resolveDefaultProfileName returns the default profile from file, or the fallback.
func (s *ProfileService) resolveDefaultProfileName() (string, error) {
	name, err := s.store.loadDefaultProfileName()
	if err != nil {
		return "", err
	}

	if name != "" {
		return name, nil
	}

	return fallbackProfileName, nil
}

// loadConfig reads config for a specific profile.
func (s *ProfileService) loadConfig(profile string) (*Config, error) {
	if err := validateProfileName(profile); err != nil {
		return nil, err
	}

	configPath, err := s.store.profileConfigPath(profile)
	if err != nil {
		return nil, err
	}

	config, err := s.store.loadConfigFromPath(configPath)
	if err == nil {
		s.applyEnvOverrides(config)
		config.Profile = profile
		return config, nil
	}

	if !isConfigNotFoundError(err) {
		return nil, err
	}

	config = s.loadConfigFromEnv()
	if config.APIURL == "" && config.APIToken == "" {
		return nil, fmt.Errorf("profile %q not configured: run beebuzz connect --profile %s", profile, profile)
	}

	config.Profile = profile
	return config, nil
}

// loadConfigFromEnv reads runtime config values from supported environment variables.
func (s *ProfileService) loadConfigFromEnv() *Config {
	config := &Config{
		APIURL:     strings.TrimSpace(s.getenv(envBeeBuzzAPIURL)),
		APIToken:   s.getenv(envBeeBuzzAPIToken),
		DeviceKeys: []DeviceKey{},
	}
	config.Normalize()

	return config
}

// applyEnvOverrides overlays environment variables on top of a loaded config.
func (s *ProfileService) applyEnvOverrides(config *Config) {
	if config == nil {
		return
	}

	envConfig := s.loadConfigFromEnv()
	if envConfig.APIURL != "" {
		config.APIURL = envConfig.APIURL
	}
	if envConfig.APIToken != "" {
		config.APIToken = envConfig.APIToken
	}

	config.Normalize()
}

// applyAPIURLOverride overlays a non-empty api_url flag value on top of a loaded config.
func (s *ProfileService) applyAPIURLOverride(config *Config, apiURL string) {
	if config == nil {
		return
	}

	trimmedAPIURL := strings.TrimSpace(apiURL)
	if trimmedAPIURL == "" {
		return
	}

	config.APIURL = trimmedAPIURL
	config.Normalize()
}

func validateAPIURLOverrideHost(currentAPIURL, override string) error {
	trimmedOverride := strings.TrimSpace(override)
	if trimmedOverride == "" {
		return nil
	}

	currentHost, err := apiURLHost(currentAPIURL)
	if err != nil || currentHost == "" {
		return nil
	}

	overrideHost, err := apiURLHost(trimmedOverride)
	if err != nil || overrideHost == "" {
		return nil
	}

	if strings.EqualFold(currentHost, overrideHost) {
		return nil
	}

	return fmt.Errorf("api-url override host %q does not match configured host %q; run beebuzz connect to switch servers", overrideHost, currentHost)
}

func apiURLHost(rawURL string) (string, error) {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return "", nil
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", err
	}

	return parsed.Host, nil
}

func (s *ProfileService) saveResolvedConfig(config *Config) error {
	if config == nil {
		return fmt.Errorf("config is required")
	}
	if config.Profile == "" {
		return fmt.Errorf("config profile is required")
	}

	return s.store.saveConfig(config.Profile, config)
}

func (s *ProfileService) setupDefaults() setupDefaults {
	config := s.loadConfigFromEnv()
	if config.APIURL == "" {
		config.APIURL = defaultAPIURL
	}

	return setupDefaults{
		APIURL:   config.APIURL,
		APIToken: config.APIToken,
	}
}

// list returns all available profile names.
func (s *ProfileService) list() ([]string, error) {
	return s.store.listProfileNames()
}

// show returns config for a profile (token masked).
func (s *ProfileService) show(name string) (*Config, error) {
	if err := validateProfileName(name); err != nil {
		return nil, err
	}

	config, err := s.loadConfig(name)
	if err != nil {
		return nil, err
	}

	config.APIToken = maskToken(config.APIToken)
	return config, nil
}

// setDefault sets the default profile name.
func (s *ProfileService) setDefault(name string) error {
	if err := validateProfileName(name); err != nil {
		return err
	}

	exists, err := s.store.profileExists(name)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("profile %q does not exist", name)
	}

	return s.store.saveDefaultProfileName(name)
}

// delete removes a profile.
func (s *ProfileService) delete(name string) error {
	if err := validateProfileName(name); err != nil {
		return err
	}

	currentDefault, err := s.store.loadDefaultProfileName()
	if err != nil {
		return err
	}

	if currentDefault == name {
		return fmt.Errorf("cannot delete active default profile %q", name)
	}

	exists, err := s.store.profileExists(name)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("profile %q does not exist", name)
	}

	return s.store.deleteProfileDir(name)
}

var defaultProfileService = newProfileService(defaultProfileStore, os.Getenv)
