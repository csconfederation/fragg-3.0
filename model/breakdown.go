package model

import "fmt"

// RatingComponent represents one piece of the final rating formula.
type RatingComponent struct {
	Metric       string  `json:"metric"`
	Value        float64 `json:"value"`
	Baseline     float64 `json:"baseline,omitempty"`
	Multiplier   float64 `json:"multiplier"`
	Contribution float64 `json:"contribution"`
	Notes        string  `json:"notes,omitempty"`
}

// RatingBreakdown captures how the final rating value is composed.
type RatingBreakdown struct {
	Baseline         float64         `json:"baseline"`
	ADR              RatingComponent `json:"adr"`
	KAST             RatingComponent `json:"kast"`
	ProbabilitySwing RatingComponent `json:"probability_swing"`
	UnclampedRating  float64         `json:"unclamped_rating"`
	FinalRating      float64         `json:"final_rating"`
	Formula          string          `json:"formula"`
}

// RoundSwingBreakdown captures per-round swing context for a player.
type RoundSwingBreakdown struct {
	RoundNumber      int                 `json:"round_number"`
	ProbabilitySwing float64             `json:"probability_swing"`
	PlayerSide       string              `json:"player_side"`
	IsPistolRound    bool                `json:"is_pistol_round"`
	TeamWon          bool                `json:"team_won"`
	Kills            int                 `json:"kills"`
	Assists          int                 `json:"assists"`
	Damage           int                 `json:"damage"`
	OpeningKill      bool                `json:"opening_kill"`
	OpeningDeath     bool                `json:"opening_death"`
	TradeKill        bool                `json:"trade_kill"`
	TradeDeath       bool                `json:"trade_death"`
	ClutchAttempt    bool                `json:"clutch_attempt"`
	ClutchWon        bool                `json:"clutch_won"`
	BombPlanted      bool                `json:"bomb_planted"`
	BombDefused      bool                `json:"bomb_defused"`
	EcoKill          bool                `json:"eco_kill"`
	AntiEcoKill      bool                `json:"anti_eco_kill"`
	EntryFragger     bool                `json:"entry_fragger"`
	Survived         bool                `json:"survived"`
	ImpactFactors    []string            `json:"impact_factors"`
	Contributions    []SwingContribution `json:"contributions"`
}

// NewRoundSwingBreakdown converts round stats into a user-facing breakdown entry.
func NewRoundSwingBreakdown(roundNumber int, stats *RoundStats) RoundSwingBreakdown {
	breakdown := RoundSwingBreakdown{
		RoundNumber:      roundNumber,
		ProbabilitySwing: stats.ProbabilitySwing,
		PlayerSide:       stats.PlayerSide,
		IsPistolRound:    stats.IsPistolRound,
		TeamWon:          stats.TeamWon,
		Kills:            stats.Kills,
		Assists:          stats.Assists,
		Damage:           stats.Damage,
		OpeningKill:      stats.OpeningKill,
		OpeningDeath:     stats.OpeningDeath,
		TradeKill:        stats.TradeKill,
		TradeDeath:       stats.TradeDeath,
		ClutchAttempt:    stats.ClutchAttempt,
		ClutchWon:        stats.ClutchWon,
		BombPlanted:      stats.PlantedBomb,
		BombDefused:      stats.DefusedBomb,
		EcoKill:          stats.EcoKill,
		AntiEcoKill:      stats.AntiEcoKill,
		EntryFragger:     stats.EntryFragger,
		Survived:         stats.Survived,
		Contributions:    stats.SwingContributions,
	}

	factors := make([]string, 0, 8)
	if stats.Kills > 0 {
		factors = append(factors, fmt.Sprintf("%d kill(s)", stats.Kills))
	}
	if stats.Assists > 0 {
		factors = append(factors, fmt.Sprintf("%d assist(s)", stats.Assists))
	}
	if stats.OpeningKill {
		factors = append(factors, "Opening kill")
	}
	if stats.OpeningDeath {
		factors = append(factors, "Opening death")
	}
	if stats.TradeKill {
		factors = append(factors, "Trade kill")
	}
	if stats.TradeDeath {
		factors = append(factors, "Trade death")
	}
	if stats.PlantedBomb {
		factors = append(factors, "Bomb plant")
	}
	if stats.DefusedBomb {
		factors = append(factors, "Bomb defuse")
	}
	if stats.EcoKill {
		factors = append(factors, "Eco kill")
	}
	if stats.AntiEcoKill {
		factors = append(factors, "Anti-eco kill")
	}
	if stats.ClutchAttempt {
		label := "Clutch attempt"
		if stats.ClutchWon {
			label = "Clutch win"
		}
		factors = append(factors, label)
	}
	if stats.EntryFragger {
		factors = append(factors, "Entry frag")
	}
	if stats.Survived {
		factors = append(factors, "Round survival")
	}

	breakdown.ImpactFactors = factors
	return breakdown
}
