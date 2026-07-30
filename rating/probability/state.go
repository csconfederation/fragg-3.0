package probability

import (
	"fmt"

	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
)

// EconomyCategory represents equipment value tiers for probability calculations.
type EconomyCategory int

const (
	EcoStarterPistol  EconomyCategory = iota // $0-1000 (Glock, USP, no armor)
	EcoUpgradedPistol                        // $1000-2000 (Deagle, P250 + armor)
	EcoSMG                                   // $2000-3500 (SMGs, shotguns)
	EcoRifle                                 // $3500-4750 (AK, M4, etc.)
	EcoAWP                                   // $4750+ (AWP loadout)
)

// String returns the string representation of an EconomyCategory.
func (e EconomyCategory) String() string {
	switch e {
	case EcoStarterPistol:
		return "starter_pistol"
	case EcoUpgradedPistol:
		return "upgraded_pistol"
	case EcoSMG:
		return "smg"
	case EcoRifle:
		return "rifle"
	case EcoAWP:
		return "awp"
	default:
		return "unknown"
	}
}

// CategorizeEquipment converts an equipment value to an EconomyCategory.
func CategorizeEquipment(equipValue float64) EconomyCategory {
	switch {
	case equipValue >= 4750:
		return EcoAWP
	case equipValue >= 3500:
		return EcoRifle
	case equipValue >= 2000:
		return EcoSMG
	case equipValue >= 1000:
		return EcoUpgradedPistol
	default:
		return EcoStarterPistol
	}
}

// RoundState represents the current state of a round for probability calculations.
type RoundState struct {
	TAlive        int             // Number of terrorists alive (0-5)
	CTAlive       int             // Number of CTs alive (0-5)
	BombPlanted   bool            // Whether the bomb has been planted
	BombDefused   bool            // Whether the bomb has been defused
	TimeRemaining float64         // Seconds remaining in the round
	TEconomy      EconomyCategory // T side average economy category
	CTEconomy     EconomyCategory // CT side average economy category
	Map           string          // Map name (de_dust2, de_inferno, etc.)
}

// NewRoundState creates a new RoundState with initial values.
func NewRoundState(tAlive, ctAlive int, mapName string) *RoundState {
	return &RoundState{
		TAlive:        tAlive,
		CTAlive:       ctAlive,
		BombPlanted:   false,
		BombDefused:   false,
		TimeRemaining: 115.0, // Default round time
		TEconomy:      EcoRifle,
		CTEconomy:     EcoRifle,
		Map:           mapName,
	}
}

// Clone creates a deep copy of the RoundState.
func (s *RoundState) Clone() *RoundState {
	return &RoundState{
		TAlive:        s.TAlive,
		CTAlive:       s.CTAlive,
		BombPlanted:   s.BombPlanted,
		BombDefused:   s.BombDefused,
		TimeRemaining: s.TimeRemaining,
		TEconomy:      s.TEconomy,
		CTEconomy:     s.CTEconomy,
		Map:           s.Map,
	}
}

// RecordDeath updates the state when a player dies.
func (s *RoundState) RecordDeath(side common.Team) {
	if side == common.TeamTerrorists {
		if s.TAlive > 0 {
			s.TAlive--
		}
	} else if side == common.TeamCounterTerrorists {
		if s.CTAlive > 0 {
			s.CTAlive--
		}
	}
}

// SetBombPlanted marks the bomb as planted.
func (s *RoundState) SetBombPlanted() {
	s.BombPlanted = true
}

// SetBombDefused marks the bomb as defused.
func (s *RoundState) SetBombDefused() {
	s.BombDefused = true
}

// SetTimeRemaining sets seconds remaining on the round/bomb clock.
func (s *RoundState) SetTimeRemaining(seconds float64) {
	if seconds < 0 {
		seconds = 0
	}
	s.TimeRemaining = seconds
}

// StateKey returns a unique key for this state for table lookups.
func (s *RoundState) StateKey() string {
	bombStatus := "none"
	if s.BombPlanted {
		bombStatus = "planted"
	}
	if s.BombDefused {
		bombStatus = "defused"
	}
	return stateKeyFromComponents(s.TAlive, s.CTAlive, bombStatus)
}

// stateKeyFromComponents builds a state key from individual components.
// Format: "5v4_none" or "3v2_planted"
func stateKeyFromComponents(tAlive, ctAlive int, bombStatus string) string {
	return fmt.Sprintf("%dv%d_%s", tAlive, ctAlive, bombStatus)
}
