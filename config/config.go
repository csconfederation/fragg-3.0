package config

import (
	"encoding/json"
	"os"
	"strings"
)

type Config struct {
	// Cumulative mode - when true, fetches all demos from specified tier across all combine days
	Cumulative bool `json:"cumulative"`

	// Tier to filter demos by (e.g., "contender", "challenger", "elite", "premier", "prospect", "recruit")
	Tier string `json:"tier"`

	// Base URL for the S3 bucket
	BaseURL string `json:"base_url"`

	// Prefix for the combines folder (e.g., "s19/Combines/")
	Prefix string `json:"prefix"`

	// Output directory for downloaded demos
	OutputDir string `json:"output_dir"`

	// Path to a single demo file (used when cumulative=false)
	DemoPath string `json:"demo_path"`

	// Enable logging during parsing
	EnableLogging bool `json:"enable_logging"`

	// Enable CS Demo Manager integration (analyze + heatmaps)
	EnableCsdm bool `json:"enable_csdm"`

	// Output directory for generated heatmaps
	HeatmapPath string `json:"heatmap_path"`

	// Google Sheets integration
	StatsSheetURL  string `json:"stats_sheet_url"`
	StatsSheetName string `json:"stats_sheet_name"`
	UploadToSheet  bool   `json:"upload_to_sheet"`
}

// DefaultConfig returns a config with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Cumulative:    false,
		Tier:          "",
		BaseURL:       "https://cscdemos.nyc3.digitaloceanspaces.com/",
		Prefix:        "s19/Combines/",
		OutputDir:     "./demos",
		DemoPath:      "",
		EnableLogging: true,
		EnableCsdm:    true,
		HeatmapPath:   "./heatmaps",
	}
}

// LoadConfig loads configuration from a JSON file
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

// SaveConfig saves the configuration to a JSON file
func SaveConfig(cfg *Config, path string) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// ValidTiers returns the list of valid tier names
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

// IsValidTier checks if the given tier is valid
func IsValidTier(tier string) bool {
	for _, t := range ValidTiers() {
		if t == tier {
			return true
		}
	}
	return false
}

// DemoPrefix returns the prefix used to filter demo files for a given tier
func DemoPrefix(tier string) string {
	return "combine-" + tier
}

// ParseTiers splits a comma-separated tier string into individual tiers
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
