// Package config loads wwlog settings from ~/.config/wwlog/config.toml.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config holds user-configurable settings.
type Config struct {
	TLD        string `mapstructure:"tld"`
	Theme      string `mapstructure:"theme"`
	CacheTTL   int    `mapstructure:"cache_ttl"`
	WeightUnit string `mapstructure:"weight_unit"`
	StoreDir   string `mapstructure:"store_dir"`
}

// DefaultStoreDir returns the platform default location for the local day-log
// store: alongside the config directory under a "store" subdirectory.
// On macOS: ~/Library/Application Support/wwlog/store
// On Linux: ~/.config/wwlog/store
func DefaultStoreDir() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, ".config", "wwlog", "store")
		}
		return ""
	}
	return filepath.Join(dir, "wwlog", "store")
}

// Load reads ~/.config/wwlog/config.toml, applying defaults where absent.
func Load() (*Config, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return defaults(), fmt.Errorf("locate config dir: %w", err)
	}

	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("toml")
	v.AddConfigPath(filepath.Join(dir, "wwlog"))
	// On macOS, UserConfigDir() returns ~/Library/Application Support; also check the
	// XDG-style ~/.config/wwlog path that most users expect on Unix.
	if home, err := os.UserHomeDir(); err == nil {
		v.AddConfigPath(filepath.Join(home, ".config", "wwlog"))
	}

	v.SetDefault("tld", "com")
	v.SetDefault("theme", "auto")
	v.SetDefault("cache_ttl", 3600)
	v.SetDefault("weight_unit", "")
	v.SetDefault("store_dir", "")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return defaults(), err
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return defaults(), err
	}
	// Read weight_unit explicitly — Viper may not surface unregistered keys via Unmarshal.
	if wu := v.GetString("weight_unit"); wu != "" {
		cfg.WeightUnit = wu
	}
	return &cfg, nil
}

func defaults() *Config {
	return &Config{TLD: "com", Theme: "auto", CacheTTL: 3600}
}
