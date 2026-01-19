// =============================================================================
// DISCLAIMER: Comments in this file were generated with AI assistance to help
// users find and understand code for reference while building FraGG 3.0.
// =============================================================================

// Package rating implements the eco-rating calculation system.
// This file contains HLTV 2.0 rating calculation functions that provide
// a single source of truth for rating computations used across the codebase.
package rating

// HLTVInput contains the raw statistics needed to compute an HLTV 2.0 rating.
// This struct provides a clean interface for rating calculations.
type HLTVInput struct {
	RoundsPlayed int
	Kills        int
	Deaths       int
	Survivals    int
	MultiKills   [6]int // Index 0 unused, 1-5 for 1K through 5K
}

// ComputeHLTVRating calculates the HLTV 2.0 rating from raw statistics.
// This is the single source of truth for HLTV rating calculations.
// The formula combines kill rating, survival rating, and round multi-kill rating.
func ComputeHLTVRating(input HLTVInput) float64 {
	if input.RoundsPlayed == 0 {
		return 0
	}

	rounds := float64(input.RoundsPlayed)

	// Kill rating component
	kpr := float64(input.Kills) / rounds
	killRating := kpr / HLTVBaselineKPR

	// Survival rating component
	survivalRating := ((float64(input.Survivals) - float64(input.Deaths)) / rounds) / HLTVBaselineSPR

	// Round multi-kill rating component
	rmkPoints := ComputeRMKPoints(input.MultiKills)
	rmkRating := (float64(rmkPoints) / rounds) / HLTVBaselineRMK

	return (killRating + HLTVSurvivalWeight*survivalRating + rmkRating) / HLTVRatingDivisor
}

// ComputeRMKPoints calculates the round multi-kill points from a multi-kill array.
// Points are weighted: 1K=1, 2K=4, 3K=9, 4K=16, 5K=25 (squared values).
func ComputeRMKPoints(multiKills [6]int) int {
	return multiKills[1]*1 + multiKills[2]*4 + multiKills[3]*9 + multiKills[4]*16 + multiKills[5]*25
}

// ComputePistolRoundRating calculates the HLTV-style rating for pistol rounds only.
func ComputePistolRoundRating(roundsPlayed, kills, deaths, survivals, multiKills int) float64 {
	if roundsPlayed == 0 {
		return 0
	}

	rounds := float64(roundsPlayed)

	// Kill rating
	kpr := float64(kills) / rounds
	killRating := kpr / HLTVBaselineKPR

	// Survival rating
	survivalRating := ((float64(survivals) - float64(deaths)) / rounds) / HLTVBaselineSPR

	// Multi-kill rating (simplified: each 2K+ counts as 4 points)
	rmkPoints := float64(multiKills) * 4.0
	rmkRating := (rmkPoints / rounds) / HLTVBaselineRMK

	return (killRating + HLTVSurvivalWeight*survivalRating + rmkRating) / HLTVRatingDivisor
}

// ComputeSideHLTVRating calculates HLTV rating for a specific side (T or CT).
func ComputeSideHLTVRating(roundsPlayed, kills, deaths, survivals int, multiKills [6]int) float64 {
	return ComputeHLTVRating(HLTVInput{
		RoundsPlayed: roundsPlayed,
		Kills:        kills,
		Deaths:       deaths,
		Survivals:    survivals,
		MultiKills:   multiKills,
	})
}
