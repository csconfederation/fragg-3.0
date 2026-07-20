package export

import (
	"fmt"
	"io"
	"os"

	"github.com/ethsmith/eco-rating/csc"
	"github.com/ethsmith/eco-rating/model"
	"github.com/ethsmith/eco-rating/parser"
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
	// result the worker relies on, so they are swallowed after logging.
	if ecoFile, oerr := os.Open(tmpPath); oerr == nil {
		mergeEcoStats(game, ecoFile)
		ecoFile.Close()
	}

	return game, err
}

// mergeEcoStats parses the demo with eco-rating's pipeline and copies eco-only
// metrics onto the CSC game's player maps, matched by SteamID.
func mergeEcoStats(game *csc.Game, r io.ReadCloser) {
	defer func() {
		// eco-rating parsing is best-effort; never let it break ProcessDemo.
		_ = recover()
	}()

	p := parser.NewDemoParser(r)
	if perr := p.Parse(); perr != nil {
		return
	}

	players := p.GetPlayers()
	if game.MapName == "" {
		game.MapName = p.GetMapName()
	}

	mergeInto(game.TotalPlayerStats, players)
	mergeInto(game.TPlayerStats, players)
	mergeInto(game.CtPlayerStats, players)
}

// mergeInto copies eco metrics onto every CSC player present in dst.
func mergeInto(dst map[uint64]*csc.PlayerStats, eco map[uint64]*model.PlayerStats) {
	for steamID, cscPlayer := range dst {
		ecoPlayer, ok := eco[steamID]
		if !ok || ecoPlayer == nil || cscPlayer == nil {
			continue
		}
		applyEcoStats(cscPlayer, ecoPlayer)
	}
}

// applyEcoStats maps eco-rating's PlayerStats onto the additive eco* fields of
// a CSC player. It only writes eco fields; demoScrape2 fields are untouched.
func applyEcoStats(dst *csc.PlayerStats, eco *model.PlayerStats) {
	// Primary eco rating -> swing_rating (never final_rating).
	dst.SwingRating = eco.FinalRating

	// Probability-swing metrics.
	dst.EcoProbabilitySwing = eco.ProbabilitySwing
	dst.EcoProbabilitySwingPerRound = eco.ProbabilitySwingPerRound
	dst.EcoTProbabilitySwing = eco.TProbabilitySwing
	dst.EcoCTProbabilitySwing = eco.CTProbabilitySwing

	// Eco duel economy.
	dst.EcoKillValue = eco.EcoKillValue
	dst.EcoDeathValue = eco.EcoDeathValue
	dst.EcoDuelSwing = eco.DuelSwing
	dst.EcoDuelSwingPerRound = eco.DuelSwingPerRound
	dst.EcoAdjustedKills = eco.EcoAdjustedKills

	// Eco ratings.
	dst.EcoHLTVRating = eco.HLTVRating
	dst.EcoHLTVCtRating = eco.CTRating
	dst.EcoHLTVTRating = eco.TRating
	dst.EcoCTEcoRating = eco.CTEcoRating
	dst.EcoTEcoRating = eco.TEcoRating
	dst.EcoSwingDisplayRating = eco.SwingRating
	dst.EcoPistolRoundRating = eco.PistolRoundRating

	// Assisted kills (eco definition).
	dst.EcoAssistedKills = eco.AssistedKills

	// Clutch attempts.
	dst.EcoClutch1v1Attempts = eco.Clutch1v1Attempts
	dst.EcoClutch1v2Attempts = eco.Clutch1v2Attempts
	dst.EcoClutch1v3Attempts = eco.Clutch1v3Attempts
	dst.EcoClutch1v4Attempts = eco.Clutch1v4Attempts
	dst.EcoClutch1v5Attempts = eco.Clutch1v5Attempts

	// Trade detail (eco model).
	dst.EcoTradeKills = eco.TradeKills
	dst.EcoTradeDenials = eco.TradeDenials
	dst.EcoFastTrades = eco.FastTrades
	dst.EcoOpeningDeathsTraded = eco.OpeningDeathsTraded

	// AWP extras.
	dst.EcoAWPOpeningKills = eco.AWPOpeningKills
	dst.EcoAWPMultiKillRounds = eco.AWPMultiKillRounds
	dst.EcoAWPDeaths = eco.AWPDeaths
	dst.EcoRoundsWithAWPKill = eco.RoundsWithAWPKill
	dst.EcoAWPKillsPerRound = eco.AWPKillsPerRound

	// Timing.
	dst.EcoTimeAlivePerRound = eco.TimeAlivePerRound
	dst.EcoAvgTimeToDeath = eco.AvgTimeToDeath
	dst.EcoAvgTimeToKill = eco.AvgTimeToKill

	// Other unique eco metrics.
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
