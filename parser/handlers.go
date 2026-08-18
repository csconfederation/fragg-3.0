// =============================================================================
// DISCLAIMER: Comments in this file were generated with AI assistance to help
// users find and understand code for reference while building FraGG 3.0.
// =============================================================================

// Package parser provides CS2 demo file parsing functionality.
// This file contains event handlers that process game events (kills, deaths,
// bomb plants, round ends, etc.) and update player statistics accordingly.
package parser

import (
	"github.com/csconfederation/fragg-3.0/model"
	"github.com/csconfederation/fragg-3.0/rating"
	"github.com/csconfederation/fragg-3.0/rating/swing"
	"math"

	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/msg"
)

// registerHandlers sets up all event handlers for demo parsing.
// This is the core of the parsing logic, delegating to focused handler methods.
func (d *DemoParser) registerHandlers() {
	d.registerMapHandler()
	if !d.externalLifecycle {
		d.registerMatchHandlers()
		d.registerRoundLifecycleHandlers()
		d.registerRoundEndHandler()
	}
	d.registerBombHandlers()
	d.registerFlashHandlers()
	d.registerKillHandler()
	d.registerDamageHandler()
	d.registerRoundDecisionHandlers()
}

// applyLedgerAllocations applies swing ledger allocations to round stats.
func (d *DemoParser) applyLedgerAllocations(allocs []swing.PlayerSwingAllocation, timeInRound float64, opponent string) {
	for _, alloc := range allocs {
		if alloc.Amount == 0 {
			continue
		}
		roundStats, ok := d.state.Round[alloc.SteamID]
		if !ok {
			continue
		}
		roundStats.ProbabilitySwing += alloc.Amount
		roundStats.AddSwingContribution(model.SwingContribution{
			Type:        string(alloc.Reason),
			Amount:      alloc.Amount,
			TimeInRound: timeInRound,
			Opponent:    opponent,
		})
	}
}

func (d *DemoParser) buildPlayerRoundContext(gs demoinfocs.GameState) map[uint64]swing.PlayerRoundContext {
	alive := make(map[uint64]bool)
	for _, p := range gs.Participants().Playing() {
		if p != nil && !p.IsBot {
			alive[p.SteamID64] = p.IsAlive()
		}
	}

	players := make(map[uint64]swing.PlayerRoundContext)
	for steamID, roundStats := range d.state.Round {
		side := common.TeamUnassigned
		if roundStats.PlayerSide == "T" {
			side = common.TeamTerrorists
		} else if roundStats.PlayerSide == "CT" {
			side = common.TeamCounterTerrorists
		}

		positive := 0.0
		negative := 0.0
		if roundStats.ProbabilitySwing > 0 {
			positive = roundStats.ProbabilitySwing
		} else if roundStats.ProbabilitySwing < 0 {
			negative = -roundStats.ProbabilitySwing
		}

		players[steamID] = swing.PlayerRoundContext{
			SteamID:       steamID,
			Side:          side,
			Alive:         alive[steamID],
			Damage:        roundStats.Damage,
			Kills:         roundStats.Kills,
			FlashAssists:  roundStats.FlashAssists,
			PositiveSwing: positive,
			NegativeSwing: negative,
			PlantedBomb:   roundStats.PlantedBomb,
			DefusedBomb:   roundStats.DefusedBomb,
		}
	}
	return players
}

func (d *DemoParser) getLosingSideTeammates(victim *common.Player, gs demoinfocs.GameState) []uint64 {
	if victim == nil {
		return nil
	}
	teammates := make([]uint64, 0)
	for _, p := range gs.Participants().Playing() {
		if p == nil || p.IsBot || !p.IsAlive() {
			continue
		}
		if p.Team == victim.Team && p.SteamID64 != victim.SteamID64 {
			teammates = append(teammates, p.SteamID64)
		}
	}
	return teammates
}

func (d *DemoParser) getTeammateIDs(side common.Team, excludeID uint64, gs demoinfocs.GameState) []uint64 {
	ids := make([]uint64, 0)
	for _, p := range gs.Participants().Playing() {
		if p == nil || p.IsBot {
			continue
		}
		if p.Team == side && p.SteamID64 != excludeID {
			ids = append(ids, p.SteamID64)
		}
	}
	return ids
}

// registerMapHandler sets up the map name extraction from server info.
func (d *DemoParser) registerMapHandler() {
	d.parser.RegisterNetMessageHandler(func(m *msg.CSVCMsg_ServerInfo) {
		d.state.MapName = m.GetMapName()
	})
}

// registerMatchHandlers sets up match start/end detection.
func (d *DemoParser) registerMatchHandlers() {
	d.parser.RegisterEventHandler(func(e events.MatchStart) {
		d.state.MatchStarted = true
	})

	d.parser.RegisterEventHandler(func(e events.MatchStartedChanged) {
		if e.NewIsStarted {
			d.state.MatchStarted = true
		}
	})
}

// registerRoundLifecycleHandlers sets up round start and freeze time end handlers.
func (d *DemoParser) registerRoundLifecycleHandlers() {
	d.parser.RegisterEventHandler(func(e events.RoundStart) {
		d.handleRoundStart()
	})

	d.parser.RegisterEventHandler(func(e events.RoundFreezetimeEnd) {
		d.handleFreezetimeEnd()
	})
}

// handleRoundStart resets round state for a new round.
func (d *DemoParser) handleRoundStart() {
	d.resetRoundState()
	if d.collector != nil {
		d.collector.RecordRoundStart(0, 0, false, "")
	}
}

// registerBombHandlers sets up bomb plant, defuse, and explode handlers.
func (d *DemoParser) registerBombHandlers() {
	d.parser.RegisterEventHandler(func(e events.BombPlanted) {
		d.handleBombPlanted(e)
	})

	d.parser.RegisterEventHandler(func(e events.BombDefused) {
		d.handleBombDefused(e)
	})

	d.parser.RegisterEventHandler(func(e events.BombExplode) {
		d.handleBombExplode()
	})
}

// handleBombPlanted processes a bomb plant event.
func (d *DemoParser) handleBombPlanted(e events.BombPlanted) {
	if d.state.ShouldSkipEvent() {
		return
	}

	// Record state snapshot BEFORE bomb plant
	if d.collector != nil {
		gs := d.parser.GameState()
		tAlive, ctAlive := d.state.CountAlivePlayers(gs.Participants().Playing())
		d.collector.RecordStateSnapshot(tAlive, ctAlive, false) // bomb not planted yet
	}

	d.state.BombPlanted = true

	planter := d.state.ensurePlayer(e.Player)
	roundStats := d.state.ensureRound(e.Player)
	roundStats.PlantedBomb = true

	// Track bomb plant swing
	if d.state.SwingTracker != nil {
		timeInRound := d.timeInRound()
		gs := d.parser.GameState()
		players := d.buildPlayerRoundContext(gs)
		plantAllocs := d.state.SwingTracker.RecordBombPlant(e.Player.SteamID64, timeInRound, players, gs.IngameTick())
		d.applyLedgerAllocations(plantAllocs, timeInRound, "")
	}

	d.logger.LogBombPlant(d.state.RoundNumber, planter.Name)
}

// handleBombDefused processes a bomb defuse event.
func (d *DemoParser) handleBombDefused(e events.BombDefused) {
	if d.state.ShouldSkipEvent() {
		return
	}

	// Record state snapshot before defuse
	if d.collector != nil {
		gs := d.parser.GameState()
		tAlive, ctAlive := d.state.CountAlivePlayers(gs.Participants().Playing())
		d.collector.RecordStateSnapshot(tAlive, ctAlive, true) // bomb is planted
	}

	defuser := d.state.ensurePlayer(e.Player)
	roundStats := d.state.ensureRound(e.Player)
	roundStats.DefusedBomb = true

	timeInRound := d.timeInRound()

	// Track bomb defuse swing
	if d.state.SwingTracker != nil {
		gs := d.parser.GameState()
		players := d.buildPlayerRoundContext(gs)
		defuseAllocs := d.state.SwingTracker.RecordBombDefuse(e.Player.SteamID64, timeInRound, players, gs.IngameTick())
		d.applyLedgerAllocations(defuseAllocs, timeInRound, "")
	}

	d.logger.LogBombDefuse(d.state.RoundNumber, defuser.Name)

	// Mark round as decided - kills after defuse are exit frags
	d.state.RoundDecided = true
	d.state.RoundDecidedAt = timeInRound
}

// handleBombExplode marks the round as decided when the bomb explodes.
func (d *DemoParser) handleBombExplode() {
	if d.state.ShouldSkipEvent() {
		return
	}
	timeInRound := d.timeInRound()
	d.state.RoundDecided = true
	d.state.RoundDecidedAt = timeInRound

	// Record state snapshot at bomb explosion (e.g. 0v3_planted or 2v1_planted)
	if d.collector != nil {
		gs := d.parser.GameState()
		tAlive, ctAlive := d.state.CountAlivePlayers(gs.Participants().Playing())
		d.collector.RecordStateSnapshot(tAlive, ctAlive, true) // bomb is planted
	}
}

// registerFlashHandlers sets up flash and grenade throw handlers.
func (d *DemoParser) registerFlashHandlers() {
	d.parser.RegisterEventHandler(func(e events.PlayerFlashed) {
		d.handlePlayerFlashed(e)
	})

	d.parser.RegisterEventHandler(func(e events.GrenadeProjectileThrow) {
		d.handleGrenadeThrow(e)
	})
}

// handlePlayerFlashed processes a player flash event.
func (d *DemoParser) handlePlayerFlashed(e events.PlayerFlashed) {
	if d.state.ShouldSkipEvent() {
		return
	}

	if e.Attacker != nil && e.Player != nil {
		roundStats := d.state.ensureRound(e.Attacker)
		flashDuration := e.FlashDuration().Seconds()
		if e.Attacker.Team != e.Player.Team {
			roundStats.FlashAssists++
			roundStats.EnemyFlashDuration += flashDuration
			roundStats.EnemiesFlashed++

			// Track flash for swing attribution
			if d.state.SwingTracker != nil {
				d.state.SwingTracker.RecordFlash(e.Attacker.SteamID64, e.Player.SteamID64, flashDuration, d.parser.GameState().IngameTick())
			}
		} else if e.Attacker.SteamID64 != e.Player.SteamID64 {
			roundStats.TeamFlashCount++
			roundStats.TeamFlashDuration += flashDuration
		}
	}
}

// handleGrenadeThrow tracks all grenade throws.
func (d *DemoParser) handleGrenadeThrow(e events.GrenadeProjectileThrow) {
	if d.state.ShouldSkipEvent() {
		return
	}

	if e.Projectile != nil && e.Projectile.Thrower != nil && e.Projectile.WeaponInstance != nil {
		roundStats := d.state.ensureRound(e.Projectile.Thrower)

		switch e.Projectile.WeaponInstance.Type {
		case common.EqFlash:
			roundStats.FlashesThrown++
		case common.EqSmoke:
			roundStats.SmokesThrown++
		case common.EqHE:
			roundStats.HEsThrown++
		case common.EqMolotov, common.EqIncendiary:
			roundStats.MolotovsThrown++
		}
	}
}

// handleFreezetimeEnd processes the end of freeze time, detecting knife rounds
// and initializing round state for all participants.
func (d *DemoParser) handleFreezetimeEnd() {
	gs := d.parser.GameState()
	if gs.IsWarmupPeriod() {
		return
	}
	participants := gs.Participants().Playing()
	if len(participants) > 0 {
		firstPlayer := participants[0]
		if firstPlayer.Money()+firstPlayer.MoneySpentThisRound() == 0 {
			d.state.IsKnifeRound = true
			d.logger.LogKnifeRound()
			return
		}
	}
	d.BeginRound()
}

// registerKillHandler sets up the main kill event handler.
func (d *DemoParser) registerKillHandler() {
	d.parser.RegisterEventHandler(func(e events.Kill) {
		d.handleKill(e)
	})
}

// killContext holds all context needed for processing a kill event.
type killContext struct {
	event         events.Kill
	attacker      *common.Player
	victim        *common.Player
	currentTick   int
	timeInRound   float64
	killValue     float64
	deathPenalty  float64
	attackerEquip int
	victimEquip   int
	isTradeKill   bool
	tradeSpeed    float64
}

// handleKill processes a kill event, updating statistics for killer and victim.
func (d *DemoParser) handleKill(e events.Kill) {
	if d.parser.GameState().IsWarmupPeriod() || d.state.ShouldSkipEvent() {
		return
	}

	if d.shouldSkipKill(e) {
		return
	}

	ctx := d.buildKillContext(e)

	d.processVictimDeath(ctx)
	d.processTradeDetection(ctx)

	if ctx.attacker == nil || ctx.victim == nil {
		return
	}

	d.state.TradeDetector.RecordKill(ctx.attacker, ctx.victim, ctx.currentTick)
	d.recordKillForProbability(ctx)
	d.processKillerStats(ctx)
	d.processWeaponStats(ctx)
	d.processOpeningKill(ctx)
	d.processSwingTracking(ctx)
	d.processEcoKillFlags(ctx)
	d.processAssist(ctx)
}

// shouldSkipKill returns true if the kill event should be ignored.
func (d *DemoParser) shouldSkipKill(e events.Kill) bool {
	a, v := e.Killer, e.Victim
	if a != nil && v != nil && a.SteamID64 == v.SteamID64 {
		return true
	}
	if a != nil && v != nil && a.Team == v.Team {
		return true
	}
	return false
}

// buildKillContext creates the context struct for a kill event.
func (d *DemoParser) buildKillContext(e events.Kill) *killContext {
	currentTick := d.parser.CurrentFrame()
	currentTime := float64(currentTick) / float64(rating.TickRate)
	timeInRound := currentTime - d.state.RoundStartTime

	ctx := &killContext{
		event:       e,
		attacker:    e.Killer,
		victim:      e.Victim,
		currentTick: currentTick,
		timeInRound: timeInRound,
	}

	if ctx.attacker != nil && ctx.victim != nil {
		ctx.attackerEquip = ctx.attacker.EquipmentValueCurrent()
		ctx.victimEquip = ctx.victim.EquipmentValueCurrent()
		ctx.killValue = rating.EcoKillValue(float64(ctx.attackerEquip), float64(ctx.victimEquip))
		ctx.deathPenalty = rating.EcoDeathPenalty(float64(ctx.victimEquip), float64(ctx.attackerEquip))
		ctx.isTradeKill, ctx.tradeSpeed = d.state.TradeDetector.CheckTradeKill(
			ctx.attacker, ctx.victim, ctx.currentTick, ctx.timeInRound)
	}

	return ctx
}

// processVictimDeath handles victim death stats and AWP loss detection.
func (d *DemoParser) processVictimDeath(ctx *killContext) {
	if ctx.victim == nil {
		return
	}

	victimRound := d.state.ensureRound(ctx.victim)
	victimRound.DeathTime = ctx.timeInRound

	// Check if this death puts a teammate into a clutch situation
	// We need to check BEFORE the victim is marked dead in the game state
	d.checkClutchEntry(ctx)

	for _, weapon := range ctx.victim.Weapons() {
		if weapon.Type == common.EqAWP {
			victimRound.HadAWP = true
			victimRound.LostAWP = true
			break
		}
	}

	gs := d.parser.GameState()
	d.state.TradeDetector.RecordDeath(ctx.victim, ctx.attacker, ctx.currentTick, ctx.timeInRound, gs.Participants().Playing())
}

// processTradeDetection checks for trades and updates trade stats.
func (d *DemoParser) processTradeDetection(ctx *killContext) {
	if ctx.attacker != nil && ctx.victim != nil {
		tradeResult := d.state.TradeDetector.CheckForTrade(
			ctx.attacker, ctx.victim, ctx.currentTick, ctx.timeInRound,
			d.state.Players, d.state.Round)
		if tradeResult.IsTrade {
			attackerRound := d.state.ensureRound(ctx.attacker)
			attackerRound.SavedTeammate = true
			attackerRound.TradeDenials++

			if tradeResult.TradeReallocation && d.state.SwingTracker != nil {
				gs := d.parser.GameState()
				teammateIDs := d.getTeammateIDs(ctx.attacker.Team, tradeResult.TradedPlayerID, gs)
				allocs := d.state.SwingTracker.ApplyTradeReallocation(
					tradeResult.TradedPlayerID,
					ctx.attacker.Team,
					tradeResult.OriginalDeathPenalty,
					teammateIDs,
				)
				d.applyLedgerAllocations(allocs, ctx.timeInRound, ctx.victim.Name)
			}

			d.logger.LogTrade(d.state.RoundNumber, ctx.attacker.Name, tradeResult.TradedPlayerName, ctx.victim.Name)
		}
	}

	d.state.TradeDetector.ProcessExpiredTrades(ctx.currentTick, d.state.Round)
}

// recordKillForProbability records the kill for probability data collection.
func (d *DemoParser) recordKillForProbability(ctx *killContext) {
	if d.collector == nil {
		return
	}

	// Record state snapshot BEFORE the kill.
	// The demoinfocs parser has already marked the victim as dead by the time
	// the Kill event fires, so we add 1 back to the victim's side to reconstruct
	// the pre-kill state.
	gs := d.parser.GameState()
	tAlive, ctAlive := d.state.CountAlivePlayers(gs.Participants().Playing())
	if ctx.victim != nil {
		if ctx.victim.Team == common.TeamTerrorists {
			tAlive++
		} else if ctx.victim.Team == common.TeamCounterTerrorists {
			ctAlive++
		}
	}
	// Cap at 5 after reconstructing pre-kill state (CountAlivePlayers caps at 5,
	// but adding 1 back could exceed that if a 6th player was present)
	if tAlive > 5 {
		tAlive = 5
	}
	if ctAlive > 5 {
		ctAlive = 5
	}
	d.collector.RecordStateSnapshot(tAlive, ctAlive, d.state.BombPlanted)
	d.collector.RecordKill(float64(ctx.attackerEquip), float64(ctx.victimEquip))
}

// processKillerStats updates killer statistics.
func (d *DemoParser) processKillerStats(ctx *killContext) {
	round := d.state.ensureRound(ctx.attacker)
	victimRound := d.state.ensureRound(ctx.victim)

	d.logger.LogKill(d.state.RoundNumber, ctx.attacker.Name, ctx.victim.Name, ctx.attackerEquip, ctx.victimEquip, ctx.killValue)
	d.logger.LogDeath(d.state.RoundNumber, ctx.victim.Name, ctx.attacker.Name, ctx.victimEquip, ctx.attackerEquip, ctx.deathPenalty)

	round.KillTimes = append(round.KillTimes, ctx.timeInRound)

	if d.state.RoundDecided {
		round.IsExitFrag = true
		round.ExitFrags++
	}

	round.Kills++
	round.GotKill = true
	round.EconImpact += ctx.killValue
	if ctx.event.IsHeadshot {
		round.Headshots++
	}

	if d.state.SwingTracker != nil {
		ttk := d.state.SwingTracker.GetTimeToKill(ctx.attacker.SteamID64, ctx.victim.SteamID64, ctx.timeInRound)
		if ttk >= 0 {
			round.TimeToKillSum += ttk
			round.KillsWithTTK++
		}
	}

	if ctx.killValue < 1.0 {
		round.LowBuyKills++
	}
	if ctx.killValue <= 0.85 {
		round.DisadvantagedBuyKills++
	}

	gs := d.parser.GameState()
	tAlive, ctAlive := d.state.CountAlivePlayers(gs.Participants().Playing())

	var attackerAliveAfter, victimAliveAfter int
	var attackerAliveBefore, victimAliveBefore int
	if ctx.attacker.Team == common.TeamTerrorists {
		attackerAliveAfter = tAlive
		victimAliveAfter = ctAlive
		attackerAliveBefore = tAlive
		victimAliveBefore = ctAlive + 1
	} else {
		attackerAliveAfter = ctAlive
		victimAliveAfter = tAlive
		attackerAliveBefore = ctAlive
		victimAliveBefore = tAlive + 1
	}

	if attackerAliveBefore <= victimAliveBefore && attackerAliveAfter > victimAliveAfter {
		round.ManAdvantageKills++
	}
	if victimAliveBefore >= attackerAliveBefore && victimAliveAfter < attackerAliveAfter {
		victimRound.ManDisadvantageDeaths++
	}

	victimRound.EcoDeathValue += ctx.deathPenalty
}

// processWeaponStats updates weapon-specific statistics.
func (d *DemoParser) processWeaponStats(ctx *killContext) {
	if ctx.event.Weapon == nil {
		return
	}

	round := d.state.ensureRound(ctx.attacker)

	switch ctx.event.Weapon.Type {
	case common.EqAWP:
		round.AWPKills++
		round.AWPKill = true
	case common.EqKnife:
		round.KnifeKill = true
	case common.EqHE, common.EqMolotov, common.EqIncendiary:
		round.UtilityKills++
	}

	isPistol := ctx.event.Weapon.Type >= common.EqP2000 && ctx.event.Weapon.Type <= common.EqRevolver
	victimHadRifle := ctx.victimEquip > 3500
	if isPistol && victimHadRifle {
		round.PistolVsRifleKill = true
	}
}

// processOpeningKill handles first kill of the round stats.
func (d *DemoParser) processOpeningKill(ctx *killContext) {
	if d.state.RoundHasKill {
		return
	}

	round := d.state.ensureRound(ctx.attacker)
	victimRound := d.state.ensureRound(ctx.victim)

	round.OpeningKill = true
	round.EntryFragger = true
	round.InvolvedInOpening = true

	if ctx.event.Weapon != nil && ctx.event.Weapon.Type == common.EqAWP {
		round.AWPOpeningKill = true
	}

	victimRound.OpeningDeath = true
	victimRound.InvolvedInOpening = true

	d.state.RoundHasKill = true
	d.logger.LogOpeningKill(d.state.RoundNumber, ctx.attacker.Name, ctx.victim.Name)
}

// processSwingTracking handles probability-based swing calculation.
func (d *DemoParser) processSwingTracking(ctx *killContext) {
	round := d.state.ensureRound(ctx.attacker)

	if ctx.isTradeKill {
		round.TradeKill = true
		round.TradeSpeed = ctx.tradeSpeed
	}

	if d.state.SwingTracker == nil {
		return
	}

	gs := d.parser.GameState()
	losingSideTeammates := d.getLosingSideTeammates(ctx.victim, gs)

	killResult := d.state.SwingTracker.RecordKill(
		ctx.attacker.SteamID64, ctx.victim.SteamID64,
		ctx.attacker.Team, ctx.victim.Team,
		float64(ctx.attackerEquip), float64(ctx.victimEquip),
		ctx.timeInRound,
		ctx.isTradeKill, ctx.event.IsHeadshot,
		d.state.RoundDecided,
		losingSideTeammates,
		ctx.currentTick,
	)

	swingResult := killResult.Swing
	d.applyLedgerAllocations(killResult.Allocations, ctx.timeInRound, ctx.victim.Name)

	for _, alloc := range killResult.Allocations {
		if alloc.SteamID == ctx.victim.SteamID64 && alloc.Reason == swing.SwingReasonVictimDeath {
			victimRound := d.state.ensureRound(ctx.victim)
			victimRound.LastDeathSwing = alloc.Amount
		}
	}

	if killResult.SurvivalCreditPerPlayer > 0 {
		survivalAllocs := make([]swing.PlayerSwingAllocation, 0, len(killResult.SurvivalBeneficiaries))
		for _, beneficiaryID := range killResult.SurvivalBeneficiaries {
			survivalAllocs = append(survivalAllocs, swing.PlayerSwingAllocation{
				SteamID: beneficiaryID,
				Side:    ctx.attacker.Team,
				Amount:  killResult.SurvivalCreditPerPlayer,
				Reason:  swing.SwingReasonSurvivalCredit,
			})
		}
		d.applyLedgerAllocations(survivalAllocs, ctx.timeInRound, ctx.victim.Name)
	}

	if swingResult.EcoMultiplier > 0 {
		round.EcoAdjustedKills += swingResult.EcoMultiplier
	}
}

// processEcoKillFlags sets eco kill and anti-eco flags.
func (d *DemoParser) processEcoKillFlags(ctx *killContext) {
	round := d.state.ensureRound(ctx.attacker)
	victimRound := d.state.ensureRound(ctx.victim)

	equipRatio := float64(ctx.victimEquip) / math.Max(float64(ctx.attackerEquip), 500.0)
	if equipRatio > 2.0 {
		round.EcoKill = true
	}
	if equipRatio < 0.5 {
		victimRound.AntiEcoKill = true
	}
	if ctx.event.IsHeadshot {
		round.PerfectKills++
	}
}

// processAssist handles assist statistics.
func (d *DemoParser) processAssist(ctx *killContext) {
	if ctx.event.Assister == nil {
		return
	}

	assistRound := d.state.ensureRound(ctx.event.Assister)
	assistRound.GotAssist = true
	assistRound.Assists++
}

// registerDamageHandler sets up the damage event handler.
func (d *DemoParser) registerDamageHandler() {
	d.parser.RegisterEventHandler(func(e events.PlayerHurt) {
		d.handlePlayerHurt(e)
	})
}

// handlePlayerHurt processes a damage event.
func (d *DemoParser) handlePlayerHurt(e events.PlayerHurt) {
	if d.parser.GameState().IsWarmupPeriod() || d.state.ShouldSkipEvent() {
		return
	}

	if e.Attacker == nil || e.Player == nil {
		return
	}

	dmg := int(e.HealthDamageTaken)

	if e.Attacker.Team != e.Player.Team {
		roundStats := d.state.ensureRound(e.Attacker)
		roundStats.Damage += dmg

		victimRound := d.state.ensureRound(e.Player)
		victimRound.DamageTaken += dmg

		if e.Weapon != nil {
			switch e.Weapon.Type {
			case common.EqHE:
				roundStats.UtilityDamage += dmg
				roundStats.HEDamage += dmg
			case common.EqMolotov, common.EqIncendiary:
				roundStats.UtilityDamage += dmg
				roundStats.FireDamage += dmg
			}
		}

		if d.state.SwingTracker != nil {
			d.state.SwingTracker.RecordDamage(e.Attacker.SteamID64, e.Player.SteamID64, dmg, d.timeInRound())
		}
	}
}

// registerRoundDecisionHandlers sets up handlers that detect when a round is decided.
func (d *DemoParser) registerRoundDecisionHandlers() {
	// Round decided by team elimination
	d.parser.RegisterEventHandler(func(e events.Kill) {
		d.handleRoundDecisionKill()
	})
}

// handleRoundDecisionKill checks if a kill results in team elimination.
func (d *DemoParser) handleRoundDecisionKill() {
	if d.state.ShouldSkipEvent() || d.state.RoundDecided {
		return
	}

	gs := d.parser.GameState()
	tAlive, ctAlive := d.state.CountAlivePlayers(gs.Participants().Playing())

	if tAlive == 0 || ctAlive == 0 {
		d.state.RoundDecided = true
		d.state.RoundDecidedAt = d.timeInRound()
	}
}

// registerRoundEndHandler sets up the round end event handler.
func (d *DemoParser) registerRoundEndHandler() {
	d.parser.RegisterEventHandler(func(e events.RoundEnd) {
		d.handleRoundEnd(e)
	})
}

// roundEndContext holds context for round end processing.
type roundEndContext struct {
	gs            demoinfocs.GameState
	winnerTeam    common.Team
	roundDuration float64
	timeRemaining float64
}

// handleRoundEnd processes the end of a round, updating all player statistics.
func (d *DemoParser) handleRoundEnd(e events.RoundEnd) {
	if d.parser.GameState().IsWarmupPeriod() || d.state.IsKnifeRound || !d.state.RoundActive {
		return
	}

	d.FinalizeRound(int(e.Winner))
	snap := d.SnapshotRound()
	d.foldSnapshot(snap)
	d.state.RoundsCounted++
	d.updateTeamScores(e.Winner)
	d.logger.LogRoundEnd(d.state.RoundNumber)
}

// processRoundEndTrades handles pending trades at round end.
func (d *DemoParser) processRoundEndTrades() {
	currentTick := d.parser.CurrentFrame()
	d.state.TradeDetector.ProcessRoundEndTrades(currentTick, d.state.Round)
}

// processMultiKills updates multi-kill statistics.
func (d *DemoParser) processMultiKills() {
	for _, roundStats := range d.state.Round {
		if roundStats.Kills >= 2 && roundStats.Kills <= 5 {
			d.logger.LogMultiKill(d.state.RoundNumber, "", roundStats.Kills)
		}
	}
}

// processSurvivalStats updates survival and time alive statistics.
func (d *DemoParser) processSurvivalStats(ctx *roundEndContext) {
	for _, p := range ctx.gs.Participants().Playing() {
		round := d.state.ensureRound(p)

		teamWon := p.Team == ctx.winnerTeam
		round.TeamWon = teamWon

		if p.IsAlive() {
			round.Survived = true
			round.TimeAlive = ctx.roundDuration
		} else if round.DeathTime > 0 {
			round.TimeAlive = round.DeathTime
		}
	}
}

// processClutchDetection detects and records clutch situations.
// Uses ClutchEnteredSize which was set when the player entered the clutch during the round.
func (d *DemoParser) processClutchDetection(ctx *roundEndContext) {
	for _, p := range ctx.gs.Participants().Playing() {
		round := d.state.ensureRound(p)

		aliveTeammates, _ := d.countAliveByTeam(ctx.gs.Participants().Playing(), p.Team)

		if p.IsAlive() && aliveTeammates == 1 {
			round.WasLastAlive = true
		}

		if round.ClutchEnteredSize > 0 {
			d.recordClutchAttempt(round, round.ClutchEnteredSize)
		}

		if p.IsAlive() && !round.TeamWon {
			round.SavedWeapons = true
		}
	}
}

// countAliveByTeam counts alive teammates and enemies for a given team.
func (d *DemoParser) countAliveByTeam(participants []*common.Player, team common.Team) (teammates, enemies int) {
	for _, other := range participants {
		if other.IsAlive() {
			if other.Team == team {
				teammates++
			} else {
				enemies++
			}
		}
	}
	return teammates, enemies
}

// checkClutchEntry detects when a player enters a clutch situation due to a teammate dying.
// This is called when a player dies, to check if their death puts a teammate into a 1vX.
func (d *DemoParser) checkClutchEntry(ctx *killContext) {
	if ctx.victim == nil {
		return
	}

	gs := d.parser.GameState()
	participants := gs.Participants().Playing()

	// Count alive players on victim's team (excluding the victim who just died)
	// and alive enemies
	var aliveTeammates int
	var aliveEnemies int
	var lastAliveTeammate *common.Player

	for _, p := range participants {
		if p.SteamID64 == ctx.victim.SteamID64 {
			continue // Skip the victim
		}
		if !p.IsAlive() {
			continue
		}
		if p.Team == ctx.victim.Team {
			aliveTeammates++
			lastAliveTeammate = p
		} else {
			aliveEnemies++
		}
	}

	// If exactly one teammate is left alive and there are enemies, they're entering a clutch
	if aliveTeammates == 1 && aliveEnemies > 0 && lastAliveTeammate != nil {
		clutcherRound := d.state.ensureRound(lastAliveTeammate)
		// Only record if they haven't already entered a clutch this round
		// (use the highest enemy count - first entry into clutch)
		if clutcherRound.ClutchEnteredSize == 0 {
			clutcherRound.ClutchEnteredSize = aliveEnemies
		}
	}
}

// recordClutchAttempt records a clutch attempt and its outcome.
func (d *DemoParser) recordClutchAttempt(round *model.RoundStats, aliveEnemies int) {
	round.ClutchAttempt = true
	round.ClutchSize = aliveEnemies
	round.ClutchKills = round.Kills
	if round.TeamWon {
		round.ClutchWon = true
	}
}

// processRoundEndSwing applies residual swing and validates the round ledger.
func (d *DemoParser) processRoundEndSwing(ctx *roundEndContext) {
	if d.state.SwingTracker == nil {
		return
	}

	maxDamage := 0
	for _, roundStats := range d.state.Round {
		if roundStats.Damage > maxDamage {
			maxDamage = roundStats.Damage
		}
	}

	players := d.buildPlayerRoundContext(ctx.gs)
	residualAllocs := d.state.SwingTracker.ApplyRoundResidual(ctx.winnerTeam, players, maxDamage)
	d.applyLedgerAllocations(residualAllocs, ctx.roundDuration, "")

	if warnings := d.state.SwingTracker.FinalizeRound(); len(warnings) > 0 {
		for _, warning := range warnings {
			d.logger.Printf("Round %d swing warning: %s", d.state.RoundNumber, warning)
		}
	}
}

// updateTeamScores updates team scores based on round winner.
func (d *DemoParser) updateTeamScores(winnerTeam common.Team) {
	if winnerTeam == common.TeamTerrorists {
		if d.state.CurrentSide == "T" {
			d.state.TeamScore++
		} else {
			d.state.EnemyScore++
		}
	} else if winnerTeam == common.TeamCounterTerrorists {
		if d.state.CurrentSide == "CT" {
			d.state.TeamScore++
		} else {
			d.state.EnemyScore++
		}
	}
}
