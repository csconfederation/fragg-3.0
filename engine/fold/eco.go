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
	dst.EcoHLTVRating = eco.HLTVRating
	dst.EcoHLTVCtRating = eco.CTRating
	dst.EcoHLTVTRating = eco.TRating
	dst.EcoCTEcoRating = eco.CTEcoRating
	dst.EcoTEcoRating = eco.TEcoRating
	dst.EcoSwingDisplayRating = swingDisplayRating(eco.ProbabilitySwing, eco.RoundsPlayed)
	dst.EcoPistolRoundRating = eco.PistolRoundRating
	dst.EcoAssistedKills = eco.AssistedKills
	dst.EcoClutch1v1Attempts = eco.Clutch1v1Attempts
	dst.EcoClutch1v2Attempts = eco.Clutch1v2Attempts
	dst.EcoClutch1v3Attempts = eco.Clutch1v3Attempts
	dst.EcoClutch1v4Attempts = eco.Clutch1v4Attempts
	dst.EcoClutch1v5Attempts = eco.Clutch1v5Attempts
	dst.EcoTradeKills = eco.TradeKills
	dst.EcoTradeDenials = eco.TradeDenials
	dst.EcoFastTrades = eco.FastTrades
	dst.EcoOpeningDeathsTraded = eco.OpeningDeathsTraded
	dst.EcoAWPOpeningKills = eco.AWPOpeningKills
	dst.EcoAWPMultiKillRounds = eco.AWPMultiKillRounds
	dst.EcoAWPDeaths = eco.AWPDeaths
	dst.EcoRoundsWithAWPKill = eco.RoundsWithAWPKill
	dst.EcoAWPKillsPerRound = eco.AWPKillsPerRound
	dst.EcoTimeAlivePerRound = eco.TimeAlivePerRound
	dst.EcoAvgTimeToDeath = eco.AvgTimeToDeath
	dst.EcoAvgTimeToKill = eco.AvgTimeToKill
	dst.EcoManAdvantageKills = eco.ManAdvantageKills
	dst.EcoManDisadvantageDeaths = eco.ManDisadvantageDeaths
	dst.EcoExitFrags = eco.ExitFrags
	dst.EcoKnifeKills = eco.KnifeKills
	dst.EcoPistolVsRifleKills = eco.PistolVsRifleKills
	dst.EcoLowBuyKills = eco.LowBuyKills
	dst.EcoDisadvantagedBuyKills = eco.DisadvantagedBuyKills
	dst.EcoUtilityKills = eco.UtilityKills
	dst.EcoPerfectKills = eco.PerfectKills
	dst.EcoEarlyDeaths = eco.EarlyDeaths
	dst.EcoSavedByTeammate = eco.SavedByTeammate
	dst.EcoSavedTeammate = eco.SavedTeammate
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
	dst.EcoHLTVRating = eco.TRating
	dst.EcoHLTVCtRating = 0
	dst.EcoHLTVTRating = eco.TRating
	dst.EcoCTEcoRating = 0
	dst.EcoTEcoRating = eco.TEcoRating
	dst.EcoSwingDisplayRating = swingDisplayRating(eco.TProbabilitySwing, rounds)
	dst.EcoPistolRoundRating = 0
	dst.EcoAssistedKills = eco.TAssistedKills
	dst.EcoClutch1v1Attempts = eco.TClutch1v1Attempts
	dst.EcoClutch1v2Attempts = eco.TClutch1v2Attempts
	dst.EcoClutch1v3Attempts = eco.TClutch1v3Attempts
	dst.EcoClutch1v4Attempts = eco.TClutch1v4Attempts
	dst.EcoClutch1v5Attempts = eco.TClutch1v5Attempts
	dst.EcoTradeKills = eco.TTradeKills
	dst.EcoTradeDenials = eco.TTradeDenials
	dst.EcoFastTrades = eco.TFastTrades
	dst.EcoOpeningDeathsTraded = eco.TOpeningDeathsTraded
	dst.EcoAWPOpeningKills = eco.TAWPOpeningKills
	dst.EcoAWPMultiKillRounds = eco.TAWPMultiKillRounds
	dst.EcoAWPDeaths = eco.TAWPDeaths
	dst.EcoRoundsWithAWPKill = eco.TRoundsWithAWPKill
	dst.EcoAWPKillsPerRound = perRound(float64(eco.TAWPKills), rounds)
	dst.EcoTimeAlivePerRound = 0
	dst.EcoAvgTimeToDeath = 0
	dst.EcoAvgTimeToKill = 0
	dst.EcoManAdvantageKills = eco.TManAdvantageKills
	dst.EcoManDisadvantageDeaths = eco.TManDisadvantageDeaths
	dst.EcoExitFrags = eco.TExitFrags
	dst.EcoKnifeKills = eco.TKnifeKills
	dst.EcoPistolVsRifleKills = eco.TPistolVsRifleKills
	dst.EcoLowBuyKills = 0
	dst.EcoDisadvantagedBuyKills = 0
	dst.EcoUtilityKills = eco.TUtilityKills
	dst.EcoPerfectKills = 0
	dst.EcoEarlyDeaths = eco.TEarlyDeaths
	dst.EcoSavedByTeammate = eco.TSavedByTeammate
	dst.EcoSavedTeammate = eco.TSavedTeammate
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
	dst.EcoHLTVRating = eco.CTRating
	dst.EcoHLTVCtRating = eco.CTRating
	dst.EcoHLTVTRating = 0
	dst.EcoCTEcoRating = eco.CTEcoRating
	dst.EcoTEcoRating = 0
	dst.EcoSwingDisplayRating = swingDisplayRating(eco.CTProbabilitySwing, rounds)
	dst.EcoPistolRoundRating = 0
	dst.EcoAssistedKills = eco.CTAssistedKills
	dst.EcoClutch1v1Attempts = eco.CTClutch1v1Attempts
	dst.EcoClutch1v2Attempts = eco.CTClutch1v2Attempts
	dst.EcoClutch1v3Attempts = eco.CTClutch1v3Attempts
	dst.EcoClutch1v4Attempts = eco.CTClutch1v4Attempts
	dst.EcoClutch1v5Attempts = eco.CTClutch1v5Attempts
	dst.EcoTradeKills = eco.CTTradeKills
	dst.EcoTradeDenials = eco.CTTradeDenials
	dst.EcoFastTrades = eco.CTFastTrades
	dst.EcoOpeningDeathsTraded = eco.CTOpeningDeathsTraded
	dst.EcoAWPOpeningKills = eco.CTAWPOpeningKills
	dst.EcoAWPMultiKillRounds = eco.CTAWPMultiKillRounds
	dst.EcoAWPDeaths = eco.CTAWPDeaths
	dst.EcoRoundsWithAWPKill = eco.CTRoundsWithAWPKill
	dst.EcoAWPKillsPerRound = perRound(float64(eco.CTAWPKills), rounds)
	dst.EcoTimeAlivePerRound = 0
	dst.EcoAvgTimeToDeath = 0
	dst.EcoAvgTimeToKill = 0
	dst.EcoManAdvantageKills = eco.CTManAdvantageKills
	dst.EcoManDisadvantageDeaths = eco.CTManDisadvantageDeaths
	dst.EcoExitFrags = eco.CTExitFrags
	dst.EcoKnifeKills = eco.CTKnifeKills
	dst.EcoPistolVsRifleKills = eco.CTPistolVsRifleKills
	dst.EcoLowBuyKills = 0
	dst.EcoDisadvantagedBuyKills = 0
	dst.EcoUtilityKills = eco.CTUtilityKills
	dst.EcoPerfectKills = 0
	dst.EcoEarlyDeaths = eco.CTEarlyDeaths
	dst.EcoSavedByTeammate = eco.CTSavedByTeammate
	dst.EcoSavedTeammate = eco.CTSavedTeammate
}
