package swing

import (
	"maps"
	"math"
	"slices"

	"github.com/csconfederation/fragg-3.0/engine/fact"
)

// ResidualInput holds round-end context for residual swing allocation.
type ResidualInput struct {
	Winner           fact.Side
	CurrentWinProb   float64
	Players          map[uint64]PlayerRoundContext
	MaxDamageInRound int
}

// ApplyRoundResidual distributes uncredited probability to round outcome.
func ApplyRoundResidual(ledger *RoundSwingLedger, input ResidualInput, cfg Config) []PlayerSwingAllocation {
	if ledger == nil || ledger.ResidualApplied || !cfg.ResidualEnabled {
		return nil
	}
	if input.Winner != fact.SideT && input.Winner != fact.SideCT {
		return nil
	}

	residual := 1.0 - input.CurrentWinProb
	if residual <= 0 {
		return nil
	}
	if residual > cfg.ResidualMax {
		residual = cfg.ResidualMax
	}

	losingSide := fact.SideCT
	if input.Winner == fact.SideCT {
		losingSide = fact.SideT
	}

	positiveAllocs := allocatePositiveResidual(input, input.Winner, residual)
	negativeAllocs := allocateNegativeResidual(input, losingSide, residual, cfg)

	event := SwingLedgerEvent{
		RawDelta:      residual,
		NegativePool:  residual,
		PositiveAlloc: positiveAllocs,
		NegativeAlloc: negativeAllocs,
	}
	ledger.RecordEvent(event)
	ledger.ResidualApplied = true

	allocs := make([]PlayerSwingAllocation, 0, len(positiveAllocs)+len(negativeAllocs))
	allocs = append(allocs, positiveAllocs...)
	allocs = append(allocs, negativeAllocs...)
	return allocs
}

func allocatePositiveResidual(input ResidualInput, winner fact.Side, residual float64) []PlayerSwingAllocation {
	weights := make(map[uint64]float64)
	aliveWinners := make([]uint64, 0)

	maxDamage := float64(input.MaxDamageInRound)
	if maxDamage <= 0 {
		maxDamage = 1
	}

	for _, steamID := range slices.Sorted(maps.Keys(input.Players)) {
		ctx := input.Players[steamID]
		if ctx.Side != winner {
			continue
		}
		if ctx.Alive {
			aliveWinners = append(aliveWinners, steamID)
		}

		score := ctx.PositiveSwing*1.0 +
			(float64(ctx.Damage)/maxDamage)*0.25 +
			float64(ctx.FlashAssists)*0.15
		if ctx.PlantedBomb {
			score += 0.25
		}
		if ctx.DefusedBomb {
			score += 0.25
		}
		if score > 0 {
			weights[steamID] = score
		}
	}

	total := 0.0
	for _, id := range slices.Sorted(maps.Keys(weights)) {
		w := weights[id]
		total += w
	}

	if total <= 0 {
		if len(aliveWinners) == 0 {
			return nil
		}
		equal := residual / float64(len(aliveWinners))
		allocs := make([]PlayerSwingAllocation, 0, len(aliveWinners))
		for _, id := range aliveWinners {
			allocs = append(allocs, PlayerSwingAllocation{
				SteamID: id,
				Side:    winner,
				Amount:  equal,
				Reason:  SwingReasonRoundResidualWin,
			})
		}
		return allocs
	}

	allocs := make([]PlayerSwingAllocation, 0, len(weights))
	for _, steamID := range slices.Sorted(maps.Keys(weights)) {
		w := weights[steamID]
		allocs = append(allocs, PlayerSwingAllocation{
			SteamID: steamID,
			Side:    winner,
			Amount:  residual * (w / total),
			Reason:  SwingReasonRoundResidualWin,
		})
	}
	return allocs
}

func allocateNegativeResidual(input ResidualInput, losingSide fact.Side, residual float64, cfg Config) []PlayerSwingAllocation {
	weights := make(map[uint64]float64)

	for _, steamID := range slices.Sorted(maps.Keys(input.Players)) {
		ctx := input.Players[steamID]
		if ctx.Side != losingSide {
			continue
		}

		score := math.Abs(ctx.NegativeSwing)
		if ctx.Alive {
			score += cfg.SavePenaltyWeight
		}
		if score <= 0 {
			score = 1.0
		}
		weights[steamID] = score
	}

	total := 0.0
	for _, id := range slices.Sorted(maps.Keys(weights)) {
		w := weights[id]
		total += w
	}
	if total <= 0 {
		return nil
	}

	allocs := make([]PlayerSwingAllocation, 0, len(weights))
	for _, steamID := range slices.Sorted(maps.Keys(weights)) {
		w := weights[steamID]
		allocs = append(allocs, PlayerSwingAllocation{
			SteamID: steamID,
			Side:    losingSide,
			Amount:  -residual * (w / total),
			Reason:  SwingReasonRoundResidualLoss,
		})
	}
	return allocs
}
