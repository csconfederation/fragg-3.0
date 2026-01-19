// =============================================================================
// DISCLAIMER: Comments in this file were generated with AI assistance to help
// users find and understand code for reference while building FraGG 3.0.
// =============================================================================

// Package model defines the core data structures for player and round statistics.
// This file provides a builder pattern for constructing RoundContext objects,
// making the creation of complex context objects more readable and maintainable.
package model

import "math"

// RoundContextBuilder provides a fluent interface for constructing RoundContext objects.
type RoundContextBuilder struct {
	ctx *RoundContext
}

// NewRoundContextBuilder creates a new builder with default values.
func NewRoundContextBuilder() *RoundContextBuilder {
	return &RoundContextBuilder{
		ctx: &RoundContext{
			TotalPlayers:    10,
			RoundImportance: 1.0,
		},
	}
}

// WithRoundNumber sets the round number.
func (b *RoundContextBuilder) WithRoundNumber(n int) *RoundContextBuilder {
	b.ctx.RoundNumber = n
	return b
}

// WithTotalPlayers sets the total number of players.
func (b *RoundContextBuilder) WithTotalPlayers(n int) *RoundContextBuilder {
	b.ctx.TotalPlayers = n
	return b
}

// WithBombPlanted sets whether the bomb was planted.
func (b *RoundContextBuilder) WithBombPlanted(planted bool) *RoundContextBuilder {
	b.ctx.BombPlanted = planted
	return b
}

// WithBombDefused sets whether the bomb was defused.
func (b *RoundContextBuilder) WithBombDefused(defused bool) *RoundContextBuilder {
	b.ctx.BombDefused = defused
	return b
}

// WithRoundType sets the round type (pistol, eco, force, full).
func (b *RoundContextBuilder) WithRoundType(roundType string) *RoundContextBuilder {
	b.ctx.RoundType = roundType
	return b
}

// WithTimeRemaining sets the time remaining in the round.
func (b *RoundContextBuilder) WithTimeRemaining(time float64) *RoundContextBuilder {
	b.ctx.TimeRemaining = time
	return b
}

// WithOvertime sets whether this is an overtime round.
func (b *RoundContextBuilder) WithOvertime(isOvertime bool) *RoundContextBuilder {
	b.ctx.IsOvertimeRound = isOvertime
	return b
}

// WithMapSide sets the map side (T or CT).
func (b *RoundContextBuilder) WithMapSide(side string) *RoundContextBuilder {
	b.ctx.MapSide = side
	return b
}

// WithScores sets the team and enemy scores.
func (b *RoundContextBuilder) WithScores(teamScore, enemyScore int) *RoundContextBuilder {
	b.ctx.TeamScore = teamScore
	b.ctx.EnemyScore = enemyScore
	b.ctx.ScoreDiff = teamScore - enemyScore
	return b
}

// WithMatchPoint sets whether this is a match point round.
func (b *RoundContextBuilder) WithMatchPoint(isMatchPoint bool) *RoundContextBuilder {
	b.ctx.IsMatchPoint = isMatchPoint
	return b
}

// WithCloseGame sets whether this is a close game.
func (b *RoundContextBuilder) WithCloseGame(isClose bool) *RoundContextBuilder {
	b.ctx.IsCloseGame = isClose
	return b
}

// WithRoundImportance sets the round importance multiplier.
func (b *RoundContextBuilder) WithRoundImportance(importance float64) *RoundContextBuilder {
	b.ctx.RoundImportance = importance
	return b
}

// WithRoundDecision sets the round decision state.
func (b *RoundContextBuilder) WithRoundDecision(decided bool, decidedAt float64) *RoundContextBuilder {
	b.ctx.RoundDecided = decided
	b.ctx.RoundDecidedAt = decidedAt
	return b
}

// CalculateImportance automatically calculates round importance based on scores.
func (b *RoundContextBuilder) CalculateImportance() *RoundContextBuilder {
	scoreDiff := b.ctx.TeamScore - b.ctx.EnemyScore
	isMatchPoint := b.ctx.TeamScore == 12 || b.ctx.EnemyScore == 12 ||
		(b.ctx.RoundNumber > 30 && (b.ctx.TeamScore >= 15 || b.ctx.EnemyScore >= 15))
	isCloseGame := math.Abs(float64(scoreDiff)) <= 3

	b.ctx.IsMatchPoint = isMatchPoint
	b.ctx.IsCloseGame = isCloseGame

	if isMatchPoint {
		b.ctx.RoundImportance = 1.3
	} else if isCloseGame {
		b.ctx.RoundImportance = 1.15
	} else if math.Abs(float64(scoreDiff)) >= 8 {
		b.ctx.RoundImportance = 0.85
	} else {
		b.ctx.RoundImportance = 1.0
	}

	return b
}

// Build returns the constructed RoundContext.
func (b *RoundContextBuilder) Build() *RoundContext {
	return b.ctx
}

// BuildFromRoundStats creates a RoundContext and checks round stats for bomb events.
func (b *RoundContextBuilder) BuildFromRoundStats(rounds map[uint64]*RoundStats) *RoundContext {
	for _, roundStats := range rounds {
		if roundStats.PlantedBomb {
			b.ctx.BombPlanted = true
		}
		if roundStats.DefusedBomb {
			b.ctx.BombDefused = true
		}
	}
	return b.ctx
}
