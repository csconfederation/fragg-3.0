// =============================================================================
// DISCLAIMER: Comments in this file were generated with AI assistance to help
// users find and understand code for reference while building FraGG 3.0.
// =============================================================================

// Package parser provides CS2 demo file parsing functionality.
// This file contains event handlers that process game events (kills, deaths,
// bomb plants, round ends, etc.) and update player statistics accordingly.
package parser

import (
	"eco-rating/model"
	"eco-rating/rating"
	"eco-rating/rating/probability"
	"eco-rating/rating/swing"
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
	d.registerMatchHandlers()
	d.registerRoundLifecycleHandlers()
	d.registerBombHandlers()
	d.registerFlashHandlers()
	d.registerKillHandler()
	d.registerDamageHandler()
	d.registerRoundDecisionHandlers()
	d.registerRoundEndHandler()
}

// addKillSwingContribution records per-event swing contributions for killer and victim.
func (d *DemoParser) addKillSwingContribution(ctx *killContext, swingResult swing.KillSwingResult, victimContribution float64) {
	if ctx.attacker == nil || ctx.victim == nil {
		return
	}

	weaponName := ""
	if ctx.event.Weapon != nil {
		weaponName = ctx.event.Weapon.String()
	}

	attackerRound := d.state.ensureRound(ctx.attacker)
	if swingResult.KillerSwing != 0 {
		attackerRound.AddSwingContribution(model.SwingContribution{
			Type:          "kill",
			Amount:        swingResult.KillerSwing,
			TimeInRound:   ctx.timeInRound,
			Opponent:      ctx.victim.Name,
			Weapon:        weaponName,
			IsTrade:       ctx.isTradeKill,
			IsHeadshot:    ctx.event.IsHeadshot,
			EcoMultiplier: swingResult.EcoMultiplier,
		})
	}

	victimRound := d.state.ensureRound(ctx.victim)
	if victimContribution != 0 {
		victimRound.AddSwingContribution(model.SwingContribution{
			Type:        "death",
			Amount:      victimContribution,
			TimeInRound: ctx.timeInRound,
			Opponent:    ctx.attacker.Name,
			Weapon:      weaponName,
		})
	}
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
	d.state.Round = make(map[uint64]*model.RoundStats)
	d.state.RoundHasKill = false
	d.state.TradeDetector.Reset()
	d.state.RoundDecided = false
	d.state.RoundDecidedAt = 0
	d.state.BombPlanted = false
	d.state.RoundStartState = nil

	// Clear any pending probability snapshots from skipped/aborted rounds
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
		currentTime := float64(d.parser.CurrentFrame()) / float64(rating.TickRate)
		timeInRound := currentTime - d.state.RoundStartTime
		plantSwing := d.state.SwingTracker.RecordBombPlant(e.Player.SteamID64, timeInRound)
		roundStats.ProbabilitySwing += plantSwing
		roundStats.AddSwingContribution(model.SwingContribution{
			Type:        "bomb_plant",
			Amount:      plantSwing,
			TimeInRound: timeInRound,
		})
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

	// Calculate time in round
	currentTime := float64(d.parser.CurrentFrame()) / float64(rating.TickRate)
	timeInRound := currentTime - d.state.RoundStartTime

	// Track bomb defuse swing
	if d.state.SwingTracker != nil {
		defuseSwing := d.state.SwingTracker.RecordBombDefuse(e.Player.SteamID64, timeInRound)
		roundStats.ProbabilitySwing += defuseSwing
		roundStats.AddSwingContribution(model.SwingContribution{
			Type:        "bomb_defuse",
			Amount:      defuseSwing,
			TimeInRound: timeInRound,
		})
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
	currentTime := float64(d.parser.CurrentFrame()) / float64(rating.TickRate)
	timeInRound := currentTime - d.state.RoundStartTime
	d.state.RoundDecided = true
	d.state.RoundDecidedAt = timeInRound

	// Record state snapshot at bomb explosion (e.g. 0v3_planted or 2v1_planted)
	if d.collector != nil {
		gs := d.parser.GameState()
		tAlive, ctAlive := d.state.CountAlivePlayers(gs.Participants().Playing())
		d.collector.RecordStateSnapshot(tAlive, ctAlive, true) // bomb is planted
	}

	// Track bomb explode event
	if d.state.SwingTracker != nil {
		d.state.SwingTracker.RecordBombExplode(timeInRound)
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

			// Track flash for swing attribution
			if d.state.SwingTracker != nil {
				d.state.SwingTracker.RecordFlash(e.Attacker.SteamID64, e.Player.SteamID64, flashDuration)
			}
		} else if e.Attacker.SteamID64 != e.Player.SteamID64 {
			roundStats.TeamFlashCount++
			roundStats.TeamFlashDuration += flashDuration
		}
	}
}

// handleGrenadeThrow tracks flash grenade throws.
func (d *DemoParser) handleGrenadeThrow(e events.GrenadeProjectileThrow) {
	if d.state.ShouldSkipEvent() {
		return
	}

	if e.Projectile != nil && e.Projectile.Thrower != nil {
		if e.Projectile.WeaponInstance != nil && e.Projectile.WeaponInstance.Type == common.EqFlash {
			roundStats := d.state.ensureRound(e.Projectile.Thrower)
			roundStats.FlashesThrown++
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
	d.state.IsKnifeRound = false
	d.state.RoundNumber++

	d.state.IsPistolRound = rating.IsPistolRound(d.state.RoundNumber)

	d.state.RoundStartTime = float64(d.parser.CurrentFrame()) / float64(rating.TickRate)

	for _, p := range participants {
		if p.Team == common.TeamTerrorists {
			d.state.CurrentSide = "T"
			break
		} else if p.Team == common.TeamCounterTerrorists {
			d.state.CurrentSide = "CT"
			break
		}
	}

	d.logger.LogRoundStart(d.state.RoundNumber)

	// Count players and calculate team economies for swing tracking
	tAlive := 0
	ctAlive := 0
	tEquipTotal := 0
	ctEquipTotal := 0

	for _, p := range participants {
		if p.IsBot {
			continue
		}
		d.state.ensurePlayer(p)
		roundStats := d.state.ensureRound(p)
		roundStats.IsPistolRound = d.state.IsPistolRound
		roundStats.EquipmentValue = float64(p.EquipmentValueCurrent())

		if p.Team == common.TeamTerrorists {
			roundStats.PlayerSide = "T"
			tAlive++
			tEquipTotal += p.EquipmentValueCurrent()
		} else if p.Team == common.TeamCounterTerrorists {
			roundStats.PlayerSide = "CT"
			ctAlive++
			ctEquipTotal += p.EquipmentValueCurrent()
		}
	}

	// Cap at 5 per side as safety net (CS2 is 5v5)
	if tAlive > 5 {
		tAlive = 5
	}
	if ctAlive > 5 {
		ctAlive = 5
	}

	// Initialize swing tracker for the round
	if d.state.SwingTracker != nil && d.state.SwingTracker.IsEnabled() {
		d.state.SwingTracker.ResetRound(tAlive, ctAlive, d.state.MapName)

		// Set team economies
		tAvgEquip := 0.0
		ctAvgEquip := 0.0
		if tAlive > 0 {
			tAvgEquip = float64(tEquipTotal) / float64(tAlive)
		}
		if ctAlive > 0 {
			ctAvgEquip = float64(ctEquipTotal) / float64(ctAlive)
		}
		d.state.SwingTracker.SetEconomyFromValues(tAvgEquip, ctAvgEquip)

		// Store initial state for end-of-round calculation
		d.state.RoundStartState = probability.NewRoundState(tAlive, ctAlive, d.state.MapName)
		d.state.RoundStartState.TEconomy = probability.CategorizeEquipment(tAvgEquip)
		d.state.RoundStartState.CTEconomy = probability.CategorizeEquipment(ctAvgEquip)
	}
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
	if d.parser.GameState().IsWarmupPeriod() || d.state.IsKnifeRound {
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

	victim := d.state.ensurePlayer(ctx.victim)
	victim.Deaths++
	victimRound := d.state.ensureRound(ctx.victim)
	victimRound.DeathTime = ctx.timeInRound

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
			ctx.attacker, ctx.victim, ctx.currentTick, ctx.timeInRound, d.state.Players, d.state.Round)
		if tradeResult.IsTrade {
			attackerStats := d.state.ensurePlayer(ctx.attacker)
			attackerStats.TradeDenials++
			attackerStats.SavedTeammate++
			attackerRound := d.state.ensureRound(ctx.attacker)
			attackerRound.SavedTeammate = true

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
	attacker := d.state.ensurePlayer(ctx.attacker)
	victim := d.state.ensurePlayer(ctx.victim)
	round := d.state.ensureRound(ctx.attacker)

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
	attacker.Kills++
	attacker.EcoKillValue += ctx.killValue
	attacker.RoundImpact += ctx.killValue
	attacker.EconImpact += ctx.killValue
	if ctx.event.IsHeadshot {
		attacker.Headshots++
	}

	// Calculate proper TTK (time from first damage to kill)
	if d.state.SwingTracker != nil {
		ttk := d.state.SwingTracker.GetTimeToKill(ctx.attacker.SteamID64, ctx.victim.SteamID64, ctx.timeInRound)
		if ttk >= 0 {
			attacker.TotalTimeToKill += ttk
			attacker.KillsWithTTK++
		}
	}

	if ctx.killValue < 1.0 {
		attacker.LowBuyKills++
	}
	if ctx.killValue <= 0.85 {
		attacker.DisadvantagedBuyKills++
	}

	victim.EcoDeathValue += ctx.deathPenalty
}

// processWeaponStats updates weapon-specific statistics.
func (d *DemoParser) processWeaponStats(ctx *killContext) {
	if ctx.event.Weapon == nil {
		return
	}

	attacker := d.state.ensurePlayer(ctx.attacker)
	round := d.state.ensureRound(ctx.attacker)

	switch ctx.event.Weapon.Type {
	case common.EqAWP:
		round.AWPKills++
		round.AWPKill = true
		attacker.AWPKills++
	case common.EqKnife:
		round.KnifeKill = true
	case common.EqHE, common.EqMolotov, common.EqIncendiary:
		round.UtilityKills++
		attacker.UtilityKills++
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

	attacker := d.state.ensurePlayer(ctx.attacker)
	victim := d.state.ensurePlayer(ctx.victim)
	round := d.state.ensureRound(ctx.attacker)
	victimRound := d.state.ensureRound(ctx.victim)

	attacker.OpeningKills++
	attacker.OpeningAttempts++
	attacker.OpeningSuccesses++
	round.OpeningKill = true
	round.EntryFragger = true
	round.InvolvedInOpening = true

	if ctx.event.Weapon != nil && ctx.event.Weapon.Type == common.EqAWP {
		round.AWPOpeningKill = true
		attacker.AWPOpeningKills++
	}

	victim.OpeningDeaths++
	victim.OpeningAttempts++
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

	swingResult := d.state.SwingTracker.RecordKill(
		ctx.attacker.SteamID64, ctx.victim.SteamID64,
		ctx.attacker.Team, ctx.victim.Team,
		float64(ctx.attackerEquip), float64(ctx.victimEquip),
		ctx.timeInRound,
		ctx.isTradeKill, ctx.event.IsHeadshot,
	)

	round.ProbabilitySwing += swingResult.KillerSwing

	victimRound := d.state.ensureRound(ctx.victim)
	victimContribution := -swingResult.VictimSwing
	victimRound.ProbabilitySwing += victimContribution
	d.addKillSwingContribution(ctx, swingResult, victimContribution)

	// Credit damage contributors and flash assisters with their share of the kill swing
	for contributorID, contributorSwing := range swingResult.ContributorSwings {
		if contributorRound, ok := d.state.Round[contributorID]; ok {
			contributorRound.ProbabilitySwing += contributorSwing
			contributorRound.AddSwingContribution(model.SwingContribution{
				Type:        "assist",
				Amount:      contributorSwing,
				TimeInRound: ctx.timeInRound,
				Opponent:    ctx.victim.Name,
			})
		}
	}

	if swingResult.EcoMultiplier > 0 {
		attacker := d.state.ensurePlayer(ctx.attacker)
		attacker.EcoAdjustedKills += swingResult.EcoMultiplier
	}
}

// processEcoKillFlags sets eco kill and anti-eco flags.
func (d *DemoParser) processEcoKillFlags(ctx *killContext) {
	round := d.state.ensureRound(ctx.attacker)
	victimRound := d.state.ensureRound(ctx.victim)
	attacker := d.state.ensurePlayer(ctx.attacker)

	equipRatio := float64(ctx.victimEquip) / math.Max(float64(ctx.attackerEquip), 500.0)
	if equipRatio > 2.0 {
		round.EcoKill = true
	}
	if equipRatio < 0.5 {
		victimRound.AntiEcoKill = true
	}
	if ctx.event.IsHeadshot {
		attacker.PerfectKills++
	}
}

// processAssist handles assist statistics.
func (d *DemoParser) processAssist(ctx *killContext) {
	if ctx.event.Assister == nil {
		return
	}

	assister := d.state.ensurePlayer(ctx.event.Assister)
	assister.Assists++
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
	if d.parser.GameState().IsWarmupPeriod() || d.state.IsKnifeRound {
		return
	}

	if e.Attacker == nil || e.Player == nil {
		return
	}

	if e.Attacker.Team != e.Player.Team {
		ps := d.state.ensurePlayer(e.Attacker)
		ps.Damage += int(e.HealthDamageTaken)

		roundStats := d.state.ensureRound(e.Attacker)
		roundStats.Damage += int(e.HealthDamageTaken)

		if e.Weapon != nil {
			switch e.Weapon.Type {
			case common.EqHE, common.EqMolotov, common.EqIncendiary:
				roundStats.UtilityDamage += int(e.HealthDamageTaken)
			}
		}

		// Track damage for swing attribution and TTK calculation
		if d.state.SwingTracker != nil {
			currentTime := float64(d.parser.CurrentFrame()) / float64(rating.TickRate)
			timeInRound := currentTime - d.state.RoundStartTime
			d.state.SwingTracker.RecordDamage(e.Attacker.SteamID64, e.Player.SteamID64, int(e.HealthDamageTaken), timeInRound)
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
		currentTime := float64(d.parser.CurrentFrame()) / float64(rating.TickRate)
		timeInRound := currentTime - d.state.RoundStartTime
		d.state.RoundDecided = true
		d.state.RoundDecidedAt = timeInRound
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
	roundContext  *model.RoundContext
}

// handleRoundEnd processes the end of a round, updating all player statistics.
func (d *DemoParser) handleRoundEnd(e events.RoundEnd) {
	if d.parser.GameState().IsWarmupPeriod() || d.state.IsKnifeRound {
		return
	}

	ctx := d.buildRoundEndContext(e)

	d.processRoundEndTrades()
	d.processMultiKills()
	d.processSurvivalStats(ctx)
	d.processClutchDetection(ctx)
	d.processProbabilitySwings(ctx)
	d.updateSideStats()
	d.incrementRoundsPlayed()
	d.updateTeamScores(ctx.winnerTeam)
	d.recordRoundEndProbability(ctx)

	d.logger.LogRoundEnd(d.state.RoundNumber)
}

// buildRoundEndContext creates the context for round end processing.
func (d *DemoParser) buildRoundEndContext(e events.RoundEnd) *roundEndContext {
	gs := d.parser.GameState()
	roundEndTime := float64(d.parser.CurrentFrame()) / float64(rating.TickRate)
	roundDuration := roundEndTime - d.state.RoundStartTime
	currentTime := float64(d.parser.CurrentFrame()) / float64(rating.TickRate)
	timeRemaining := math.Max(0.0, 115.0-(currentTime-d.state.RoundStartTime))

	roundContext := model.NewRoundContextBuilder().
		WithRoundNumber(d.state.RoundNumber).
		WithScores(d.state.TeamScore, d.state.EnemyScore).
		WithRoundType(determineRoundType(d.state.RoundNumber)).
		WithTimeRemaining(timeRemaining).
		WithOvertime(d.state.RoundNumber > 30).
		WithMapSide(d.state.CurrentSide).
		WithRoundDecision(d.state.RoundDecided, d.state.RoundDecidedAt).
		CalculateImportance().
		BuildFromRoundStats(d.state.Round)

	return &roundEndContext{
		gs:            gs,
		winnerTeam:    e.Winner,
		roundDuration: roundDuration,
		timeRemaining: timeRemaining,
		roundContext:  roundContext,
	}
}

// processRoundEndTrades handles pending trades at round end.
func (d *DemoParser) processRoundEndTrades() {
	currentTick := d.parser.CurrentFrame()
	d.state.TradeDetector.ProcessRoundEndTrades(currentTick, d.state.Round)
}

// processMultiKills updates multi-kill statistics.
func (d *DemoParser) processMultiKills() {
	for steamID, roundStats := range d.state.Round {
		player := d.state.Players[steamID]
		if player == nil {
			continue
		}

		if roundStats.Kills >= 1 && roundStats.Kills <= 5 {
			player.MultiKillsRaw[roundStats.Kills]++
			d.logger.LogMultiKill(d.state.RoundNumber, player.Name, roundStats.Kills)
		}

		if player.RoundsPlayed > 0 {
			player.AWPKillsPerRound = float64(player.AWPKills) / float64(player.RoundsPlayed)
		}
	}
}

// processSurvivalStats updates survival and time alive statistics.
func (d *DemoParser) processSurvivalStats(ctx *roundEndContext) {
	for _, p := range ctx.gs.Participants().Playing() {
		ps := d.state.ensurePlayer(p)
		round := d.state.ensureRound(p)

		teamWon := p.Team == ctx.winnerTeam
		round.TeamWon = teamWon
		if teamWon {
			ps.RoundsWon++
		} else {
			ps.RoundsLost++
		}

		if p.IsAlive() {
			ps.Survival++
			round.Survived = true
			round.TimeAlive = ctx.roundDuration
			ps.TotalTimeAlive += ctx.roundDuration

			if !teamWon {
				ps.SavesOnLoss++
			}
		} else if round.DeathTime > 0 {
			round.TimeAlive = round.DeathTime
			ps.TotalTimeAlive += round.DeathTime
		}
	}
}

// processClutchDetection detects and records clutch situations.
func (d *DemoParser) processClutchDetection(ctx *roundEndContext) {
	for _, p := range ctx.gs.Participants().Playing() {
		round := d.state.ensureRound(p)
		ps := d.state.ensurePlayer(p)

		aliveTeammates, aliveEnemies := d.countAliveByTeam(ctx.gs.Participants().Playing(), p.Team)

		if p.IsAlive() && aliveTeammates == 1 {
			ps.LastAliveRounds++
			round.WasLastAlive = true

			if aliveEnemies > 0 {
				d.recordClutchAttempt(ps, round, aliveEnemies)
			}
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

// recordClutchAttempt records a clutch attempt and its outcome.
func (d *DemoParser) recordClutchAttempt(ps *model.PlayerStats, round *model.RoundStats, aliveEnemies int) {
	round.ClutchAttempt = true
	round.ClutchSize = aliveEnemies
	round.ClutchKills = round.Kills
	ps.ClutchRounds++

	if aliveEnemies == 1 {
		ps.Clutch1v1Attempts++
		if round.TeamWon {
			ps.Clutch1v1Wins++
		}
	}

	if round.TeamWon {
		round.ClutchWon = true
		ps.ClutchWins++
	}
}

// processProbabilitySwings accumulates probability swing values per player.
func (d *DemoParser) processProbabilitySwings(ctx *roundEndContext) {
	for steamID, roundStats := range d.state.Round {
		player := d.state.Players[steamID]
		if player == nil {
			continue
		}

		roundStats.MultiKillRound = roundStats.Kills

		player.ProbabilitySwing += roundStats.ProbabilitySwing
		player.RoundBreakdowns = append(player.RoundBreakdowns, model.NewRoundSwingBreakdown(d.state.RoundNumber, roundStats))

		if roundStats.PlayerSide == "T" {
			player.TProbabilitySwing += roundStats.ProbabilitySwing
		} else if roundStats.PlayerSide == "CT" {
			player.CTProbabilitySwing += roundStats.ProbabilitySwing
		}
	}
}

// updateSideStats applies side-specific statistics using SideStatsUpdater.
func (d *DemoParser) updateSideStats() {
	for steamID, roundStats := range d.state.Round {
		player := d.state.Players[steamID]
		if player == nil {
			continue
		}

		updater := NewSideStatsUpdater(player, roundStats)
		updater.UpdateCommonRoundStats()
		updater.UpdateSideStats()
	}
}

// incrementRoundsPlayed increments rounds played for all players.
func (d *DemoParser) incrementRoundsPlayed() {
	for _, p := range d.state.Players {
		p.RoundsPlayed++
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

// recordRoundEndProbability records round outcome for probability collection.
func (d *DemoParser) recordRoundEndProbability(ctx *roundEndContext) {
	if d.collector == nil {
		return
	}

	tAlive, ctAlive := d.state.CountAlivePlayers(ctx.gs.Participants().Playing())

	// Only snapshot the final state for non-elimination endings (time expiry,
	// bomb scenarios with survivors on both sides). Elimination rounds are fully
	// captured by kill snapshots. The RoundEnd event can fire with unreliable
	// player alive states (engine resetting for next round), producing false
	// Xv0 or 0vX snapshots.
	if tAlive > 0 && ctAlive > 0 {
		d.collector.RecordStateSnapshot(tAlive, ctAlive, d.state.BombPlanted)
	}

	d.collector.RecordRoundEnd(tAlive, ctAlive, d.state.BombPlanted, ctx.winnerTeam, d.state.MapName)
}

// determineRoundType categorizes a round as pistol, eco, force, or full buy
// based on the round number. Uses MR12 format constants.
func determineRoundType(roundNumber int) string {
	if rating.IsPistolRound(roundNumber) {
		return "pistol"
	}

	// Eco rounds: typically rounds 2-3 after pistol (first half) and 14-15 (second half)
	isFirstHalfEco := roundNumber >= 2 && roundNumber <= 3
	isSecondHalfEco := roundNumber >= rating.SecondHalfPistolRound+1 && roundNumber <= rating.SecondHalfPistolRound+2

	if isFirstHalfEco || isSecondHalfEco {
		return "eco"
	}

	// Force buy rounds (simplified heuristic)
	if roundNumber%3 == 0 {
		return "force"
	}

	return "full"
}
