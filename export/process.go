package export

import (
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/csconfederation/fragg-3.0/csc"
	"github.com/csconfederation/fragg-3.0/model"
	"github.com/csconfederation/fragg-3.0/parser"
	"github.com/csconfederation/fragg-3.0/rating/swing"

	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
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
// It parses the demo once. CSC handlers own round lifecycle (integrity, knife,
// redo-round dedup). Eco/swing handlers write round-scoped stats that are folded
// into match totals only for rounds that survive removeInvalidRounds.
//
// Eco metrics are merged onto the CSC game additively (never overwriting a CSC
// field); where the two pipelines compute the same concept differently, the eco
// value is carried under a distinct eco* name. The eco primary rating is
// surfaced as swing_rating.
//
// The error contract matches demoScrape2: a non-nil *Game is always returned,
// ErrNoValidRounds is joined when there are no rounds, and an unexpected EOF on
// an already-finished match leaves Game.Result == "Ended".
func ProcessDemo(demo io.ReadCloser) (game *Game, err error) {
	game, _, err = ProcessDemoWithEco(demo, ProcessOptions{SwingConfig: swing.DefaultConfig()})
	return game, err
}

// ProcessOptions configures the shared CSC+eco parse.
type ProcessOptions struct {
	EnableLogging bool
	KDPRModifier  bool
	SwingConfig   swing.Config
}

// ProcessDemoWithEco runs the single-parse pipeline and also returns the eco
// parser (players, collector, logs) for CLI/batch callers.
func ProcessDemoWithEco(demo io.ReadCloser, opts ProcessOptions) (game *Game, eco *parser.DemoParser, err error) {
	defer func() {
		if r := recover(); r != nil {
			if game == nil {
				game = csc.InitGameObject()
			}
			err = fmt.Errorf("panic during demo processing: %v", r)
		}
	}()

	if opts.SwingConfig == (swing.Config{}) {
		opts.SwingConfig = swing.DefaultConfig()
	}

	cfg := demoinfocs.DefaultParserConfig
	cfg.IgnorePacketEntitiesPanic = true
	p := demoinfocs.NewParserWithConfig(demo, cfg)
	defer p.Close()

	eco = parser.AttachToParser(p, opts.EnableLogging, opts.KDPRModifier, opts.SwingConfig)

	game, err = csc.ProcessParser(p, csc.ParseHooks{
		OnInitRound:   eco.BeginRound,
		OnRoundWinCon: eco.FinalizeRound,
		OnRoundCommitted: func() any {
			return eco.SnapshotRound()
		},
	})
	if game == nil {
		game = csc.InitGameObject()
	}

	mergeEcoFromParser(game, eco)
	return game, eco, err
}

// mergeEcoFromParser folds surviving eco snapshots into match totals and copies
// eco-only metrics onto the CSC game's player maps.
func mergeEcoFromParser(game *csc.Game, eco *parser.DemoParser) {
	defer func() {
		if rec := recover(); rec != nil {
			log.Printf("export: eco merge panic recovered: %v", rec)
			game.EcoStatsOK = false
		}
	}()
	if eco == nil {
		return
	}

	snaps := make([]*parser.RoundSnapshot, 0, len(game.Rounds))
	for _, r := range game.Rounds {
		if r == nil {
			continue
		}
		snap, ok := r.EcoSnapshot().(*parser.RoundSnapshot)
		if ok && snap != nil {
			snaps = append(snaps, snap)
		}
	}
	eco.FoldSnapshots(snaps)
	eco.FinalizeStats()

	players := eco.GetPlayers()
	if game.MapName == "" {
		game.MapName = eco.GetMapName()
	}

	mergeInto(game.TotalPlayerStats, players, applyTotalEcoStats)
	mergeInto(game.TPlayerStats, players, applyTEcoStats)
	mergeInto(game.CtPlayerStats, players, applyCTEcoStats)

	ok, reason := evaluateEcoStats(game, players, eco.GetRoundsCounted())
	game.EcoStatsOK = ok
	if !ok {
		log.Printf("export: eco stats not OK: %s", reason)
	}
}

// evaluateEcoStats decides whether the eco merge produced complete, trustworthy
// data for this game. It returns false plus a short machine-greppable reason so
// operators can tell the failure classes apart in logs.
//
// Three things have to hold:
//
//  1. The eco parser produced players at all. A truncated demo can leave the
//     eco pipeline "successful" (demoinfocs tolerates ErrUnexpectedEndOfDemo)
//     with an empty player map, in which case every eco field stays zero.
//  2. Every non-bot player CSC saw was also seen by eco, on the whole-match map
//     and on each side map they played. Unmatched players silently keep zero
//     eco fields, which downstream would read as a genuine zero rating.
//  3. Eco's round count agrees with CSC's post-dedup valid round count. Eco
//     stats are folded from snapshots attached to CSC rounds, so after a
//     successful shared parse these should match. Any disagreement still
//     means the two pipelines aggregated different round sets.
//
// Bots are excluded from the coverage requirement: the eco parser deliberately
// skips them while the CSC pipeline keeps bot rows.
//
// Every condition is evaluated (rather than returning on the first failure) so a
// noisy condition cannot mask the others in logs.
func evaluateEcoStats(game *csc.Game, eco map[uint64]*model.PlayerStats, ecoRounds int) (bool, string) {
	// Nothing downstream of this is meaningful when eco saw no players at all,
	// and every other condition would trivially fail too, so report just this.
	if len(eco) == 0 {
		return false, "empty-eco-map (eco parse produced no players)"
	}

	var reasons []string

	humans, missing := coverage(game.TotalPlayerStats, eco)
	switch {
	case humans == 0:
		reasons = append(reasons, "no-csc-players (nothing to merge onto)")
	case missing > 0:
		reasons = append(reasons, fmt.Sprintf("coverage-gap (%d of %d non-bot players missing from eco map)", missing, humans))
	}

	// game.Rounds is the post-removeInvalidRounds slice (endOfMatchProcessing
	// runs it before aggregating), so it is the deduped count. game.TotalRounds
	// is deliberately not used here: it is derived from the server's round
	// integrity counters, not from the rounds that were actually aggregated.
	if cscRounds := len(game.Rounds); ecoRounds != cscRounds {
		reasons = append(reasons, fmt.Sprintf("round-divergence (eco=%d csc=%d)", ecoRounds, cscRounds))
	}

	if n := sideCoverageGaps(game.TPlayerStats, eco, func(p *model.PlayerStats) int { return p.TRoundsPlayed }); n > 0 {
		reasons = append(reasons, fmt.Sprintf("side-coverage-gap (%d non-bot players have T rounds in csc but none in eco)", n))
	}
	if n := sideCoverageGaps(game.CtPlayerStats, eco, func(p *model.PlayerStats) int { return p.CTRoundsPlayed }); n > 0 {
		reasons = append(reasons, fmt.Sprintf("side-coverage-gap (%d non-bot players have CT rounds in csc but none in eco)", n))
	}

	if len(reasons) > 0 {
		return false, strings.Join(reasons, "; ")
	}
	return true, ""
}

// coverage counts the non-bot players in dst and how many of them have no
// counterpart in the eco map.
func coverage(dst map[uint64]*csc.PlayerStats, eco map[uint64]*model.PlayerStats) (humans, missing int) {
	for steamID, cscPlayer := range dst {
		if cscPlayer == nil || cscPlayer.IsBot {
			continue
		}
		humans++
		if eco[steamID] == nil {
			missing++
		}
	}
	return humans, missing
}

// sideCoverageGaps counts non-bot players who played rounds on a side according
// to CSC but have no eco rounds recorded for that side. Their side swing_rating
// would otherwise stay zero while the game was flagged OK.
func sideCoverageGaps(dst map[uint64]*csc.PlayerStats, eco map[uint64]*model.PlayerStats, sideRounds func(*model.PlayerStats) int) int {
	gaps := 0
	for steamID, cscPlayer := range dst {
		if cscPlayer == nil || cscPlayer.IsBot || cscPlayer.Rounds <= 0 {
			continue
		}
		ecoPlayer := eco[steamID]
		if ecoPlayer == nil || sideRounds(ecoPlayer) <= 0 {
			gaps++
		}
	}
	return gaps
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
func applyTEcoStats(dst *csc.PlayerStats, eco *model.PlayerStats) {
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
func applyCTEcoStats(dst *csc.PlayerStats, eco *model.PlayerStats) {
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
