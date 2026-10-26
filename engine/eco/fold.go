package eco

import (
	"github.com/csconfederation/fragg-3.0/model"
	"github.com/csconfederation/fragg-3.0/rating"
)

// FoldRound adds one round's stats to the player's match totals.
func FoldRound(player *model.PlayerStats, round *model.RoundStats) {
	player.Kills += round.Kills
	player.Damage += round.Damage
	player.EcoKillValue += round.EconImpact
	player.EcoDeathValue += round.EcoDeathValue
	player.TotalTimeToKill += round.TimeToKillSum
	player.KillsWithTTK += round.KillsWithTTK
	player.LowBuyKills += round.LowBuyKills
	player.DisadvantagedBuyKills += round.DisadvantagedBuyKills
	player.ManAdvantageKills += round.ManAdvantageKills
	player.ManDisadvantageDeaths += round.ManDisadvantageDeaths
	player.EcoAdjustedKills += round.EcoAdjustedKills
	player.PerfectKills += round.PerfectKills
	player.TradeDenials += round.TradeDenials
	player.AWPKills += round.AWPKills

	if round.Deaths > 0 {
		player.Deaths++
		player.TotalTimeAlive += round.DeathTime
		player.TotalDeathTime += round.DeathTime
		player.DeathTimeRounds++
	} else if round.Survived {
		player.Survival++
		player.TotalTimeAlive += round.TimeAlive
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

	if round.ClutchAttempt {
		switch round.ClutchSize {
		case 1:
			player.Clutch1v1Attempts++
		case 2:
			player.Clutch1v2Attempts++
		case 3:
			player.Clutch1v3Attempts++
		case 4:
			player.Clutch1v4Attempts++
		case 5:
			player.Clutch1v5Attempts++
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

	player.ProbabilitySwing += round.ProbabilitySwing
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
	}

	if p.KillsWithTTK > 0 {
		p.AvgTimeToKill = p.TotalTimeToKill / float64(p.KillsWithTTK)
	}

	// Calculate Average Time to Death (ATD)
	if p.DeathTimeRounds > 0 {
		p.AvgTimeToDeath = p.TotalDeathTime / float64(p.DeathTimeRounds)
	}

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

	p.FinalRating = rating.ComputeFinalRating(p, kdprModifier)

	if p.TRoundsPlayed > 0 {
		p.TEcoRating = rating.ComputeSideRating(
			p.TRoundsPlayed, p.TKills, p.TDeaths, p.TDamage, p.TEcoKillValue,
			p.TProbabilitySwing, p.TKAST, p.TMultiKills, p.TClutchRounds, p.TClutchWins, kdprModifier)
	}
	if p.CTRoundsPlayed > 0 {
		p.CTEcoRating = rating.ComputeSideRating(
			p.CTRoundsPlayed, p.CTKills, p.CTDeaths, p.CTDamage, p.CTEcoKillValue,
			p.CTProbabilitySwing, p.CTKAST, p.CTMultiKills, p.CTClutchRounds, p.CTClutchWins, kdprModifier)
	}
}
