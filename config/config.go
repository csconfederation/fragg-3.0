// =============================================================================
// DISCLAIMER: Comments in this file were generated with AI assistance to help
// users find and understand code for reference while building FraGG 3.0.
// =============================================================================

// Package config handles application configuration loading, saving, and validation.
// It supports JSON configuration files and provides sensible defaults for all settings.
package config

import (
	"encoding/json"
	"os"
	"strings"
)

// Config holds all application configuration settings.
// These can be set via JSON config file or command-line flags.
type Config struct {
	Cumulative    bool     `json:"cumulative"`     // Enable batch processing mode
	Tier          string   `json:"tier"`           // Competitive tier filter (comma-separated for multiple)
	BaseURL       string   `json:"base_url"`       // Cloud bucket base URL
	Prefixes      []string `json:"prefixes"`       // Bucket prefixes for demo files (multiple paths)
	DemoPath      string   `json:"demo_path"`      // Path to single demo file (single mode)
	DemoDir       string   `json:"demo_dir"`       // Local directory for downloaded demos
	EnableLogging bool     `json:"enable_logging"` // Enable detailed parsing logs
}

// DefaultConfig returns a Config with sensible default values.
// The defaults point to the CSC demo bucket for season 19 combines.
func DefaultConfig() *Config {
	return &Config{
		Cumulative:    false,
		Tier:          "",
		BaseURL:       "https://cscdemos.nyc3.digitaloceanspaces.com/",
		Prefixes:      []string{"s19/Combines/"},
		DemoPath:      "",
		DemoDir:       "./demos",
		EnableLogging: true,
	}
}

// LoadConfig reads configuration from a JSON file at the given path.
// If the file doesn't exist, it returns default configuration.
func LoadConfig(path string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// ValidTiers returns the list of valid competitive tier names.
// Tiers are ordered from highest to lowest skill level.
func ValidTiers() []string {
	return []string{
		"challenger",
		"contender",
		"elite",
		"premier",
		"prospect",
		"recruit",
	}
}

// IsValidTier checks if the given tier name is usable.
// Accepts standard tiers (challenger, contender, etc.), "all", or any
// non-empty string which is treated as a team name filter.
func IsValidTier(tier string) bool {
	return tier != ""
}

// IsStandardTier returns true if the tier is one of the 6 known competitive tiers.
func IsStandardTier(tier string) bool {
	for _, t := range ValidTiers() {
		if t == tier {
			return true
		}
	}
	return false
}

// IsAllTier returns true if the tier value means "fetch all demos".
func IsAllTier(tier string) bool {
	return tier == "all"
}

// IsTeamFilter returns true if the tier value is a team name filter
// (not a standard tier and not "all").
func IsTeamFilter(tier string) bool {
	return tier != "" && !IsStandardTier(tier) && !IsAllTier(tier)
}

// ParseTiers splits a comma-separated tier string into individual tier names.
// It trims whitespace and filters out empty strings.
func ParseTiers(tierStr string) []string {
	if tierStr == "" {
		return nil
	}
	parts := strings.Split(tierStr, ",")
	tiers := make([]string, 0, len(parts))
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t != "" {
			tiers = append(tiers, t)
		}
	}
	return tiers
}
