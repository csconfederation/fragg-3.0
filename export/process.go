package export

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/csconfederation/fragg-3.0/csc"
	"github.com/csconfederation/fragg-3.0/model"
	"github.com/csconfederation/fragg-3.0/parser"
)

// Game is the demoScrape2-compatible output type returned by ProcessDemo.
// It is the ported CSC game augmented with eco-rating's additive metrics.
type Game = csc.Game

// ErrNoValidRounds is returned (joined into the error) when a demo produced no
// valid, complete rounds. It mirrors demoScrape2 v0.3.2 so the demo worker can
// classify an unprocessable-but-not-broken demo as HTTP 422.
var ErrNoValidRounds = csc.ErrNoValidRounds

// ProcessDemo is the drop-in replacement for demoscrape2.ProcessDemo.
//
// It runs two independent pipelines over the same demo:
//  1. The ported CSC pipeline (package csc), producing demoScrape2-identical
//     fields under their original JSON names.
//  2. eco-rating's own parser, producing the new probability-swing metrics.
//
// eco-rating's metrics are merged onto the CSC game additively (never
// overwriting a CSC field); where the two parsers compute the same concept
// differently, the eco value is carried under a distinct eco* name. The eco
// primary rating is surfaced as swing_rating.
//
// The error contract matches demoScrape2: a non-nil *Game is always returned,
// ErrNoValidRounds is joined when there are no rounds, and an unexpected EOF on
// an already-finished match leaves Game.Result == "Ended".
func ProcessDemo(demo io.ReadCloser) (game *Game, err error) {
	defer func() {
		if r := recover(); r != nil {
			// The ported CSC pipeline can panic on pathological demos (matching
			// demoScrape2). Convert that into an error so the worker returns a
			// clean 5xx instead of crashing the process.
			if game == nil {
				game = csc.InitGameObject()
			}
			err = fmt.Errorf("panic during demo processing: %v", r)
		}
	}()

	// Buffer the demo to a temp file so it can be parsed twice without holding
	// the whole (often hundreds of MB) stream in memory.
	tmp, terr := os.CreateTemp("", "ecorating-demo-*.dem")
	if terr != nil {
		return csc.InitGameObject(), fmt.Errorf("failed to create temp demo file: %w", terr)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, cerr := io.Copy(tmp, demo); cerr != nil {
		tmp.Close()
		demo.Close()
		return csc.InitGameObject(), fmt.Errorf("failed to buffer demo: %w", cerr)
	}
	tmp.Close()
	demo.Close()

	// Pipeline 1: CSC (authoritative for the worker contract).
	cscFile, oerr := os.Open(tmpPath)
	if oerr != nil {
		return csc.InitGameObject(), fmt.Errorf("failed to reopen demo: %w", oerr)
	}
	game, err = csc.ProcessDemo(cscFile)
	if game == nil {
		game = csc.InitGameObject()
	}

	// Pipeline 2: eco-rating (additive). Failures here must not change the CSC
	// result the worker relies on; they are logged and EcoStatsOK stays false.
	ecoFile, oerr := os.Open(tmpPath)
	if oerr != nil {
		log.Printf("export: eco merge skipped: reopen demo: %v", oerr)
	} else {
		mergeEcoStats(game, ecoFile)
		ecoFile.Close()
	}

	return game, err
}

// mergeEcoStats parses the demo with eco-rating's pipeline and copies eco-only
// metrics onto the CSC game's player maps, matched by SteamID.
func mergeEcoStats(game *csc.Game, r io.ReadCloser) {
	defer func() {
		if rec := recover(); rec != nil {
			log.Printf("export: eco merge panic recovered: %v", rec)
			game.EcoStatsOK = false
		}
	}()

	p := parser.NewDemoParser(r)
	if perr := p.Parse(); perr != nil {
		log.Printf("export: eco parse failed: %v", perr)
		return
	}

	players := p.GetPlayers()
	if game.MapName == "" {
		game.MapName = p.GetMapName()
	}

	mergeInto(game.TotalPlayerStats, players, applyTotalEcoStats)
	mergeInto(game.TPlayerStats, players, applyTEcoStats)
	mergeInto(game.CtPlayerStats, players, applyCTEcoStats)
	game.EcoStatsOK = true
}

type ecoApplyFunc func(dst *csc.PlayerStats, eco *model.PlayerStats)

// mergeInto copies eco metrics onto every CSC player present in dst.
func mergeInto(dst map[uint64]*csc.PlayerStats, eco map[uint64]*model.PlayerStats, apply ecoApplyFunc) {
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

// applyTotalEcoStats maps whole-match eco fields onto TotalPlayerStats.
func applyTotalEcoStats(dst *csc.PlayerStats, eco *model.PlayerStats) {
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
	dst.EcoSwingDisplayRating = eco.SwingRating
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
func applyTEcoStats(dst *csc.PlayerStats, eco *model.PlayerStats) {
	rounds := eco.TRoundsPlayed
	dst.SwingRating = eco.TEcoRating
	dst.EcoProbabilitySwing = eco.TProbabilitySwing
	dst.EcoProbabilitySwingPerRound = perRound(eco.TProbabilitySwing, rounds)
	dst.EcoTProbabilitySwing = eco.TProbabilitySwing
	dst.EcoCTProbabilitySwing = 0
	dst.EcoKillValue = eco.TEcoKillValue
	dst.EcoDeathValue = 0
	dst.EcoDuelSwing = eco.TProbabilitySwing
	dst.EcoDuelSwingPerRound = perRound(eco.TProbabilitySwing, rounds)
	dst.EcoAdjustedKills = 0
	dst.EcoHLTVRating = eco.TRating
	dst.EcoHLTVCtRating = 0
	dst.EcoHLTVTRating = eco.TRating
	dst.EcoCTEcoRating = 0
	dst.EcoTEcoRating = eco.TEcoRating
	dst.EcoSwingDisplayRating = eco.TEcoRating
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
func applyCTEcoStats(dst *csc.PlayerStats, eco *model.PlayerStats) {
	rounds := eco.CTRoundsPlayed
	dst.SwingRating = eco.CTEcoRating
	dst.EcoProbabilitySwing = eco.CTProbabilitySwing
	dst.EcoProbabilitySwingPerRound = perRound(eco.CTProbabilitySwing, rounds)
	dst.EcoTProbabilitySwing = 0
	dst.EcoCTProbabilitySwing = eco.CTProbabilitySwing
	dst.EcoKillValue = eco.CTEcoKillValue
	dst.EcoDeathValue = 0
	dst.EcoDuelSwing = eco.CTProbabilitySwing
	dst.EcoDuelSwingPerRound = perRound(eco.CTProbabilitySwing, rounds)
	dst.EcoAdjustedKills = 0
	dst.EcoHLTVRating = eco.CTRating
	dst.EcoHLTVCtRating = eco.CTRating
	dst.EcoHLTVTRating = 0
	dst.EcoCTEcoRating = eco.CTEcoRating
	dst.EcoTEcoRating = 0
	dst.EcoSwingDisplayRating = eco.CTEcoRating
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
