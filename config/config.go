// Package config provides configuration file support for Aha Studio.
package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	configFileName = "config.yaml"
	configDirName  = ".ahastudio"
)

// Config represents the Aha Studio configuration.
type Config struct {
	// Defaults contains default settings.
	Defaults Defaults `yaml:"defaults,omitempty"`

	// Profiles contains named connection profiles.
	Profiles map[string]Profile `yaml:"profiles,omitempty"`

	// Queries contains saved named queries.
	Queries map[string]string `yaml:"queries,omitempty"`

	// path is the config file path (not persisted).
	path string `yaml:"-"`
}

// Defaults contains default settings.
type Defaults struct {
	// Output is the default output format (table, json, csv).
	Output string `yaml:"output,omitempty"`

	// Product is the default product ID or reference.
	Product string `yaml:"product,omitempty"`

	// PerPage is the default results per page.
	PerPage int `yaml:"per_page,omitempty"`

	// Profile is the default connection profile.
	Profile string `yaml:"profile,omitempty"`
}

// Profile contains connection profile settings.
type Profile struct {
	// Subdomain is the Aha account subdomain.
	Subdomain string `yaml:"subdomain,omitempty"`

	// APIKeyEnv is the environment variable containing the API key.
	APIKeyEnv string `yaml:"api_key_env,omitempty"`
}

// Load loads the configuration from the default location.
func Load() (*Config, error) {
	path := configPath()
	return LoadFrom(path)
}

// LoadFrom loads the configuration from the specified path.
func LoadFrom(path string) (*Config, error) {
	cfg := &Config{
		Profiles: make(map[string]Profile),
		Queries:  make(map[string]string),
		path:     path,
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	// Ensure maps are initialized
	if cfg.Profiles == nil {
		cfg.Profiles = make(map[string]Profile)
	}
	if cfg.Queries == nil {
		cfg.Queries = make(map[string]string)
	}

	cfg.path = path
	return cfg, nil
}

// Save saves the configuration to disk.
func (c *Config) Save() error {
	if c.path == "" {
		c.path = configPath()
	}

	// Ensure directory exists
	dir := filepath.Dir(c.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(c.path, data, 0600)
}

// SaveQuery saves a named query.
func (c *Config) SaveQuery(name, query string) {
	if c.Queries == nil {
		c.Queries = make(map[string]string)
	}
	c.Queries[name] = query
}

// GetQuery retrieves a named query.
func (c *Config) GetQuery(name string) (string, bool) {
	if c.Queries == nil {
		return "", false
	}
	q, ok := c.Queries[name]
	return q, ok
}

// DeleteQuery deletes a named query.
func (c *Config) DeleteQuery(name string) {
	if c.Queries != nil {
		delete(c.Queries, name)
	}
}

// GetProfile returns a connection profile by name.
func (c *Config) GetProfile(name string) (Profile, bool) {
	if c.Profiles == nil {
		return Profile{}, false
	}
	p, ok := c.Profiles[name]
	return p, ok
}

// configPath returns the default config file path.
func configPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return configFileName
	}
	return filepath.Join(home, configDirName, configFileName)
}

// ConfigDir returns the configuration directory path.
func ConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return configDirName
	}
	return filepath.Join(home, configDirName)
}
