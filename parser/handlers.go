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
	"math"

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
	if d.state.IsKnifeRound || !d.state.MatchStarted {
		return
	}

	// Record state snapshot BEFORE bomb plant
	if d.collector != nil {
		gs := d.parser.GameState()
		tAlive := 0
		ctAlive := 0
		for _, p := range gs.Participants().Playing() {
			if p.IsAlive() {
				if p.Team == common.TeamTerrorists {
					tAlive++
				} else if p.Team == common.TeamCounterTerrorists {
					ctAlive++
				}
			}
		}
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
	}

	d.logger.LogBombPlant(d.state.RoundNumber, planter.Name)
}

// handleBombDefused processes a bomb defuse event.
func (d *DemoParser) handleBombDefused(e events.BombDefused) {
	if d.state.IsKnifeRound || !d.state.MatchStarted {
		return
	}

	// Record state snapshot before defuse
	if d.collector != nil {
		gs := d.parser.GameState()
		tAlive := 0
		ctAlive := 0
		for _, p := range gs.Participants().Playing() {
			if p.IsAlive() {
				if p.Team == common.TeamTerrorists {
					tAlive++
				} else if p.Team == common.TeamCounterTerrorists {
					ctAlive++
				}
			}
		}
		d.collector.RecordStateSnapshot(tAlive, ctAlive, true) // bomb is planted
	}

	defuser := d.state.ensurePlayer(e.Player)
	roundStats := d.state.ensureRound(e.Player)
	roundStats.DefusedBomb = true

	// Track bomb defuse swing
	if d.state.SwingTracker != nil {
		currentTime := float64(d.parser.CurrentFrame()) / float64(rating.TickRate)
		timeInRound := currentTime - d.state.RoundStartTime
		defuseSwing := d.state.SwingTracker.RecordBombDefuse(e.Player.SteamID64, timeInRound)
		roundStats.ProbabilitySwing += defuseSwing
	}

	d.logger.LogBombDefuse(d.state.RoundNumber, defuser.Name)
}

// handleBombExplode marks the round as decided when the bomb explodes.
func (d *DemoParser) handleBombExplode() {
	if d.state.IsKnifeRound || !d.state.MatchStarted {
		return
	}
	currentTime := float64(d.parser.CurrentFrame()) / float64(rating.TickRate)
	timeInRound := currentTime - d.state.RoundStartTime
	d.state.RoundDecided = true
	d.state.RoundDecidedAt = timeInRound

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
	if d.state.IsKnifeRound || !d.state.MatchStarted {
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
	if d.state.IsKnifeRound || !d.state.MatchStarted {
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

// handleKill processes a kill event, updating statistics for killer and victim.
func (d *DemoParser) handleKill(e events.Kill) {
	if d.parser.GameState().IsWarmupPeriod() || d.state.IsKnifeRound {
		return
	}

	a := e.Killer
	v := e.Victim

	if a != nil && v != nil && a.SteamID64 == v.SteamID64 {
		return
	}

	if a != nil && v != nil && a.Team == v.Team {
		return
	}

	currentTick := d.parser.CurrentFrame()

	currentTime := float64(currentTick) / float64(rating.TickRate)
	timeInRound := currentTime - d.state.RoundStartTime

	// Process victim death and record for trade detection
	if v != nil {
		victim := d.state.ensurePlayer(v)
		victim.Deaths++
		victimRound := d.state.ensureRound(v)
		victimRound.DeathTime = timeInRound

		// Check for AWP loss
		for _, weapon := range v.Weapons() {
			if weapon.Type == common.EqAWP {
				victimRound.HadAWP = true
				victimRound.LostAWP = true
				break
			}
		}

		// Record death for trade detection using TradeDetector
		gs := d.parser.GameState()
		d.state.TradeDetector.RecordDeath(v, a, currentTick, timeInRound, gs.Participants().Playing())
	}

	// Check if this kill is a trade using TradeDetector
	if a != nil && v != nil {
		tradeResult := d.state.TradeDetector.CheckForTrade(a, v, currentTick, timeInRound, d.state.Players, d.state.Round)
		if tradeResult.IsTrade {
			attackerStats := d.state.ensurePlayer(a)
			attackerStats.TradeDenials++
			attackerStats.SavedTeammate++
			attackerRound := d.state.ensureRound(a)
			attackerRound.SavedTeammate = true

			d.logger.LogTrade(d.state.RoundNumber, a.Name, tradeResult.TradedPlayerName, v.Name)
		}
	}

	// Process expired pending trades using TradeDetector
	d.state.TradeDetector.ProcessExpiredTrades(currentTick, d.state.Round)

	if a == nil || v == nil {
		return
	}

	// Record kill for future trade detection
	d.state.TradeDetector.RecordKill(a, v, currentTick)

	attacker := d.state.ensurePlayer(a)
	victim := d.state.ensurePlayer(v)
	round := d.state.ensureRound(a)

	attackerEquip := a.EquipmentValueCurrent()
	victimEquip := v.EquipmentValueCurrent()

	// Record kill for probability data collection
	if d.collector != nil {
		// Record state snapshot BEFORE this kill (victim still counted as alive)
		gs := d.parser.GameState()
		tAlive := 0
		ctAlive := 0
		for _, p := range gs.Participants().Playing() {
			if p.IsAlive() {
				if p.Team == common.TeamTerrorists {
					tAlive++
				} else if p.Team == common.TeamCounterTerrorists {
					ctAlive++
				}
			}
		}
		d.collector.RecordStateSnapshot(tAlive, ctAlive, d.state.BombPlanted)
		d.collector.RecordKill(float64(attackerEquip), float64(victimEquip))
	}

	killValue := rating.EcoKillValue(float64(attackerEquip), float64(victimEquip))
	deathPenalty := rating.EcoDeathPenalty(float64(victimEquip), float64(attackerEquip))

	d.logger.LogKill(d.state.RoundNumber, a.Name, v.Name, attackerEquip, victimEquip, killValue)
	d.logger.LogDeath(d.state.RoundNumber, v.Name, a.Name, victimEquip, attackerEquip, deathPenalty)

	round.KillTimes = append(round.KillTimes, timeInRound)

	if d.state.RoundDecided {
		round.IsExitFrag = true
		round.ExitFrags++
	}

	round.Kills++
	round.GotKill = true
	round.EconImpact += killValue
	attacker.Kills++
	attacker.EcoKillValue += killValue
	attacker.RoundImpact += killValue
	attacker.EconImpact += killValue

	if killValue < 1.0 {
		attacker.LowBuyKills++
	}
	if killValue <= 0.85 {
		attacker.DisadvantagedBuyKills++
	}

	if e.Weapon != nil {
		switch e.Weapon.Type {
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

		isPistol := e.Weapon.Type >= common.EqP2000 && e.Weapon.Type <= common.EqRevolver
		victimHadRifle := victimEquip > 3500
		if isPistol && victimHadRifle {
			round.PistolVsRifleKill = true
		}
	}

	victim.EcoDeathValue += deathPenalty

	victimRound := d.state.ensureRound(v)
	if !d.state.RoundHasKill {
		attacker.OpeningKills++
		attacker.OpeningAttempts++
		attacker.OpeningSuccesses++
		round.OpeningKill = true
		round.EntryFragger = true
		round.InvolvedInOpening = true

		if e.Weapon != nil && e.Weapon.Type == common.EqAWP {
			round.AWPOpeningKill = true
			attacker.AWPOpeningKills++
		}

		victim.OpeningDeaths++
		victim.OpeningAttempts++
		victimRound.OpeningDeath = true
		victimRound.InvolvedInOpening = true

		d.state.RoundHasKill = true
		d.logger.LogOpeningKill(d.state.RoundNumber, a.Name, v.Name)
	}

	// Check if this kill is a trade kill using TradeDetector
	isTradeKill, tradeSpeed := d.state.TradeDetector.CheckTradeKill(a, v, currentTick, timeInRound)
	if isTradeKill {
		round.TradeKill = true
		round.TradeSpeed = tradeSpeed
	}

	// Track kill swing using probability-based calculator
	if d.state.SwingTracker != nil {
		swingResult := d.state.SwingTracker.RecordKill(
			a.SteamID64, v.SteamID64,
			a.Team, v.Team,
			float64(attackerEquip), float64(victimEquip),
			timeInRound,
			isTradeKill, e.IsHeadshot,
		)

		// Killer gets economy-adjusted swing (bonus for hard kills like pistol vs rifle)
		round.ProbabilitySwing += swingResult.KillerSwing

		// Victim gets economy-adjusted penalty (extra penalty for embarrassing deaths like rifle dying to pistol)
		victimRound.ProbabilitySwing -= swingResult.VictimSwing

		// Track eco-adjusted kills using the multiplier
		if swingResult.EcoMultiplier > 0 {
			attacker.EcoAdjustedKills += swingResult.EcoMultiplier
		}
	}

	equipRatio := float64(victimEquip) / math.Max(float64(attackerEquip), 500.0)
	if equipRatio > 2.0 {
		round.EcoKill = true
	}
	if equipRatio < 0.5 {
		victimRound.AntiEcoKill = true
	}
	if e.IsHeadshot {
		attacker.PerfectKills++
	}

	if e.Assister != nil {
		assister := d.state.ensurePlayer(e.Assister)
		assister.Assists++
		assistRound := d.state.ensureRound(e.Assister)
		assistRound.GotAssist = true
	}
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

		// Track damage for swing attribution
		if d.state.SwingTracker != nil {
			d.state.SwingTracker.RecordDamage(e.Attacker.SteamID64, e.Player.SteamID64, int(e.HealthDamageTaken))
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
	if d.state.IsKnifeRound || !d.state.MatchStarted || d.state.RoundDecided {
		return
	}

	gs := d.parser.GameState()
	tAlive := 0
	ctAlive := 0
	for _, p := range gs.Participants().Playing() {
		if p.IsAlive() {
			if p.Team == common.TeamTerrorists {
				tAlive++
			} else if p.Team == common.TeamCounterTerrorists {
				ctAlive++
			}
		}
	}

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

// handleRoundEnd processes the end of a round, updating all player statistics.
func (d *DemoParser) handleRoundEnd(e events.RoundEnd) {
	if d.parser.GameState().IsWarmupPeriod() || d.state.IsKnifeRound {
		return
	}

	gs := d.parser.GameState()
	winnerTeam := e.Winner

	currentTick := d.parser.CurrentFrame()

	// Process any remaining pending trades at round end using TradeDetector
	d.state.TradeDetector.ProcessRoundEndTrades(currentTick, d.state.Round)

	for steamID, roundStats := range d.state.Round {
		player := d.state.Players[steamID]
		if player == nil {
			continue
		}

		if roundStats.Kills >= 2 && roundStats.Kills <= 5 {
			player.MultiKillsRaw[roundStats.Kills]++
			d.logger.LogMultiKill(d.state.RoundNumber, player.Name, roundStats.Kills)
		}

		if player.RoundsPlayed > 0 {
			player.AWPKillsPerRound = float64(player.AWPKills) / float64(player.RoundsPlayed)
		}
	}

	roundEndTime := float64(d.parser.CurrentFrame()) / float64(rating.TickRate)
	roundDuration := roundEndTime - d.state.RoundStartTime

	for _, p := range gs.Participants().Playing() {
		ps := d.state.ensurePlayer(p)
		round := d.state.ensureRound(p)

		teamWon := p.Team == winnerTeam
		round.TeamWon = teamWon
		if teamWon {
			ps.RoundsWon++
		} else {
			ps.RoundsLost++
		}

		if p.IsAlive() {
			ps.Survival++
			round.Survived = true
			round.TimeAlive = roundDuration
			ps.TotalTimeAlive += roundDuration

			if !teamWon {
				ps.SavesOnLoss++
			}
		} else if round.DeathTime > 0 {
			round.TimeAlive = round.DeathTime
			ps.TotalTimeAlive += round.DeathTime
		}
	}

	for _, p := range gs.Participants().Playing() {
		round := d.state.ensureRound(p)
		ps := d.state.ensurePlayer(p)

		aliveTeammates := 0
		aliveEnemies := 0

		for _, other := range gs.Participants().Playing() {
			if other.IsAlive() {
				if other.Team == p.Team {
					aliveTeammates++
				} else {
					aliveEnemies++
				}
			}
		}

		if p.IsAlive() && aliveTeammates == 1 {
			ps.LastAliveRounds++
			round.WasLastAlive = true

			if aliveEnemies > 0 {
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
		}

		if p.IsAlive() && !round.TeamWon {
			round.SavedWeapons = true
		}
	}

	currentTime := float64(d.parser.CurrentFrame()) / float64(rating.TickRate)
	timeRemaining := math.Max(0.0, 115.0-(currentTime-d.state.RoundStartTime))

	// Use RoundContextBuilder for cleaner context creation
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

	for steamID, roundStats := range d.state.Round {
		player := d.state.Players[steamID]
		if player == nil {
			continue
		}

		roundStats.MultiKillRound = roundStats.Kills

		var playerEquipValue float64
		var teamEquipValue float64

		for _, p := range gs.Participants().Playing() {
			if p.SteamID64 == steamID {
				playerEquipValue = float64(p.EquipmentValueCurrent())

				teamTotal := 0
				teamCount := 0
				for _, teammate := range gs.Participants().Playing() {
					if teammate.Team == p.Team {
						teamTotal += teammate.EquipmentValueCurrent()
						teamCount++
					}
				}
				if teamCount > 0 {
					teamEquipValue = float64(teamTotal)
				}
				break
			}
		}

		if playerEquipValue == 0 {
			playerEquipValue = 3000.0
		}
		if teamEquipValue == 0 {
			teamEquipValue = 15000.0
		}

		swing := CalculateAdvancedRoundSwing(roundStats, roundContext, playerEquipValue, teamEquipValue)
		player.RoundSwing += swing

		if roundStats.PlayerSide == "T" {
			player.TRoundSwing += swing
		} else if roundStats.PlayerSide == "CT" {
			player.CTRoundSwing += swing
		}

		// Add probability-based swing to player stats
		player.ProbabilitySwing += roundStats.ProbabilitySwing

		// Track side-specific probability swing
		if roundStats.PlayerSide == "T" {
			player.TProbabilitySwing += roundStats.ProbabilitySwing
		} else if roundStats.PlayerSide == "CT" {
			player.CTProbabilitySwing += roundStats.ProbabilitySwing
		}
	}

	// Use SideStatsUpdater for cleaner stats updates
	for steamID, roundStats := range d.state.Round {
		player := d.state.Players[steamID]
		if player == nil {
			continue
		}

		updater := NewSideStatsUpdater(player, roundStats)
		updater.UpdateCommonRoundStats()
		updater.UpdateSideStats()
	}

	for _, p := range d.state.Players {
		p.RoundsPlayed++
	}

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

	d.logger.LogRoundEnd(d.state.RoundNumber)

	// Record round outcome for probability data collection
	if d.collector != nil {
		tAlive := 0
		ctAlive := 0
		for _, p := range gs.Participants().Playing() {
			if p.IsAlive() {
				if p.Team == common.TeamTerrorists {
					tAlive++
				} else if p.Team == common.TeamCounterTerrorists {
					ctAlive++
				}
			}
		}
		d.collector.RecordRoundEnd(tAlive, ctAlive, d.state.BombPlanted, winnerTeam, d.state.MapName)
	}
}
