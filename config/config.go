package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Cloudflare    CloudflareConfig `yaml:"cloudflare"`
	CheckInterval string           `yaml:"check_interval"`
	Records       []DNSRecord      `yaml:"records"`
}

// CloudflareConfig holds Cloudflare API credentials
type CloudflareConfig struct {
	APIToken string `yaml:"api_token"`
}

// DNSRecord represents a DNS record to update
type DNSRecord struct {
	ZoneID  string   `yaml:"zone_id"`
	Name    string   `yaml:"name"`
	Types   []string `yaml:"types"` // A, AAAA
	TTL     int      `yaml:"ttl"`
	Proxied bool     `yaml:"proxied"`
}

// Load reads and parses the configuration file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Cloudflare.APIToken == "" {
		return fmt.Errorf("cloudflare.api_token is required")
	}

	if c.CheckInterval == "" {
		return fmt.Errorf("check_interval is required")
	}

	// Validate check interval format
	if _, err := time.ParseDuration(c.CheckInterval); err != nil {
		return fmt.Errorf("invalid check_interval format: %w", err)
	}

	if len(c.Records) == 0 {
		return fmt.Errorf("at least one DNS record must be configured")
	}

	for i, record := range c.Records {
		if record.ZoneID == "" {
			return fmt.Errorf("record %d: zone_id is required", i)
		}
		if record.Name == "" {
			return fmt.Errorf("record %d: name is required", i)
		}
		if len(record.Types) == 0 {
			return fmt.Errorf("record %d: at least one type (A or AAAA) is required", i)
		}
		for _, t := range record.Types {
			if t != "A" && t != "AAAA" {
				return fmt.Errorf("record %d: invalid type %s (must be A or AAAA)", i, t)
			}
		}
		if record.TTL < 60 || record.TTL > 86400 {
			return fmt.Errorf("record %d: ttl must be between 60 and 86400", i)
		}
	}

	return nil
}

// GetCheckInterval returns the check interval as a duration
func (c *Config) GetCheckInterval() time.Duration {
	duration, _ := time.ParseDuration(c.CheckInterval)
	return duration
}
