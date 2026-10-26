package swing

import (
	"fmt"
	"math"

	"github.com/csconfederation/fragg-3.0/engine/fact"
)

// SwingReason identifies why swing was allocated to a player.
type SwingReason string

const (
	SwingReasonKillFinalHit      SwingReason = "kill_final_hit"
	SwingReasonKillDamageShare   SwingReason = "kill_damage_share"
	SwingReasonVictimDeath       SwingReason = "victim_death"
	SwingReasonTradeRefund       SwingReason = "trade_refund"
	SwingReasonBombPlant         SwingReason = "bomb_plant"
	SwingReasonPlantSupport      SwingReason = "plant_support"
	SwingReasonBombDefuse        SwingReason = "bomb_defuse"
	SwingReasonDefuseSupport     SwingReason = "defuse_support"
	SwingReasonRoundResidualWin  SwingReason = "round_residual_win"
	SwingReasonRoundResidualLoss SwingReason = "round_residual_loss"
	SwingReasonExitFragIgnored   SwingReason = "exit_frag_ignored"
	SwingReasonSurvivalCredit    SwingReason = "survival_credit"
)

// PlayerSwingAllocation records a single swing credit or debit.
type PlayerSwingAllocation struct {
	SteamID uint64
	Side    fact.Side
	Amount  float64
	Reason  SwingReason
}

// SwingLedgerEvent records one event's pool allocation before applying to stats.
type SwingLedgerEvent struct {
	RawDelta      float64
	NegativePool  float64
	PositiveAlloc []PlayerSwingAllocation
	NegativeAlloc []PlayerSwingAllocation
}

// RoundSwingLedger tracks per-round swing accounting for zero-sum validation.
type RoundSwingLedger struct {
	TeamPositive map[fact.Side]float64
	TeamNegative map[fact.Side]float64

	PlayerSwing map[uint64]float64

	ResidualApplied bool
	DebugWarnings   []string
}

// NewRoundSwingLedger creates a fresh ledger for a round.
func NewRoundSwingLedger() *RoundSwingLedger {
	return &RoundSwingLedger{
		TeamPositive: make(map[fact.Side]float64),
		TeamNegative: make(map[fact.Side]float64),
		PlayerSwing:  make(map[uint64]float64),
	}
}

// RecordEvent appends an event and applies all allocations to player totals.
func (l *RoundSwingLedger) RecordEvent(event SwingLedgerEvent) {
	for _, alloc := range event.PositiveAlloc {
		l.applyAllocation(alloc, true)
	}
	for _, alloc := range event.NegativeAlloc {
		l.applyAllocation(alloc, false)
	}
}

// ApplyAllocations adds swing amounts directly (e.g. trade reallocation).
func (l *RoundSwingLedger) ApplyAllocations(allocs []PlayerSwingAllocation) {
	for _, alloc := range allocs {
		positive := alloc.Amount > 0
		l.applyAllocation(alloc, positive)
	}
}

func (l *RoundSwingLedger) applyAllocation(alloc PlayerSwingAllocation, positiveSide bool) {
	if alloc.Amount == 0 {
		return
	}
	l.PlayerSwing[alloc.SteamID] += alloc.Amount
	if alloc.Amount > 0 {
		l.TeamPositive[alloc.Side] += alloc.Amount
	} else {
		l.TeamNegative[alloc.Side] += alloc.Amount
	}
}

// ValidateRoundSwingLedger checks that player swing sums to approximately zero.
func ValidateRoundSwingLedger(ledger *RoundSwingLedger, tolerance float64) []string {
	if ledger == nil {
		return nil
	}

	total := 0.0
	for _, swing := range ledger.PlayerSwing {
		total += swing
	}

	if math.Abs(total) > tolerance {
		warning := fmt.Sprintf("round swing not zero-sum: total=%f", total)
		ledger.DebugWarnings = append(ledger.DebugWarnings, warning)
		return ledger.DebugWarnings
	}
	return ledger.DebugWarnings
}
