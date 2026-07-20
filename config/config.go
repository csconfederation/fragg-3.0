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

	"github.com/ethsmith/eco-rating/rating/swing"
)

// Config holds all application configuration settings.
// These can be set via JSON config file or command-line flags.
type Config struct {
	Cumulative       bool     `json:"cumulative"`     // Enable batch processing mode
	Tier             string   `json:"tier"`           // Competitive tier filter (comma-separated for multiple)
	BaseURL          string   `json:"base_url"`       // Cloud bucket base URL
	Prefixes         []string `json:"prefixes"`       // Bucket prefixes for demo files (multiple paths)
	DemoPath         string   `json:"demo_path"`      // Path to single demo file (single mode)
	DemoDir          string   `json:"demo_dir"`       // Local directory for downloaded demos
	EnableLogging    bool     `json:"enable_logging"` // Enable detailed parsing logs
	IgnoreScrims     bool     `json:"ignore_scrims"`
	KDPRModifier     bool     `json:"kdpr_modifier"`     // Enable KPR/DPR rating adjustment
	Workers          int      `json:"workers"`           // Number of parallel parsing workers (0 = auto)
	GenerateFiles    bool     `json:"generate_files"`    // Generate stats.csv and probability_data.json files
	CSCCompatibility bool     `json:"csc_compatibility"` // Output demoScrape2-compatible JSON (mutually exclusive with cumulative)

	// Swing parser settings (see rating/swing/config.go for defaults)
	SwingExitFrags              string  `json:"swing_exit_frags"`
	SwingResidualEnabled        bool    `json:"swing_residual_enabled"`
	SwingResidualMax            float64 `json:"swing_residual_max"`
	SwingSavePenaltyWeight      float64 `json:"swing_save_penalty_weight"`
	SwingPlantPlanterShare      float64 `json:"swing_plant_planter_share"`
	SwingDefuseDefuserShare     float64 `json:"swing_defuse_defuser_share"`
	SwingKillFinalHitBase       float64 `json:"swing_kill_final_hit_base"`
	SwingKillDamageShareWeight  float64 `json:"swing_kill_damage_share_weight"`
	SwingTradeKillMultiplier    float64 `json:"swing_trade_kill_multiplier"`
	SwingSurvivalCreditEnabled  bool    `json:"swing_survival_credit_enabled"`
	SwingSurvivalCreditMaxShare float64 `json:"swing_survival_credit_max_share"`
	SwingZeroSumTolerance       float64 `json:"swing_zero_sum_tolerance"`
}

// DefaultConfig returns a Config with sensible default values.
// The defaults point to the CSC demo bucket for season 19 combines.
func DefaultConfig() *Config {
	return &Config{
		Cumulative:       false,
		Tier:             "",
		BaseURL:          "https://cscdemos.nyc3.digitaloceanspaces.com/",
		Prefixes:         []string{"s19/Combines/"},
		DemoPath:         "",
		DemoDir:          "./demos",
		EnableLogging:    true,
		IgnoreScrims:     false,
		KDPRModifier:     false,
		Workers:          4,     // Parallel demo parsers (each uses significant RAM)
		GenerateFiles:    true,  // Generate output files by default
		CSCCompatibility: false, // Disabled by default

		SwingExitFrags:              "zero",
		SwingResidualEnabled:        true,
		SwingResidualMax:            0.35,
		SwingSavePenaltyWeight:      0.35,
		SwingPlantPlanterShare:      0.45,
		SwingDefuseDefuserShare:     0.60,
		SwingKillFinalHitBase:       0.35,
		SwingKillDamageShareWeight:  0.45,
		SwingTradeKillMultiplier:    0.80,
		SwingSurvivalCreditEnabled:  false,
		SwingSurvivalCreditMaxShare: 0.05,
		SwingZeroSumTolerance:       0.005,
	}
}

// SwingConfig returns swing settings as a rating/swing.Config.
func (c *Config) SwingConfig() swing.Config {
	cfg := swing.DefaultConfig()
	if c == nil {
		return cfg
	}
	cfg.ExitFrags = swing.ExitFragMode(c.SwingExitFrags)
	cfg.ResidualEnabled = c.SwingResidualEnabled
	if c.SwingResidualMax > 0 {
		cfg.ResidualMax = c.SwingResidualMax
	}
	if c.SwingSavePenaltyWeight > 0 {
		cfg.SavePenaltyWeight = c.SwingSavePenaltyWeight
	}
	if c.SwingPlantPlanterShare > 0 {
		cfg.PlantPlanterShare = c.SwingPlantPlanterShare
	}
	if c.SwingDefuseDefuserShare > 0 {
		cfg.DefuseDefuserShare = c.SwingDefuseDefuserShare
	}
	if c.SwingKillFinalHitBase > 0 {
		cfg.KillFinalHitBase = c.SwingKillFinalHitBase
	}
	if c.SwingKillDamageShareWeight > 0 {
		cfg.KillDamageShareWeight = c.SwingKillDamageShareWeight
	}
	if c.SwingTradeKillMultiplier > 0 {
		cfg.TradeKillMultiplier = c.SwingTradeKillMultiplier
	}
	cfg.SurvivalCreditEnabled = c.SwingSurvivalCreditEnabled
	if c.SwingSurvivalCreditMaxShare > 0 {
		cfg.SurvivalCreditMaxShare = c.SwingSurvivalCreditMaxShare
	}
	if c.SwingZeroSumTolerance > 0 {
		cfg.ZeroSumTolerance = c.SwingZeroSumTolerance
	}
	return cfg
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
