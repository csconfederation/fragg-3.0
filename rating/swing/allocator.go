package swing

import (
	"math"

	"github.com/ethsmith/eco-rating/rating/probability"

	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
)

const (
	baseVictimWeight        = 1.0
	damageHelperPoolWeight  = 0.75
	flashHelperPoolWeight   = 0.25
	aliveAtObjectivePenalty = 0.25
	tradeDeathRefundShare   = 0.30
)

// PlayerRoundContext holds per-player round data used for allocation scoring.
type PlayerRoundContext struct {
	SteamID       uint64
	Side          common.Team
	Alive         bool
	Damage        int
	Kills         int
	FlashAssists  int
	PositiveSwing float64
	NegativeSwing float64
	PlantedBomb   bool
	DefusedBomb   bool
}

// KillAllocationInput holds context needed for zero-sum kill allocation.
type KillAllocationInput struct {
	Kill                *KillEvent
	VictimPriorDamage   int
	LosingSideTeammates []uint64 // alive teammates on victim side excluding victim
	ExitFrag            bool
	ExitFragScale       float64 // 1.0 = full, 0.05 = tiny mode
}

// PoolAllocator distributes probability delta pools among players.
type PoolAllocator struct {
	engine *probability.Engine
	cfg    Config
}

// NewPoolAllocator creates an allocator with the given probability engine and config.
func NewPoolAllocator(engine *probability.Engine, cfg Config) *PoolAllocator {
	return &PoolAllocator{engine: engine, cfg: cfg}
}

// AllocateKillEvent computes zero-sum positive/negative pools for a kill.
func (a *PoolAllocator) AllocateKillEvent(
	state *probability.RoundState,
	input KillAllocationInput,
) (SwingLedgerEvent, KillSwingResult) {
	kill := input.Kill
	event := SwingLedgerEvent{
		Type:           SwingEventKill,
		BenefitingSide: kill.KillerSide,
		HurtSide:       kill.VictimSide,
	}

	if input.ExitFrag {
		event.IsExitFrag = true
		event.Type = SwingEventExitFrag
		positive := []PlayerSwingAllocation{{
			SteamID: kill.KillerID,
			Side:    kill.KillerSide,
			Amount:  0,
			Reason:  SwingReasonExitFragIgnored,
		}}
		event.PositiveAlloc = positive
		event.NegativeAlloc = []PlayerSwingAllocation{{
			SteamID: kill.VictimID,
			Side:    kill.VictimSide,
			Amount:  0,
			Reason:  SwingReasonExitFragIgnored,
		}}
		return event, KillSwingResult{}
	}

	stateBefore := state.Clone()
	probBefore := a.engine.GetWinProbability(stateBefore, kill.KillerSide)

	stateAfter := stateBefore.Clone()
	stateAfter.RecordDeath(kill.VictimSide)
	probAfter := a.engine.GetWinProbability(stateAfter, kill.KillerSide)

	rawDelta := probAfter - probBefore
	if rawDelta < 0 {
		rawDelta = 0
	}
	if input.ExitFragScale > 0 && input.ExitFragScale < 1 {
		rawDelta *= input.ExitFragScale
	}

	event.RawDelta = rawDelta
	pool := math.Abs(rawDelta)
	event.PositivePool = pool
	event.NegativePool = pool

	if pool <= 0 {
		return event, KillSwingResult{}
	}

	duelWinRate := a.engine.GetDuelWinRate(kill.KillerEquip, kill.VictimEquip)
	killerEcoWeight := a.killerEconomyWeight(duelWinRate)
	victimEcoWeight := a.victimEconomyWeight(duelWinRate)

	positiveAllocs := a.allocateKillPositivePool(kill, pool, killerEcoWeight)
	negativeAllocs := a.allocateKillNegativePool(kill, pool, victimEcoWeight, input, positiveAllocs)

	event.PositiveAlloc = positiveAllocs
	event.NegativeAlloc = negativeAllocs

	result := a.buildKillSwingResult(rawDelta, killerEcoWeight)
	return event, result
}

func (a *PoolAllocator) allocateKillPositivePool(
	kill *KillEvent,
	pool float64,
	killerEcoWeight float64,
) []PlayerSwingAllocation {
	killerCredit := a.calculateKillerCredit(kill)
	if kill.IsTradeKill {
		killerCredit *= a.cfg.TradeKillMultiplier
	}
	if killerCredit > 1 {
		killerCredit = 1
	}
	if killerCredit < 0 {
		killerCredit = 0
	}

	weights := map[uint64]float64{
		kill.KillerID: killerCredit * killerEcoWeight,
	}

	shareable := 1.0 - killerCredit
	if kill.TotalDamageToVictim > 0 {
		for _, contributor := range kill.DamageContributors {
			if contributor.PlayerID == kill.KillerID {
				continue
			}
			share := float64(contributor.Damage) / float64(kill.TotalDamageToVictim)
			w := shareable * damageHelperPoolWeight * share
			if w > 0 {
				weights[contributor.PlayerID] = w
			}
		}
	}

	for _, flash := range kill.FlashAssists {
		flashCredit := math.Min(flash.Duration/3.0, 1.0)
		w := shareable * flashHelperPoolWeight * flashCredit
		if w > 0 {
			weights[flash.PlayerID] += w
		}
	}

	totalWeight := 0.0
	for _, w := range weights {
		totalWeight += w
	}
	if totalWeight <= 0 {
		return nil
	}

	allocs := make([]PlayerSwingAllocation, 0, len(weights))
	for steamID, w := range weights {
		amount := pool * (w / totalWeight)
		if amount <= 0 {
			continue
		}
		reason := SwingReasonKillFinalHit
		if steamID != kill.KillerID {
			reason = SwingReasonKillDamageShare
		}
		allocs = append(allocs, PlayerSwingAllocation{
			SteamID: steamID,
			Side:    kill.KillerSide,
			Amount:  amount,
			Reason:  reason,
		})
	}
	return allocs
}

func (a *PoolAllocator) allocateKillNegativePool(
	kill *KillEvent,
	pool float64,
	victimEcoWeight float64,
	input KillAllocationInput,
	positiveAllocs []PlayerSwingAllocation,
) []PlayerSwingAllocation {
	victimWeight := baseVictimWeight * victimEcoWeight

	healthFactor := 1.0
	victimHP := 100 - input.VictimPriorDamage
	if victimHP < 1 {
		victimHP = 1
	}
	if victimHP < 100 {
		healthFactor = 0.50 + 0.50*(float64(victimHP)/100.0)
	}

	weights := map[uint64]float64{
		kill.VictimID: victimWeight * healthFactor,
	}
	for _, teammateID := range input.LosingSideTeammates {
		weights[teammateID] = 0.25
	}

	totalWeight := 0.0
	for _, w := range weights {
		totalWeight += w
	}
	if totalWeight <= 0 {
		totalWeight = 1
		weights[kill.VictimID] = 1
	}

	allocs := make([]PlayerSwingAllocation, 0, len(weights))
	for steamID, w := range weights {
		amount := -pool * (w / totalWeight)
		if amount == 0 {
			continue
		}
		side := kill.VictimSide
		reason := SwingReasonVictimDeath
		if steamID != kill.VictimID {
			reason = SwingReasonTradeRefund
		}
		allocs = append(allocs, PlayerSwingAllocation{
			SteamID: steamID,
			Side:    side,
			Amount:  amount,
			Reason:  reason,
		})
	}

	// If health factor reduced victim share, the redistribution is handled by teammate weights
	_ = positiveAllocs
	return allocs
}

func (a *PoolAllocator) calculateKillerCredit(kill *KillEvent) float64 {
	credit := a.cfg.KillFinalHitBase
	if kill.TotalDamageToVictim > 0 && kill.KillerDamageDealt > 0 {
		damageRatio := float64(kill.KillerDamageDealt) / float64(kill.TotalDamageToVictim)
		credit += a.cfg.KillDamageShareWeight * damageRatio
	}
	return credit
}

func (a *PoolAllocator) killerEconomyWeight(duelWinRate float64) float64 {
	if duelWinRate <= 0.01 {
		return 2.0
	}
	return 0.50 / duelWinRate
}

func (a *PoolAllocator) victimEconomyWeight(duelWinRate float64) float64 {
	victimWinRate := 1.0 - duelWinRate
	if victimWinRate < 0.01 {
		victimWinRate = 0.01
	}
	weight := victimWinRate / 0.50
	if weight > 2.0 {
		weight = 2.0
	}
	return weight
}

func (a *PoolAllocator) buildKillSwingResult(rawSwing, killerEcoWeight float64) KillSwingResult {
	return KillSwingResult{
		RawSwing:      rawSwing,
		EcoMultiplier: killerEcoWeight,
	}
}

// AllocateTradeReallocation redistributes victim death penalty when a trade occurs.
func (a *PoolAllocator) AllocateTradeReallocation(
	victimID uint64,
	victimSide common.Team,
	originalVictimPenalty float64,
	teammateIDs []uint64,
) []PlayerSwingAllocation {
	if originalVictimPenalty >= 0 {
		return nil
	}

	refund := -originalVictimPenalty * tradeDeathRefundShare
	if refund <= 0 {
		return nil
	}

	allocs := make([]PlayerSwingAllocation, 0)
	allocs = append(allocs, PlayerSwingAllocation{
		SteamID: victimID,
		Side:    victimSide,
		Amount:  refund,
		Reason:  SwingReasonTradeRefund,
	})

	others := make([]uint64, 0, len(teammateIDs))
	for _, id := range teammateIDs {
		if id != victimID {
			others = append(others, id)
		}
	}
	if len(others) == 0 {
		return allocs
	}

	perTeammate := refund / float64(len(others))
	for _, id := range others {
		allocs = append(allocs, PlayerSwingAllocation{
			SteamID: id,
			Side:    victimSide,
			Amount:  -perTeammate,
			Reason:  SwingReasonTradeRefund,
		})
	}
	return allocs
}

// ObjectiveAllocationInput holds context for bomb plant/defuse sharing.
type ObjectiveAllocationInput struct {
	PrimaryID     uint64
	PrimarySide   common.Team
	OpposingSide  common.Team
	TimeInRound   float64
	Players       map[uint64]PlayerRoundContext
	PrimaryShare  float64
	PrimaryReason SwingReason
	SupportReason SwingReason
}

// AllocateObjectiveEvent distributes plant or defuse swing across both teams.
func (a *PoolAllocator) AllocateObjectiveEvent(
	state *probability.RoundState,
	primarySide common.Team,
	calcDelta func(*probability.RoundState) float64,
	applyState func(*probability.RoundState),
	input ObjectiveAllocationInput,
	eventType SwingEventType,
) (SwingLedgerEvent, map[uint64]float64) {
	event := SwingLedgerEvent{
		Type:           eventType,
		BenefitingSide: input.PrimarySide,
		HurtSide:       input.OpposingSide,
	}

	stateBefore := state.Clone()
	delta := calcDelta(stateBefore)
	if delta <= 0 {
		return event, nil
	}

	event.RawDelta = delta
	pool := math.Abs(delta)
	event.PositivePool = pool
	event.NegativePool = pool

	event.PositiveAlloc = a.allocateObjectivePositive(input, pool)
	event.NegativeAlloc = a.allocateObjectiveNegative(input, pool)

	playerAmounts := make(map[uint64]float64)
	for _, alloc := range event.PositiveAlloc {
		playerAmounts[alloc.SteamID] += alloc.Amount
	}
	for _, alloc := range event.NegativeAlloc {
		playerAmounts[alloc.SteamID] += alloc.Amount
	}

	applyState(state)
	return event, playerAmounts
}

func (a *PoolAllocator) allocateObjectivePositive(input ObjectiveAllocationInput, pool float64) []PlayerSwingAllocation {
	primaryAmount := pool * input.PrimaryShare
	supportPool := pool - primaryAmount

	allocs := []PlayerSwingAllocation{{
		SteamID: input.PrimaryID,
		Side:    input.PrimarySide,
		Amount:  primaryAmount,
		Reason:  input.PrimaryReason,
	}}

	supportWeights := make(map[uint64]float64)
	for steamID, ctx := range input.Players {
		if ctx.Side != input.PrimarySide || steamID == input.PrimaryID {
			continue
		}
		weight := float64(ctx.Kills)*1.0 + float64(ctx.Damage)*0.001 + float64(ctx.FlashAssists)*0.5
		if ctx.Alive {
			weight += 0.25
		}
		if weight > 0 {
			supportWeights[steamID] = weight
		}
	}

	totalSupport := 0.0
	for _, w := range supportWeights {
		totalSupport += w
	}

	if totalSupport <= 0 {
		allocs[0].Amount = pool
		return allocs
	}

	for steamID, w := range supportWeights {
		allocs = append(allocs, PlayerSwingAllocation{
			SteamID: steamID,
			Side:    input.PrimarySide,
			Amount:  supportPool * (w / totalSupport),
			Reason:  input.SupportReason,
		})
	}
	return allocs
}

func (a *PoolAllocator) allocateObjectiveNegative(input ObjectiveAllocationInput, pool float64) []PlayerSwingAllocation {
	weights := make(map[uint64]float64)
	for steamID, ctx := range input.Players {
		if ctx.Side != input.OpposingSide {
			continue
		}
		weight := ctx.NegativeSwing
		if ctx.Alive {
			weight += aliveAtObjectivePenalty
		}
		if weight <= 0 {
			weight = 1.0
		}
		weights[steamID] = weight
	}

	total := 0.0
	for _, w := range weights {
		total += w
	}
	if total <= 0 {
		return nil
	}

	allocs := make([]PlayerSwingAllocation, 0, len(weights))
	for steamID, w := range weights {
		allocs = append(allocs, PlayerSwingAllocation{
			SteamID: steamID,
			Side:    input.OpposingSide,
			Amount:  -pool * (w / total),
			Reason:  input.PrimaryReason,
		})
	}
	return allocs
}
