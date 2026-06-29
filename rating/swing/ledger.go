package swing

import (
	"fmt"
	"math"

	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
)

// SwingReason identifies why swing was allocated to a player.
type SwingReason string

const (
	SwingReasonKillFinalHit      SwingReason = "kill_final_hit"
	SwingReasonKillDamageShare   SwingReason = "kill_damage_share"
	SwingReasonFlashAssist       SwingReason = "flash_assist"
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

// SwingEventType identifies the source event for a ledger entry.
type SwingEventType int

const (
	SwingEventKill SwingEventType = iota
	SwingEventBombPlant
	SwingEventBombDefuse
	SwingEventResidual
	SwingEventTradeReallocation
	SwingEventExitFrag
)

// PlayerSwingAllocation records a single swing credit or debit.
type PlayerSwingAllocation struct {
	SteamID uint64
	Side    common.Team
	Amount  float64
	Reason  SwingReason
}

// SwingLedgerEvent records one event's pool allocation before applying to stats.
type SwingLedgerEvent struct {
	Tick int
	Type SwingEventType

	BenefitingSide common.Team
	HurtSide       common.Team

	RawDelta      float64
	PositivePool  float64
	NegativePool  float64
	PositiveAlloc []PlayerSwingAllocation
	NegativeAlloc []PlayerSwingAllocation

	IsExitFrag bool
	IsResidual bool
}

// RoundSwingLedger tracks per-round swing accounting for zero-sum validation.
type RoundSwingLedger struct {
	RoundNumber int

	Events []SwingLedgerEvent

	PositiveTotal float64
	NegativeTotal float64

	TeamPositive map[common.Team]float64
	TeamNegative map[common.Team]float64

	PlayerSwing map[uint64]float64

	ResidualApplied  bool
	ExitFragsIgnored int
	DebugWarnings    []string
}

// NewRoundSwingLedger creates a fresh ledger for a round.
func NewRoundSwingLedger(roundNumber int) *RoundSwingLedger {
	return &RoundSwingLedger{
		RoundNumber:  roundNumber,
		Events:       make([]SwingLedgerEvent, 0),
		TeamPositive: make(map[common.Team]float64),
		TeamNegative: make(map[common.Team]float64),
		PlayerSwing:  make(map[uint64]float64),
	}
}

// RecordEvent appends an event and applies all allocations to player totals.
func (l *RoundSwingLedger) RecordEvent(event SwingLedgerEvent) {
	l.Events = append(l.Events, event)

	for _, alloc := range event.PositiveAlloc {
		l.applyAllocation(alloc, true)
	}
	for _, alloc := range event.NegativeAlloc {
		l.applyAllocation(alloc, false)
	}

	if event.IsExitFrag {
		l.ExitFragsIgnored++
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
		l.PositiveTotal += alloc.Amount
		l.TeamPositive[alloc.Side] += alloc.Amount
	} else {
		l.NegativeTotal += alloc.Amount
		l.TeamNegative[alloc.Side] += alloc.Amount
	}
}

// PlayerTotals returns a copy of per-player swing totals.
func (l *RoundSwingLedger) PlayerTotals() map[uint64]float64 {
	out := make(map[uint64]float64, len(l.PlayerSwing))
	for id, v := range l.PlayerSwing {
		out[id] = v
	}
	return out
}

// PlayerPositiveSwing returns positive swing credited to a player so far.
func (l *RoundSwingLedger) PlayerPositiveSwing(steamID uint64) float64 {
	total := 0.0
	for _, event := range l.Events {
		for _, alloc := range event.PositiveAlloc {
			if alloc.SteamID == steamID && alloc.Amount > 0 {
				total += alloc.Amount
			}
		}
	}
	return total
}

// PlayerNegativeSwing returns absolute negative swing debited from a player so far.
func (l *RoundSwingLedger) PlayerNegativeSwing(steamID uint64) float64 {
	total := 0.0
	for _, event := range l.Events {
		for _, alloc := range event.NegativeAlloc {
			if alloc.SteamID == steamID && alloc.Amount < 0 {
				total += -alloc.Amount
			}
		}
	}
	return total
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
