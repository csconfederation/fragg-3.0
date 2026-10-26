package fold

import (
	"github.com/csconfederation/fragg-3.0/engine/cscstats"
	"github.com/csconfederation/fragg-3.0/model"
)

// mergeEco copies the eco metrics onto the CSC rows, per scope.
func mergeEco(g *cscstats.Game, players map[uint64]*model.PlayerStats) {
	mergeInto(g.TotalPlayerStats, players, applyTotalEcoStats)
	mergeInto(g.TPlayerStats, players, applyTEcoStats)
	mergeInto(g.CtPlayerStats, players, applyCTEcoStats)
}

type ecoApplyFunc func(dst *cscstats.PlayerStats, eco *model.PlayerStats)

// mergeInto copies eco metrics onto every CSC player present in dst.
func mergeInto(dst map[uint64]*cscstats.PlayerStats, eco map[uint64]*model.PlayerStats, apply ecoApplyFunc) {
	for steamID, cscPlayer := range dst {
		ecoPlayer, ok := eco[steamID]
		if !ok || ecoPlayer == nil || cscPlayer == nil {
			continue
		}
		apply(cscPlayer, ecoPlayer)
	}
}

func perRound(total float64, rounds int) float64 {
	if rounds <= 0 {
		return 0
	}
	return total / float64(rounds)
}

// swingDisplayRating scales probability swing into the display rating used by
// ecoSwingDisplayRating (0% = 1.0, +4% = 1.4, -3% = 0.7), clamped to [0.5, 1.5].
func swingDisplayRating(probSwing float64, rounds int) float64 {
	if rounds <= 0 {
		return 0
	}
	rating := 1.0 + (probSwing/float64(rounds))*10.0
	if rating < 0.5 {
		return 0.5
	}
	if rating > 1.5 {
		return 1.5
	}
	return rating
}

// applyTotalEcoStats maps whole-match eco fields onto TotalPlayerStats.
func applyTotalEcoStats(dst *cscstats.PlayerStats, eco *model.PlayerStats) {
	dst.SwingRating = eco.FinalRating
	dst.EcoProbabilitySwing = eco.ProbabilitySwing
	dst.EcoProbabilitySwingPerRound = eco.ProbabilitySwingPerRound
	dst.EcoTProbabilitySwing = eco.TProbabilitySwing
	dst.EcoCTProbabilitySwing = eco.CTProbabilitySwing
	dst.EcoKillValue = eco.EcoKillValue
	dst.EcoDeathValue = eco.EcoDeathValue
	dst.EcoDuelSwing = eco.DuelSwing
	dst.EcoDuelSwingPerRound = eco.DuelSwingPerRound
	dst.EcoAdjustedKills = eco.EcoAdjustedKills
	dst.HLTVRating = eco.HLTVRating
	dst.HLTVCtRating = eco.CTRating
	dst.HLTVTRating = eco.TRating
	dst.EcoCTEcoRating = eco.CTEcoRating
	dst.EcoTEcoRating = eco.TEcoRating
	dst.EcoSwingDisplayRating = swingDisplayRating(eco.ProbabilitySwing, eco.RoundsPlayed)
	dst.AssistedKills = eco.AssistedKills
	dst.Clutch1v1Attempts = eco.Clutch1v1Attempts
	dst.Clutch1v2Attempts = eco.Clutch1v2Attempts
	dst.Clutch1v3Attempts = eco.Clutch1v3Attempts
	dst.Clutch1v4Attempts = eco.Clutch1v4Attempts
	dst.Clutch1v5Attempts = eco.Clutch1v5Attempts
	dst.TradeKills = eco.TradeKills
	dst.TradeDenials = eco.TradeDenials
	dst.FastTrades = eco.FastTrades
	dst.OpeningDeathsTraded = eco.OpeningDeathsTraded
	dst.AWPOpeningKills = eco.AWPOpeningKills
	dst.AWPMultiKillRounds = eco.AWPMultiKillRounds
	dst.AWPDeaths = eco.AWPDeaths
	dst.RoundsWithAWPKill = eco.RoundsWithAWPKill
	dst.AWPKillsPerRound = eco.AWPKillsPerRound
	dst.TimeAlivePerRound = eco.TimeAlivePerRound
	dst.AvgTimeToDeath = eco.AvgTimeToDeath
	dst.AvgTimeToKill = eco.AvgTimeToKill
	dst.ManAdvantageKills = eco.ManAdvantageKills
	dst.ManDisadvantageDeaths = eco.ManDisadvantageDeaths
	dst.ExitFrags = eco.ExitFrags
	dst.KnifeKills = eco.KnifeKills
	dst.PistolVsRifleKills = eco.PistolVsRifleKills
	dst.KillsVsLowerBuy = eco.LowBuyKills
	dst.KillsVsMuchLowerBuy = eco.DisadvantagedBuyKills
	dst.UtilityKills = eco.UtilityKills
	dst.PerfectKills = eco.PerfectKills
	dst.EarlyDeaths = eco.EarlyDeaths
	dst.SavedByTeammate = eco.SavedByTeammate
	dst.SavedTeammate = eco.SavedTeammate
}

// applyTEcoStats maps T-side eco fields onto TPlayerStats.
func applyTEcoStats(dst *cscstats.PlayerStats, eco *model.PlayerStats) {
	rounds := eco.TRoundsPlayed
	duel := eco.TEcoKillValue - eco.TEcoDeathValue
	dst.SwingRating = eco.TEcoRating
	dst.EcoProbabilitySwing = eco.TProbabilitySwing
	dst.EcoProbabilitySwingPerRound = perRound(eco.TProbabilitySwing, rounds)
	dst.EcoTProbabilitySwing = eco.TProbabilitySwing
	dst.EcoCTProbabilitySwing = 0
	dst.EcoKillValue = eco.TEcoKillValue
	dst.EcoDeathValue = eco.TEcoDeathValue
	dst.EcoDuelSwing = duel
	dst.EcoDuelSwingPerRound = perRound(duel, rounds)
	dst.EcoAdjustedKills = 0
	dst.HLTVRating = eco.TRating
	dst.HLTVCtRating = 0
	dst.HLTVTRating = eco.TRating
	dst.EcoCTEcoRating = 0
	dst.EcoTEcoRating = eco.TEcoRating
	dst.EcoSwingDisplayRating = swingDisplayRating(eco.TProbabilitySwing, rounds)
	dst.AssistedKills = eco.TAssistedKills
	dst.Clutch1v1Attempts = eco.TClutch1v1Attempts
	dst.Clutch1v2Attempts = eco.TClutch1v2Attempts
	dst.Clutch1v3Attempts = eco.TClutch1v3Attempts
	dst.Clutch1v4Attempts = eco.TClutch1v4Attempts
	dst.Clutch1v5Attempts = eco.TClutch1v5Attempts
	dst.TradeKills = eco.TTradeKills
	dst.TradeDenials = eco.TTradeDenials
	dst.FastTrades = eco.TFastTrades
	dst.OpeningDeathsTraded = eco.TOpeningDeathsTraded
	dst.AWPOpeningKills = eco.TAWPOpeningKills
	dst.AWPMultiKillRounds = eco.TAWPMultiKillRounds
	dst.AWPDeaths = eco.TAWPDeaths
	dst.RoundsWithAWPKill = eco.TRoundsWithAWPKill
	dst.AWPKillsPerRound = perRound(float64(eco.TAWPKills), rounds)
	dst.TimeAlivePerRound = 0
	dst.AvgTimeToDeath = 0
	dst.AvgTimeToKill = 0
	dst.ManAdvantageKills = eco.TManAdvantageKills
	dst.ManDisadvantageDeaths = eco.TManDisadvantageDeaths
	dst.ExitFrags = eco.TExitFrags
	dst.KnifeKills = eco.TKnifeKills
	dst.PistolVsRifleKills = eco.TPistolVsRifleKills
	dst.KillsVsLowerBuy = 0
	dst.KillsVsMuchLowerBuy = 0
	dst.UtilityKills = eco.TUtilityKills
	dst.PerfectKills = 0
	dst.EarlyDeaths = eco.TEarlyDeaths
	dst.SavedByTeammate = eco.TSavedByTeammate
	dst.SavedTeammate = eco.TSavedTeammate
}

// applyCTEcoStats maps CT-side eco fields onto CtPlayerStats.
func applyCTEcoStats(dst *cscstats.PlayerStats, eco *model.PlayerStats) {
	rounds := eco.CTRoundsPlayed
	duel := eco.CTEcoKillValue - eco.CTEcoDeathValue
	dst.SwingRating = eco.CTEcoRating
	dst.EcoProbabilitySwing = eco.CTProbabilitySwing
	dst.EcoProbabilitySwingPerRound = perRound(eco.CTProbabilitySwing, rounds)
	dst.EcoTProbabilitySwing = 0
	dst.EcoCTProbabilitySwing = eco.CTProbabilitySwing
	dst.EcoKillValue = eco.CTEcoKillValue
	dst.EcoDeathValue = eco.CTEcoDeathValue
	dst.EcoDuelSwing = duel
	dst.EcoDuelSwingPerRound = perRound(duel, rounds)
	dst.EcoAdjustedKills = 0
	dst.HLTVRating = eco.CTRating
	dst.HLTVCtRating = eco.CTRating
	dst.HLTVTRating = 0
	dst.EcoCTEcoRating = eco.CTEcoRating
	dst.EcoTEcoRating = 0
	dst.EcoSwingDisplayRating = swingDisplayRating(eco.CTProbabilitySwing, rounds)
	dst.AssistedKills = eco.CTAssistedKills
	dst.Clutch1v1Attempts = eco.CTClutch1v1Attempts
	dst.Clutch1v2Attempts = eco.CTClutch1v2Attempts
	dst.Clutch1v3Attempts = eco.CTClutch1v3Attempts
	dst.Clutch1v4Attempts = eco.CTClutch1v4Attempts
	dst.Clutch1v5Attempts = eco.CTClutch1v5Attempts
	dst.TradeKills = eco.CTTradeKills
	dst.TradeDenials = eco.CTTradeDenials
	dst.FastTrades = eco.CTFastTrades
	dst.OpeningDeathsTraded = eco.CTOpeningDeathsTraded
	dst.AWPOpeningKills = eco.CTAWPOpeningKills
	dst.AWPMultiKillRounds = eco.CTAWPMultiKillRounds
	dst.AWPDeaths = eco.CTAWPDeaths
	dst.RoundsWithAWPKill = eco.CTRoundsWithAWPKill
	dst.AWPKillsPerRound = perRound(float64(eco.CTAWPKills), rounds)
	dst.TimeAlivePerRound = 0
	dst.AvgTimeToDeath = 0
	dst.AvgTimeToKill = 0
	dst.ManAdvantageKills = eco.CTManAdvantageKills
	dst.ManDisadvantageDeaths = eco.CTManDisadvantageDeaths
	dst.ExitFrags = eco.CTExitFrags
	dst.KnifeKills = eco.CTKnifeKills
	dst.PistolVsRifleKills = eco.CTPistolVsRifleKills
	dst.KillsVsLowerBuy = 0
	dst.KillsVsMuchLowerBuy = 0
	dst.UtilityKills = eco.CTUtilityKills
	dst.PerfectKills = 0
	dst.EarlyDeaths = eco.CTEarlyDeaths
	dst.SavedByTeammate = eco.CTSavedByTeammate
	dst.SavedTeammate = eco.CTSavedTeammate
}
