package eco

import (
	"github.com/csconfederation/fragg-3.0/model"
	"github.com/csconfederation/fragg-3.0/rating"
)

// FoldRound adds one round's stats to the player's match totals.
func FoldRound(player *model.PlayerStats, round *model.RoundStats, roundNumber int) {
	player.Kills += round.Kills
	player.Assists += round.Assists
	player.Damage += round.Damage
	player.DamageTaken += round.DamageTaken
	player.EcoKillValue += round.EconImpact
	player.EconImpact += round.EconImpact
	player.EcoDeathValue += round.EcoDeathValue
	player.Headshots += round.Headshots
	player.TotalTimeToKill += round.TimeToKillSum
	player.KillsWithTTK += round.KillsWithTTK
	player.LowBuyKills += round.LowBuyKills
	player.DisadvantagedBuyKills += round.DisadvantagedBuyKills
	player.ManAdvantageKills += round.ManAdvantageKills
	player.ManDisadvantageDeaths += round.ManDisadvantageDeaths
	player.EcoAdjustedKills += round.EcoAdjustedKills
	player.PerfectKills += round.PerfectKills
	player.EnemiesFlashed += round.EnemiesFlashed
	player.TradeDenials += round.TradeDenials
	player.AWPKills += round.AWPKills
	player.HEDamage += round.HEDamage
	player.FireDamage += round.FireDamage
	player.SmokesThrown += round.SmokesThrown
	player.HEsThrown += round.HEsThrown
	player.MolotovsThrown += round.MolotovsThrown
	player.TotalNadesThrown += round.SmokesThrown + round.HEsThrown + round.MolotovsThrown + round.FlashesThrown

	if round.Deaths > 0 {
		player.Deaths++
		player.TotalTimeAlive += round.DeathTime
		player.TotalDeathTime += round.DeathTime
		player.DeathTimeRounds++
	} else if round.Survived {
		player.Survival++
		player.TotalTimeAlive += round.TimeAlive
	}

	if round.TeamWon {
		player.RoundsWon++
	} else {
		player.RoundsLost++
		if round.Survived {
			player.SavesOnLoss++
		}
	}

	if round.WasLastAlive {
		player.LastAliveRounds++
	}

	if round.OpeningKill {
		player.OpeningKills++
		player.OpeningAttempts++
		player.OpeningSuccesses++
		if round.PlayerSide == "T" {
			player.TOpeningKills++
		} else if round.PlayerSide == "CT" {
			player.CTOpeningKills++
		}
	}
	if round.OpeningDeath {
		player.OpeningDeaths++
		player.OpeningAttempts++
		if round.PlayerSide == "T" {
			player.TOpeningDeaths++
		} else if round.PlayerSide == "CT" {
			player.CTOpeningDeaths++
		}
	}

	if round.Traded {
		player.TradedDeaths++
	}
	if round.OpeningDeath && round.Traded {
		player.OpeningDeathsTraded++
	}
	if round.SavedTeammate {
		player.SavedTeammate++
	}
	if round.AWPOpeningKill {
		player.AWPOpeningKills++
	}
	player.UtilityKills += round.UtilityKills

	if round.Kills >= 1 && round.Kills <= 5 {
		player.MultiKillsRaw[round.Kills]++
	}

	if round.ClutchEnteredSize > 0 || round.ClutchAttempt {
		size := round.ClutchEnteredSize
		if size == 0 {
			size = round.ClutchSize
		}
		player.ClutchRounds++
		switch size {
		case 1:
			player.Clutch1v1Attempts++
			if round.TeamWon {
				player.Clutch1v1Wins++
			}
		case 2:
			player.Clutch1v2Attempts++
			if round.TeamWon {
				player.Clutch1v2Wins++
			}
		case 3:
			player.Clutch1v3Attempts++
			if round.TeamWon {
				player.Clutch1v3Wins++
			}
		case 4:
			player.Clutch1v4Attempts++
			if round.TeamWon {
				player.Clutch1v4Wins++
			}
		case 5:
			player.Clutch1v5Attempts++
			if round.TeamWon {
				player.Clutch1v5Wins++
			}
		}
		if round.TeamWon {
			player.ClutchWins++
		}
	}

	if round.PlayerSide == "T" {
		player.TEcoDeathValue += round.EcoDeathValue
		player.TManAdvantageKills += round.ManAdvantageKills
		player.TManDisadvantageDeaths += round.ManDisadvantageDeaths
	} else if round.PlayerSide == "CT" {
		player.CTEcoDeathValue += round.EcoDeathValue
		player.CTManAdvantageKills += round.ManAdvantageKills
		player.CTManDisadvantageDeaths += round.ManDisadvantageDeaths
	}

	round.MultiKillRound = round.Kills
	player.ProbabilitySwing += round.ProbabilitySwing
	player.RoundBreakdowns = append(player.RoundBreakdowns, model.NewRoundSwingBreakdown(roundNumber, round))
	if round.PlayerSide == "T" {
		player.TProbabilitySwing += round.ProbabilitySwing
	} else if round.PlayerSide == "CT" {
		player.CTProbabilitySwing += round.ProbabilitySwing
	}
}

// Derive computes a player's rates and ratings from the folded totals.
func Derive(p *model.PlayerStats, kdprModifier bool) {
	if p.RoundsPlayed > 0 {
		rounds := float64(p.RoundsPlayed)
		p.ADR = float64(p.Damage) / rounds
		p.KPR = float64(p.Kills) / rounds
		p.DPR = float64(p.Deaths) / rounds
		p.KAST = p.KAST / rounds
		p.Survival = p.Survival / rounds

		p.AWPKillsPerRound = float64(p.AWPKills) / rounds

		// Calculate HLTV rating using centralized function
		survivals := int(p.Survival * rounds)
		p.HLTVRating = rating.ComputeHLTVRating(rating.HLTVInput{
			RoundsPlayed: p.RoundsPlayed,
			Kills:        p.Kills,
			Deaths:       p.Deaths,
			Survivals:    survivals,
			MultiKills:   p.MultiKillsRaw,
		})

		// Pistol round rating
		if p.PistolRoundsPlayed > 0 {
			p.PistolRoundRating = rating.ComputePistolRoundRating(
				p.PistolRoundsPlayed, p.PistolRoundKills, p.PistolRoundDeaths,
				p.PistolRoundSurvivals, p.PistolRoundMultiKills)
		}

		// Side-specific HLTV ratings
		if p.TRoundsPlayed > 0 {
			p.TRating = rating.ComputeSideHLTVRating(
				p.TRoundsPlayed, p.TKills, p.TDeaths, p.TSurvivals, p.TMultiKills)
		}

		if p.CTRoundsPlayed > 0 {
			p.CTRating = rating.ComputeSideHLTVRating(
				p.CTRoundsPlayed, p.CTKills, p.CTDeaths, p.CTSurvivals, p.CTMultiKills)
		}

		p.TimeAlivePerRound = p.TotalTimeAlive / rounds
		p.EnemyFlashDurationPerRound = p.EnemyFlashDuration / rounds
		p.TeamFlashDurationPerRound = p.TeamFlashDuration / rounds
		p.RoundsWithKillPct = float64(p.RoundsWithKill) / rounds
		p.RoundsWithMultiKillPct = float64(p.RoundsWithMultiKill) / rounds
		p.SavedByTeammatePerRound = float64(p.SavedByTeammate) / rounds
		p.TradedDeathsPerRound = float64(p.TradedDeaths) / rounds
		p.AssistsPerRound = float64(p.Assists) / rounds
		p.SupportRoundsPct = float64(p.SupportRounds) / rounds
		p.SavedTeammatePerRound = float64(p.SavedTeammate) / rounds
		p.TradeKillsPerRound = float64(p.TradeKills) / rounds
		p.OpeningKillsPerRound = float64(p.OpeningKills) / rounds
		p.OpeningDeathsPerRound = float64(p.OpeningDeaths) / rounds
		p.OpeningAttemptsPct = float64(p.OpeningAttempts) / rounds
		p.AttacksPerRound = float64(p.AttackRounds) / rounds
		p.ClutchPointsPerRound = float64(p.ClutchWins) / rounds
		p.LastAlivePct = float64(p.LastAliveRounds) / rounds
		p.RoundsWithAWPKillPct = float64(p.RoundsWithAWPKill) / rounds
		p.AWPMultiKillRoundsPerRound = float64(p.AWPMultiKillRounds) / rounds
		p.AWPOpeningKillsPerRound = float64(p.AWPOpeningKills) / rounds
		p.UtilityDamagePerRound = float64(p.UtilityDamage) / rounds
		p.UtilityKillsPer100Rounds = float64(p.UtilityKills) * 100 / rounds
		p.FlashesThrownPerRound = float64(p.FlashesThrown) / rounds
		p.FlashAssistsPerRound = float64(p.FlashAssists) / rounds
	}

	if p.RoundsWon > 0 {
		p.KillsPerRoundWin = float64(p.KillsInWonRounds) / float64(p.RoundsWon)
		p.DamagePerRoundWin = float64(p.DamageInWonRounds) / float64(p.RoundsWon)
	}

	if p.RoundsLost > 0 {
		p.SavesPerRoundLoss = float64(p.SavesOnLoss) / float64(p.RoundsLost)
	}

	if p.Deaths > 0 {
		p.TradedDeathsPct = float64(p.TradedDeaths) / float64(p.Deaths)
	}

	if p.OpeningDeaths > 0 {
		p.OpeningDeathsTradedPct = float64(p.OpeningDeathsTraded) / float64(p.OpeningDeaths)
	}

	if p.Kills > 0 {
		p.TradeKillsPct = float64(p.TradeKills) / float64(p.Kills)
		p.AssistedKillsPct = float64(p.AssistedKills) / float64(p.Kills)
		p.DamagePerKill = float64(p.Damage) / float64(p.Kills)
		p.AWPKillsPct = float64(p.AWPKills) / float64(p.Kills)
		p.LowBuyKillsPct = float64(p.LowBuyKills) / float64(p.Kills)
		p.DisadvantagedBuyKillsPct = float64(p.DisadvantagedBuyKills) / float64(p.Kills)
		p.HeadshotPct = float64(p.Headshots) / float64(p.Kills)
		p.ManAdvantageKillsPct = float64(p.ManAdvantageKills) / float64(p.Kills)
	}

	if p.Deaths > 0 {
		p.ManDisadvantageDeathsPct = float64(p.ManDisadvantageDeaths) / float64(p.Deaths)
	}

	if p.KillsWithTTK > 0 {
		p.AvgTimeToKill = p.TotalTimeToKill / float64(p.KillsWithTTK)
	}

	if p.OpeningAttempts > 0 {
		p.OpeningSuccessPct = float64(p.OpeningSuccesses) / float64(p.OpeningAttempts)
	}

	if p.OpeningKills > 0 {
		p.WinPctAfterOpeningKill = float64(p.RoundsWonAfterOpening) / float64(p.OpeningKills)
	}

	if p.Clutch1v1Attempts > 0 {
		p.Clutch1v1WinPct = float64(p.Clutch1v1Wins) / float64(p.Clutch1v1Attempts)
	}

	// Calculate Average Time to Death (ATD)
	if p.DeathTimeRounds > 0 {
		p.AvgTimeToDeath = p.TotalDeathTime / float64(p.DeathTimeRounds)
	}

	// Calculate DamagePerRound (same as ADR but explicit field)
	if p.RoundsPlayed > 0 {
		p.DamagePerRound = float64(p.Damage) / float64(p.RoundsPlayed)
	}

	p.MultiKills.OneK = p.MultiKillsRaw[1]
	p.MultiKills.TwoK = p.MultiKillsRaw[2]
	p.MultiKills.ThreeK = p.MultiKillsRaw[3]
	p.MultiKills.FourK = p.MultiKillsRaw[4]
	p.MultiKills.FiveK = p.MultiKillsRaw[5]

	// Compute probability-based swing metrics
	if p.RoundsPlayed > 0 {
		rounds := float64(p.RoundsPlayed)
		p.ProbabilitySwingPerRound = p.ProbabilitySwing / rounds
		// DuelSwing: EcoKillValue - EcoDeathValue (net duel economy impact)
		p.DuelSwing = p.EcoKillValue - p.EcoDeathValue
		p.DuelSwingPerRound = p.DuelSwing / rounds
		// SwingRating: scale swing to rating (0% = 1.0, +4% = 1.4, -3% = 0.7)
		p.SwingRating = 1.0 + (p.ProbabilitySwingPerRound * 10.0)
		if p.SwingRating < 0.5 {
			p.SwingRating = 0.5
		} else if p.SwingRating > 1.5 {
			p.SwingRating = 1.5
		}
	}

	if p.TRoundsPlayed > 0 {
		tRounds := float64(p.TRoundsPlayed)
		p.TDuelSwing = p.TEcoKillValue - p.TEcoDeathValue
		p.TDuelSwingPerRound = p.TDuelSwing / tRounds
	}
	if p.CTRoundsPlayed > 0 {
		ctRounds := float64(p.CTRoundsPlayed)
		p.CTDuelSwing = p.CTEcoKillValue - p.CTEcoDeathValue
		p.CTDuelSwingPerRound = p.CTDuelSwing / ctRounds
	}

	p.FinalRating = rating.ComputeFinalRating(p, kdprModifier)

	if p.TRoundsPlayed > 0 {
		p.TEcoRating = rating.ComputeSideRating(
			p.TRoundsPlayed, p.TKills, p.TDeaths, p.TDamage, p.TEcoKillValue,
			p.TProbabilitySwing, p.TKAST, p.TMultiKills, p.TClutchRounds, p.TClutchWins, kdprModifier)
	}
	if p.TKills > 0 {
		p.TManAdvantageKillsPct = float64(p.TManAdvantageKills) / float64(p.TKills)
	}
	if p.TDeaths > 0 {
		p.TManDisadvantageDeathsPct = float64(p.TManDisadvantageDeaths) / float64(p.TDeaths)
	}
	if p.CTRoundsPlayed > 0 {
		p.CTEcoRating = rating.ComputeSideRating(
			p.CTRoundsPlayed, p.CTKills, p.CTDeaths, p.CTDamage, p.CTEcoKillValue,
			p.CTProbabilitySwing, p.CTKAST, p.CTMultiKills, p.CTClutchRounds, p.CTClutchWins, kdprModifier)
	}
	if p.CTKills > 0 {
		p.CTManAdvantageKillsPct = float64(p.CTManAdvantageKills) / float64(p.CTKills)
	}
	if p.CTDeaths > 0 {
		p.CTManDisadvantageDeathsPct = float64(p.CTManDisadvantageDeaths) / float64(p.CTDeaths)
	}
}
