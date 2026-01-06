package parser

import (
	"eco-rating/model"
	"eco-rating/rating"
	"math"

	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
)

func (d *DemoParser) registerHandlers() {
	// Track when match actually starts (after warmup/knife round)
	d.parser.RegisterEventHandler(func(e events.MatchStart) {
		d.state.MatchStarted = true
	})

	// Also handle MatchStartedChanged for demos where MatchStart isn't sent
	d.parser.RegisterEventHandler(func(e events.MatchStartedChanged) {
		if e.NewIsStarted {
			d.state.MatchStarted = true
		}
	})

	d.parser.RegisterEventHandler(func(e events.RoundStart) {
		d.state.Round = make(map[uint64]*model.RoundStats)
		d.state.RoundHasKill = false
		d.state.RecentKills = make(map[uint64]recentKill)
		d.state.RecentTeamDeaths = make(map[uint64]float64)
		d.state.RoundDecided = false
		d.state.RoundDecidedAt = 0
	})

	// Track bomb plants
	d.parser.RegisterEventHandler(func(e events.BombPlanted) {
		if d.state.IsKnifeRound || !d.state.MatchStarted {
			return
		}

		planter := d.state.ensurePlayer(e.Player)
		roundStats := d.state.ensureRound(e.Player)
		roundStats.PlantedBomb = true

		d.logger.LogBombPlant(d.state.RoundNumber, planter.Name)
	})

	// Track bomb defuses
	d.parser.RegisterEventHandler(func(e events.BombDefused) {
		if d.state.IsKnifeRound || !d.state.MatchStarted {
			return
		}

		defuser := d.state.ensurePlayer(e.Player)
		roundStats := d.state.ensureRound(e.Player)
		roundStats.DefusedBomb = true

		d.logger.LogBombDefuse(d.state.RoundNumber, defuser.Name)
	})

	// Track flash assists and team flashes
	d.parser.RegisterEventHandler(func(e events.PlayerFlashed) {
		if d.state.IsKnifeRound || !d.state.MatchStarted {
			return
		}

		if e.Attacker != nil && e.Player != nil {
			roundStats := d.state.ensureRound(e.Attacker)
			if e.Attacker.Team != e.Player.Team {
				// Enemy flash - count as flash assist
				roundStats.FlashAssists++
			} else if e.Attacker.SteamID64 != e.Player.SteamID64 {
				// Team flash (not self-flash)
				roundStats.TeamFlashCount++
				roundStats.TeamFlashDuration += float64(e.FlashDuration().Seconds())
			}
		}
	})

	// Initialize round stats for all players at freeze time end (when round actually begins)
	d.parser.RegisterEventHandler(func(e events.RoundFreezetimeEnd) {
		gs := d.parser.GameState()
		if gs.IsWarmupPeriod() {
			return
		}
		// Skip knife round - check if players have $800 (pistol round) or more
		// Knife round players have $0
		participants := gs.Participants().Playing()
		if len(participants) > 0 {
			firstPlayer := participants[0]
			if firstPlayer.Money()+firstPlayer.MoneySpentThisRound() == 0 {
				// This is a knife round, skip it
				d.state.IsKnifeRound = true
				d.logger.LogKnifeRound()
				return
			}
		}
		d.state.IsKnifeRound = false
		d.state.RoundNumber++

		// Track round start time and determine current side
		d.state.RoundStartTime = float64(d.parser.CurrentFrame()) / 64.0 // Convert ticks to seconds (64 tick rate)

		// Determine current side based on first T player (for perspective)
		for _, p := range participants {
			if p.Team == common.TeamTerrorists {
				d.state.CurrentSide = "T"
				break
			} else if p.Team == common.TeamCounterTerrorists {
				d.state.CurrentSide = "CT"
				break
			}
		}

		// Log round start
		d.logger.LogRoundStart(d.state.RoundNumber)

		// Initialize round stats for ALL playing participants
		for _, p := range participants {
			d.state.ensurePlayer(p)
			d.state.ensureRound(p)
		}
	})

	d.parser.RegisterEventHandler(func(e events.Kill) {
		// Skip warmup and knife round
		if d.parser.GameState().IsWarmupPeriod() || d.state.IsKnifeRound {
			return
		}

		a := e.Killer
		v := e.Victim

		// Skip self-kills (suicides) - they shouldn't count as kills
		if a != nil && v != nil && a.SteamID64 == v.SteamID64 {
			return
		}

		currentTick := d.parser.CurrentFrame()
		const tradeWindow = 64 * 5 // ~5 seconds at 64 tick (CS2)

		// Calculate current time relative to round start
		currentTime := float64(currentTick) / 64.0
		timeInRound := currentTime - d.state.RoundStartTime

		// Track victim death with timing
		if v != nil {
			victim := d.state.ensurePlayer(v)
			victim.Deaths++
			victimRound := d.state.ensureRound(v)
			victimRound.DeathTime = timeInRound

			// Track death for trade speed calculation
			d.state.RecentTeamDeaths[v.SteamID64] = timeInRound

			// Track if victim had AWP and lost it
			for _, weapon := range v.Weapons() {
				if weapon.Type == common.EqAWP {
					victimRound.HadAWP = true
					victimRound.LostAWP = true
					break
				}
			}
		}

		// Check if this kill is a trade (attacker killed someone who recently killed a teammate)
		if a != nil && v != nil {
			if recent, ok := d.state.RecentKills[v.SteamID64]; ok {
				// Victim (who just died) had recently killed someone
				// Check if that someone was on the same team as the current attacker
				if recent.VictimTeam == a.Team && currentTick-recent.Tick <= tradeWindow {
					// This is a trade! Mark the original victim as traded
					if tradedRound, exists := d.state.Round[recent.VictimID]; exists {
						tradedRound.Traded = true
					}
					tradedPlayerName := ""
					if tradedPlayer, exists := d.state.Players[recent.VictimID]; exists {
						tradedPlayer.TradedDeaths++
						tradedPlayerName = tradedPlayer.Name
					}
					// Attacker gets a trade denial credit
					d.state.ensurePlayer(a).TradeDenials++

					// Log the trade
					d.logger.LogTrade(d.state.RoundNumber, a.Name, tradedPlayerName, v.Name)
				}
			}
		}

		if a == nil || v == nil {
			return
		}

		// Record this kill for trade detection
		d.state.RecentKills[a.SteamID64] = recentKill{
			VictimID:   v.SteamID64,
			VictimTeam: v.Team,
			Tick:       currentTick,
		}

		attacker := d.state.ensurePlayer(a)
		victim := d.state.ensurePlayer(v)
		round := d.state.ensureRound(a)

		attackerEquip := a.EquipmentValueCurrent()
		victimEquip := v.EquipmentValueCurrent()

		// Eco-adjusted kill value (bonus for killing better-equipped players)
		killValue := rating.EcoKillValue(float64(attackerEquip), float64(victimEquip))
		// Eco-adjusted death penalty (higher penalty for dying to worse-equipped players)
		deathPenalty := rating.EcoDeathPenalty(float64(victimEquip), float64(attackerEquip))

		// Log the kill
		d.logger.LogKill(d.state.RoundNumber, a.Name, v.Name, attackerEquip, victimEquip, killValue)
		// Log the death
		d.logger.LogDeath(d.state.RoundNumber, v.Name, a.Name, victimEquip, attackerEquip, deathPenalty)

		// Track kill timing
		round.KillTimes = append(round.KillTimes, timeInRound)

		// Check if this is an exit frag (round already decided)
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

		// Track weapon-specific kills
		if e.Weapon != nil {
			switch e.Weapon.Type {
			case common.EqAWP:
				round.AWPKills++
				round.AWPKill = true
				attacker.AWPKills++
			case common.EqKnife:
				round.KnifeKill = true
			}

			// Check for pistol vs rifle kills
			isPistol := e.Weapon.Type >= common.EqP2000 && e.Weapon.Type <= common.EqRevolver
			victimHadRifle := victimEquip > 3500 // Rough threshold for rifle loadout
			if isPistol && victimHadRifle {
				round.PistolVsRifleKill = true
			}
		}

		// Track death penalty for victim
		victim.EcoDeathValue += deathPenalty

		// Track opening kill and entry fragging
		if !d.state.RoundHasKill {
			attacker.OpeningKills++
			round.OpeningKill = true
			round.EntryFragger = true
			d.state.RoundHasKill = true
			d.logger.LogOpeningKill(d.state.RoundNumber, a.Name, v.Name)
		}

		// Track if victim got opening death
		victimRound := d.state.ensureRound(v)
		if !d.state.RoundHasKill {
			victimRound.OpeningDeath = true
		}

		// Track trade kills with speed
		if recent, ok := d.state.RecentKills[v.SteamID64]; ok {
			if recent.VictimTeam == a.Team && currentTick-recent.Tick <= tradeWindow {
				round.TradeKill = true
				// Calculate trade speed
				if deathTime, exists := d.state.RecentTeamDeaths[recent.VictimID]; exists {
					round.TradeSpeed = timeInRound - deathTime
				}
			}
		}

		// Track eco/anti-eco kills
		equipRatio := float64(victimEquip) / math.Max(float64(attackerEquip), 500.0)
		if equipRatio > 2.0 {
			round.EcoKill = true // Attacker had much worse equipment
		}
		if equipRatio < 0.5 {
			victimRound.AntiEcoKill = true // Victim died to much worse equipment
		}
		if e.IsHeadshot {
			attacker.PerfectKills++
		}

		// Track assists
		if e.Assister != nil {
			assister := d.state.ensurePlayer(e.Assister)
			assister.Assists++
			assistRound := d.state.ensureRound(e.Assister)
			assistRound.GotAssist = true
		}
	})

	d.parser.RegisterEventHandler(func(e events.PlayerHurt) {
		// Skip warmup and knife round
		if d.parser.GameState().IsWarmupPeriod() || d.state.IsKnifeRound {
			return
		}

		if e.Attacker == nil || e.Player == nil {
			return
		}

		// Only count damage to enemies (not team damage or self damage)
		// Use HealthDamageTaken to exclude over-damage (damage beyond player's remaining health)
		if e.Attacker.Team != e.Player.Team {
			ps := d.state.ensurePlayer(e.Attacker)
			ps.Damage += int(e.HealthDamageTaken)

			// Track utility damage
			roundStats := d.state.ensureRound(e.Attacker)
			if e.Weapon != nil {
				switch e.Weapon.Type {
				case common.EqHE, common.EqMolotov, common.EqIncendiary:
					roundStats.UtilityDamage += int(e.HealthDamageTaken)
				}
			}
		}
	})

	// Track bomb explosion (round decided for T)
	d.parser.RegisterEventHandler(func(e events.BombExplode) {
		if d.state.IsKnifeRound || !d.state.MatchStarted {
			return
		}
		currentTime := float64(d.parser.CurrentFrame()) / 64.0
		timeInRound := currentTime - d.state.RoundStartTime
		d.state.RoundDecided = true
		d.state.RoundDecidedAt = timeInRound
	})

	// Track round decided when all players on one team are dead
	d.parser.RegisterEventHandler(func(e events.Kill) {
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

		// Round is decided if one team is eliminated
		if tAlive == 0 || ctAlive == 0 {
			currentTime := float64(d.parser.CurrentFrame()) / 64.0
			timeInRound := currentTime - d.state.RoundStartTime
			d.state.RoundDecided = true
			d.state.RoundDecidedAt = timeInRound
		}
	})

	d.parser.RegisterEventHandler(func(e events.RoundEnd) {
		// Skip warmup and knife round
		if d.parser.GameState().IsWarmupPeriod() || d.state.IsKnifeRound {
			return
		}

		gs := d.parser.GameState()
		winnerTeam := e.Winner

		// Track multi-kills per round
		for steamID, roundStats := range d.state.Round {
			player := d.state.Players[steamID]
			if player == nil {
				continue
			}

			// Multi-kills tracking
			if roundStats.Kills >= 2 && roundStats.Kills <= 5 {
				player.MultiKills[roundStats.Kills]++
				d.logger.LogMultiKill(d.state.RoundNumber, player.Name, roundStats.Kills)
			}

			// Calculate AWP kills per round at round end
			if player.RoundsPlayed > 0 {
				player.AWPKillsPerRound = float64(player.AWPKills) / float64(player.RoundsPlayed)
			}
		}

		// Track survival and round outcome for players
		for _, p := range gs.Participants().Playing() {
			ps := d.state.ensurePlayer(p)
			round := d.state.ensureRound(p)

			// Track if player's team won
			teamWon := p.Team == winnerTeam
			round.TeamWon = teamWon
			if teamWon {
				ps.RoundsWon++
			}

			// Track survival
			if p.IsAlive() {
				ps.Survival++
				round.Survived = true
			}
		}

		// Detect clutch situations and track clutch performance
		for _, p := range gs.Participants().Playing() {
			round := d.state.ensureRound(p)

			// Count alive teammates and enemies
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

			// Clutch situation: player is last alive on their team
			if p.IsAlive() && aliveTeammates == 1 && aliveEnemies > 0 {
				round.ClutchAttempt = true
				round.ClutchKills = round.Kills // Kills made during clutch

				if round.TeamWon {
					round.ClutchWon = true
					ps := d.state.ensurePlayer(p)
					ps.ClutchWins++
				}

				ps := d.state.ensurePlayer(p)
				ps.ClutchRounds++
			}

			// Track weapon saves (survived a lost round)
			if p.IsAlive() && !round.TeamWon {
				round.SavedWeapons = true
			}
		}

		// Calculate Advanced Round Swing for each player
		// Create round context for situational awareness
		currentTime := float64(d.parser.CurrentFrame()) / 64.0                     // Convert ticks to seconds
		timeRemaining := math.Max(0.0, 115.0-(currentTime-d.state.RoundStartTime)) // 115s round time

		// Calculate score context
		teamScore := d.state.TeamScore
		enemyScore := d.state.EnemyScore
		scoreDiff := teamScore - enemyScore
		isMatchPoint := teamScore == 12 || enemyScore == 12 || (d.state.RoundNumber > 30 && (teamScore >= 15 || enemyScore >= 15))
		isCloseGame := math.Abs(float64(scoreDiff)) <= 3

		// Calculate round importance multiplier
		roundImportance := 1.0
		if isMatchPoint {
			roundImportance = 1.3
		} else if isCloseGame {
			roundImportance = 1.15
		} else if math.Abs(float64(scoreDiff)) >= 8 {
			roundImportance = 0.85 // Blowout games matter less
		}

		roundContext := &model.RoundContext{
			RoundNumber:     d.state.RoundNumber,
			TotalPlayers:    10,    // Standard 5v5
			BombPlanted:     false, // Will be updated based on round stats
			BombDefused:     false, // Will be updated based on round stats
			RoundType:       determineRoundType(d.state.RoundNumber),
			TimeRemaining:   timeRemaining,
			IsOvertimeRound: d.state.RoundNumber > 30,
			MapSide:         d.state.CurrentSide,
			TeamScore:       teamScore,
			EnemyScore:      enemyScore,
			ScoreDiff:       scoreDiff,
			IsMatchPoint:    isMatchPoint,
			IsCloseGame:     isCloseGame,
			RoundImportance: roundImportance,
			RoundDecided:    d.state.RoundDecided,
			RoundDecidedAt:  d.state.RoundDecidedAt,
		}

		// Check for bomb events in this round
		for _, roundStats := range d.state.Round {
			if roundStats.PlantedBomb {
				roundContext.BombPlanted = true
			}
			if roundStats.DefusedBomb {
				roundContext.BombDefused = true
			}
		}

		for steamID, roundStats := range d.state.Round {
			player := d.state.Players[steamID]
			if player == nil {
				continue
			}

			// Set multi-kill count for this round
			roundStats.MultiKillRound = roundStats.Kills

			// Calculate actual player and team equipment values
			var playerEquipValue float64
			var teamEquipValue float64

			// Find the actual player in game state to get equipment value
			for _, p := range gs.Participants().Playing() {
				if p.SteamID64 == steamID {
					playerEquipValue = float64(p.EquipmentValueCurrent())

					// Calculate team equipment value
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

			// Fallback values if player not found
			if playerEquipValue == 0 {
				playerEquipValue = 3000.0
			}
			if teamEquipValue == 0 {
				teamEquipValue = 15000.0
			}

			// Calculate advanced round swing
			swing := CalculateAdvancedRoundSwing(roundStats, roundContext, playerEquipValue, teamEquipValue)
			player.RoundSwing += swing
		}

		// Track KAST and aggregate new stats for ALL players who participated this round
		for steamID, roundStats := range d.state.Round {
			player := d.state.Players[steamID]
			if player == nil {
				continue
			}
			// KAST: Kill, Assist, Survived, or Traded
			if roundStats.GotKill || roundStats.GotAssist || roundStats.Survived || roundStats.Traded {
				player.KAST++
			}

			// Aggregate new stats for export
			player.UtilityDamage += roundStats.UtilityDamage
			player.TeamFlashCount += roundStats.TeamFlashCount
			player.TeamFlashDuration += roundStats.TeamFlashDuration
			player.ExitFrags += roundStats.ExitFrags

			if roundStats.LostAWP {
				player.AWPDeaths++
				if !roundStats.AWPKill {
					player.AWPDeathsNoKill++
				}
			}

			if roundStats.KnifeKill {
				player.KnifeKills++
			}

			if roundStats.PistolVsRifleKill {
				player.PistolVsRifleKills++
			}

			if roundStats.TradeKill {
				player.TradeKills++
				if roundStats.TradeSpeed > 0 && roundStats.TradeSpeed < 2.0 {
					player.FastTrades++
				}
			}

			if roundStats.DeathTime > 0 && roundStats.DeathTime < 30.0 {
				player.EarlyDeaths++
			}
		}

		for _, p := range d.state.Players {
			p.RoundsPlayed++
		}

		// Update score tracking
		if winnerTeam == common.TeamTerrorists {
			// Assuming we track from T perspective initially
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

		// Log round end
		d.logger.LogRoundEnd(d.state.RoundNumber)
	})
}
