// Package config loads wwlog settings from ~/.config/wwlog/config.toml.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Targets holds optional user-configured nutrition and points reference values.
// All fields are optional; when absent the TUI and reports fall back to generic
// reference daily values. Configure in ~/.config/wwlog/config.toml under [targets].
//
// Range bands ([min, max]) render as a target window on bars and in reports.
// Single _min values are floors (more is fine); single _max values are ceilings (less is better).
type Targets struct {
	Calories         []float64 `mapstructure:"calories"`            // [min, max] kcal/day band
	ProteinG         []float64 `mapstructure:"protein_g"`           // [min, max] g/day
	CarbsG           []float64 `mapstructure:"carbs_g"`             // [min, max] g/day
	FatG             []float64 `mapstructure:"fat_g"`               // [min, max] g/day
	FiberGMin        float64   `mapstructure:"fiber_g_min"`         // floor g/day
	SodiumMgMax      float64   `mapstructure:"sodium_mg_max"`       // ceiling mg/day
	AddedSugarGMax   float64   `mapstructure:"added_sugar_g_max"`   // ceiling g/day
	SaturatedFatGMax float64   `mapstructure:"saturated_fat_g_max"` // ceiling g/day
	PointsDaily      float64   `mapstructure:"points_daily"`        // override WW daily target
	WaterFlOzTarget  float64   `mapstructure:"water_fl_oz_target"`  // floor fl oz/day (default: 64)
}

// HasAny reports whether any nutrition target is configured.
func (t *Targets) HasAny() bool {
	if t == nil {
		return false
	}
	return len(t.Calories) > 0 || len(t.ProteinG) > 0 || len(t.CarbsG) > 0 ||
		len(t.FatG) > 0 || t.FiberGMin > 0 || t.SodiumMgMax > 0 ||
		t.AddedSugarGMax > 0 || t.SaturatedFatGMax > 0 || t.PointsDaily > 0 || t.WaterFlOzTarget > 0
}

// WaterTarget returns the effective daily water target in fl oz.
// Falls back to 64 fl oz (WW's 8-glass suggestion) when not configured.
func (t *Targets) WaterTarget() float64 {
	if t != nil && t.WaterFlOzTarget > 0 {
		return t.WaterFlOzTarget
	}
	return 64
}

// Config holds user-configurable settings.
type Config struct {
	TLD        string  `mapstructure:"tld"`
	WeightUnit string  `mapstructure:"weight_unit"`
	StoreDir   string  `mapstructure:"store_dir"`
	Targets    Targets `mapstructure:"targets"`
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
	// Read string keys explicitly — Viper may not surface config-file values via Unmarshal
	// when the key was also registered with SetDefault (known Viper quirk).
	if wu := v.GetString("weight_unit"); wu != "" {
		cfg.WeightUnit = wu
	}
	if sd := v.GetString("store_dir"); sd != "" {
		cfg.StoreDir = sd
	}
	return &cfg, nil
}

func defaults() *Config {
	return &Config{TLD: "com"}
}
