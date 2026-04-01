package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	WatchDir            string   `yaml:"watch_dir"`
	UploadedDir         string   `yaml:"uploaded_dir"`
	FailedDir           string   `yaml:"failed_dir"`
	LogDir              string   `yaml:"log_dir"`
	AllowedExtensions   []string `yaml:"allowed_extensions"`
	StabilizeSeconds    int      `yaml:"stabilize_seconds"`
	ScanIntervalSeconds int      `yaml:"scan_interval_seconds"`
	MaxRetry            int      `yaml:"max_retry"`

	R2 struct {
		AccountID       string `yaml:"account_id"`
		AccessKeyID     string `yaml:"access_key_id"`
		SecretAccessKey string `yaml:"secret_access_key"`
		Bucket          string `yaml:"bucket"`
		Endpoint        string `yaml:"endpoint"`
	} `yaml:"r2"`

	Backend struct {
		BaseURL        string `yaml:"base_url"`
		APIKey         string `yaml:"api_key"`
		TimeoutSeconds int    `yaml:"timeout_seconds"`
	} `yaml:"backend"`
}

func LoadConfig(path string) (*Config, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file error: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(file, &cfg); err != nil {
		return nil, fmt.Errorf("parse yaml error: %w", err)
	}

	// validate basic
	if cfg.WatchDir == "" {
		return nil, fmt.Errorf("watch_dir is required")
	}

	if cfg.R2.Bucket == "" {
		return nil, fmt.Errorf("r2.bucket is required")
	}

	if cfg.R2.Endpoint == "" {
		return nil, fmt.Errorf("r2.endpoint is required")
	}

	return &cfg, nil
}
