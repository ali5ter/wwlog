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
	TLD      string `mapstructure:"tld"`
	Theme    string `mapstructure:"theme"`
	CacheTTL int    `mapstructure:"cache_ttl"`
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

	v.SetDefault("tld", "com")
	v.SetDefault("theme", "auto")
	v.SetDefault("cache_ttl", 3600)

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return defaults(), err
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return defaults(), err
	}
	return &cfg, nil
}

func defaults() *Config {
	return &Config{TLD: "com", Theme: "auto", CacheTTL: 3600}
}
