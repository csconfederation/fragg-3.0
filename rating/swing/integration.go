package swing

import (
	"github.com/ethsmith/eco-rating/model"
	"github.com/ethsmith/eco-rating/rating/probability"
)

// RatingIntegration provides methods to convert swing values to rating contributions.
type RatingIntegration struct {
	scaleToRating float64
	baseline      float64
	minRating     float64
	maxRating     float64
}

// NewRatingIntegration creates a new rating integration with default parameters.
func NewRatingIntegration() *RatingIntegration {
	return &RatingIntegration{
		scaleToRating: probability.SwingToRatingScale,
		baseline:      probability.SwingRatingBaseline,
		minRating:     probability.MinSwingRating,
		maxRating:     probability.MaxSwingRating,
	}
}

// SwingToRating converts an average swing per round to a rating component.
// Input: average swing per round (e.g., +0.03 = +3% per round)
// Output: rating component centered around 1.0
func (r *RatingIntegration) SwingToRating(avgSwingPerRound float64) float64 {
	// Scale: +4% avg swing -> 1.40 rating
	//        0% avg swing -> 1.00 rating
	//       -3% avg swing -> 0.70 rating
	rating := r.baseline + (avgSwingPerRound * r.scaleToRating)

	return clamp(rating, r.minRating, r.maxRating)
}

// ComputeSwingRating calculates the swing rating from player stats.
func (r *RatingIntegration) ComputeSwingRating(p *model.PlayerStats) float64 {
	if p.RoundsPlayed == 0 {
		return r.baseline
	}

	avgSwing := p.ProbabilitySwing / float64(p.RoundsPlayed)
	return r.SwingToRating(avgSwing)
}

// UpdatePlayerSwingMetrics updates all swing-related fields in PlayerStats.
func (r *RatingIntegration) UpdatePlayerSwingMetrics(p *model.PlayerStats) {
	if p.RoundsPlayed == 0 {
		return
	}

	rounds := float64(p.RoundsPlayed)

	// Calculate per-round metrics
	p.ProbabilitySwingPerRound = p.ProbabilitySwing / rounds

	// Calculate swing rating contribution
	p.SwingRating = r.SwingToRating(p.ProbabilitySwingPerRound)
}

// clamp restricts a value to the range [min, max].
func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// RatingWeights defines the weights for each sub-rating in the final formula.
type RatingWeights struct {
	Kill      float64
	Damage    float64
	Survival  float64
	KAST      float64
	MultiKill float64
	Swing     float64
}

// DefaultRatingWeights returns the default weights for rating v3.0.
func DefaultRatingWeights() RatingWeights {
	return RatingWeights{
		Kill:      0.20,
		Damage:    0.15,
		Survival:  0.10,
		KAST:      0.15,
		MultiKill: 0.15,
		Swing:     0.25, // Significant weight for swing
	}
}

// ValidateWeights checks that weights sum to 1.0.
func (w RatingWeights) ValidateWeights() bool {
	sum := w.Kill + w.Damage + w.Survival + w.KAST + w.MultiKill + w.Swing
	return sum >= 0.99 && sum <= 1.01
}
