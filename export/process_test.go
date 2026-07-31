package export

import (
	"strings"
	"testing"

	"github.com/csconfederation/fragg-3.0/csc"
	"github.com/csconfederation/fragg-3.0/model"
)

func TestSideAwareEcoMergeDoesNotDuplicateWholeMatch(t *testing.T) {
	eco := &model.PlayerStats{
		FinalRating:        1.5,
		ProbabilitySwing:   0.9,
		RoundsPlayed:       18,
		TProbabilitySwing:  0.6,
		CTProbabilitySwing: 0.3,
		TEcoRating:         1.2,
		CTEcoRating:        0.8,
		TEcoKillValue:      400,
		TEcoDeathValue:     100,
		CTEcoKillValue:     250,
		CTEcoDeathValue:    50,
		TradeKills:         5,
		TTradeKills:        4,
		CTTradeKills:       1,
		TRoundsPlayed:      10,
		CTRoundsPlayed:     8,
		KnifeKills:         2,
		TKnifeKills:        2,
		CTKnifeKills:       0,
	}

	total := &csc.PlayerStats{}
	tSide := &csc.PlayerStats{}
	ctSide := &csc.PlayerStats{}

	applyTotalEcoStats(total, eco)
	applyTEcoStats(tSide, eco)
	applyCTEcoStats(ctSide, eco)

	if total.EcoTradeKills != 5 {
		t.Fatalf("total ecoTradeKills = %d, want 5", total.EcoTradeKills)
	}
	if tSide.EcoTradeKills != 4 {
		t.Fatalf("T ecoTradeKills = %d, want 4", tSide.EcoTradeKills)
	}
	if ctSide.EcoTradeKills != 1 {
		t.Fatalf("CT ecoTradeKills = %d, want 1", ctSide.EcoTradeKills)
	}
	if tSide.EcoTradeKills+ctSide.EcoTradeKills != total.EcoTradeKills {
		t.Fatal("T+CT trade kills should equal total")
	}
	if tSide.SwingRating != eco.TEcoRating {
		t.Fatalf("T swing_rating = %f, want TEcoRating %f", tSide.SwingRating, eco.TEcoRating)
	}
	if ctSide.SwingRating != eco.CTEcoRating {
		t.Fatalf("CT swing_rating = %f, want CTEcoRating %f", ctSide.SwingRating, eco.CTEcoRating)
	}
	if tSide.EcoKnifeKills != 2 || ctSide.EcoKnifeKills != 0 {
		t.Fatalf("side knife kills T=%d CT=%d", tSide.EcoKnifeKills, ctSide.EcoKnifeKills)
	}

	wantTDuel := eco.TEcoKillValue - eco.TEcoDeathValue
	wantCTDuel := eco.CTEcoKillValue - eco.CTEcoDeathValue
	if tSide.EcoDuelSwing != wantTDuel {
		t.Fatalf("T ecoDuelSwing = %f, want %f (not probability swing)", tSide.EcoDuelSwing, wantTDuel)
	}
	if ctSide.EcoDuelSwing != wantCTDuel {
		t.Fatalf("CT ecoDuelSwing = %f, want %f (not probability swing)", ctSide.EcoDuelSwing, wantCTDuel)
	}
	if tSide.EcoDeathValue != eco.TEcoDeathValue {
		t.Fatalf("T ecoDeathValue = %f, want %f", tSide.EcoDeathValue, eco.TEcoDeathValue)
	}
	if ctSide.EcoDeathValue != eco.CTEcoDeathValue {
		t.Fatalf("CT ecoDeathValue = %f, want %f", ctSide.EcoDeathValue, eco.CTEcoDeathValue)
	}

	wantTDisplay := swingDisplayRating(eco.TProbabilitySwing, eco.TRoundsPlayed)
	wantCTDisplay := swingDisplayRating(eco.CTProbabilitySwing, eco.CTRoundsPlayed)
	if tSide.EcoSwingDisplayRating != wantTDisplay {
		t.Fatalf("T ecoSwingDisplayRating = %f, want %f (not TEcoRating)", tSide.EcoSwingDisplayRating, wantTDisplay)
	}
	if ctSide.EcoSwingDisplayRating != wantCTDisplay {
		t.Fatalf("CT ecoSwingDisplayRating = %f, want %f (not CTEcoRating)", ctSide.EcoSwingDisplayRating, wantCTDisplay)
	}
	if tSide.EcoSwingDisplayRating == eco.TEcoRating {
		t.Fatal("T ecoSwingDisplayRating must not equal TEcoRating")
	}
	if ctSide.EcoSwingDisplayRating == eco.CTEcoRating {
		t.Fatal("CT ecoSwingDisplayRating must not equal CTEcoRating")
	}
	if tSide.EcoDuelSwing == eco.TProbabilitySwing {
		t.Fatal("T ecoDuelSwing must not equal TProbabilitySwing")
	}
}

// ecoPlayer builds an eco player that played both sides, so coverage passes.
func ecoPlayer() *model.PlayerStats {
	return &model.PlayerStats{RoundsPlayed: 24, TRoundsPlayed: 12, CTRoundsPlayed: 12}
}

// goodGame builds a CSC game whose maps are fully covered by eco for steamIDs.
func goodGame(rounds int, steamIDs ...uint64) *csc.Game {
	game := &csc.Game{
		Rounds:           make([]*csc.Round, rounds),
		TotalPlayerStats: map[uint64]*csc.PlayerStats{},
		TPlayerStats:     map[uint64]*csc.PlayerStats{},
		CtPlayerStats:    map[uint64]*csc.PlayerStats{},
	}
	for _, id := range steamIDs {
		game.TotalPlayerStats[id] = &csc.PlayerStats{Rounds: rounds}
		game.TPlayerStats[id] = &csc.PlayerStats{Rounds: rounds / 2}
		game.CtPlayerStats[id] = &csc.PlayerStats{Rounds: rounds / 2}
	}
	return game
}

func TestEvaluateEcoStatsHealthy(t *testing.T) {
	game := goodGame(24, 1, 2)
	eco := map[uint64]*model.PlayerStats{1: ecoPlayer(), 2: ecoPlayer()}

	ok, reason := evaluateEcoStats(game, eco, 24)
	if !ok {
		t.Fatalf("healthy game flagged not OK: %s", reason)
	}
}

func TestEvaluateEcoStatsEmptyEcoMap(t *testing.T) {
	game := goodGame(24, 1)

	ok, reason := evaluateEcoStats(game, map[uint64]*model.PlayerStats{}, 24)
	if ok {
		t.Fatal("empty eco map must not be flagged OK")
	}
	if !strings.Contains(reason, "empty-eco-map") {
		t.Fatalf("reason = %q, want empty-eco-map", reason)
	}
}

func TestEvaluateEcoStatsCoverageGap(t *testing.T) {
	game := goodGame(24, 1, 2)
	eco := map[uint64]*model.PlayerStats{1: ecoPlayer()}

	ok, reason := evaluateEcoStats(game, eco, 24)
	if ok {
		t.Fatal("player missing from eco map must not be flagged OK")
	}
	if !strings.Contains(reason, "coverage-gap") {
		t.Fatalf("reason = %q, want coverage-gap", reason)
	}
}

func TestEvaluateEcoStatsBotsDoNotCountAgainstCoverage(t *testing.T) {
	game := goodGame(24, 1)
	const botID = uint64(99)
	game.TotalPlayerStats[botID] = &csc.PlayerStats{Rounds: 24, IsBot: true}
	game.TPlayerStats[botID] = &csc.PlayerStats{Rounds: 12, IsBot: true}
	game.CtPlayerStats[botID] = &csc.PlayerStats{Rounds: 12, IsBot: true}
	eco := map[uint64]*model.PlayerStats{1: ecoPlayer()}

	ok, reason := evaluateEcoStats(game, eco, 24)
	if !ok {
		t.Fatalf("bot missing from eco map must stay OK, got %s", reason)
	}
}

func TestEvaluateEcoStatsOnlyBots(t *testing.T) {
	game := goodGame(24)
	game.TotalPlayerStats[99] = &csc.PlayerStats{Rounds: 24, IsBot: true}
	eco := map[uint64]*model.PlayerStats{1: ecoPlayer()}

	ok, reason := evaluateEcoStats(game, eco, 24)
	if ok {
		t.Fatal("game with no non-bot csc players must not be flagged OK")
	}
	if !strings.Contains(reason, "no-csc-players") {
		t.Fatalf("reason = %q, want no-csc-players", reason)
	}
}

func TestEvaluateEcoStatsRoundDivergence(t *testing.T) {
	game := goodGame(24, 1)
	eco := map[uint64]*model.PlayerStats{1: ecoPlayer()}

	// A restarted match replays rounds: eco counts them all, CSC dedupes.
	ok, reason := evaluateEcoStats(game, eco, 30)
	if ok {
		t.Fatal("inflated eco round count must not be flagged OK")
	}
	if !strings.Contains(reason, "round-divergence") {
		t.Fatalf("reason = %q, want round-divergence", reason)
	}
	if !strings.Contains(reason, "eco=30") || !strings.Contains(reason, "csc=24") {
		t.Fatalf("reason = %q, want both round counts logged", reason)
	}

	// Divergence in the other direction is equally untrustworthy.
	if ok, _ := evaluateEcoStats(game, eco, 20); ok {
		t.Fatal("deflated eco round count must not be flagged OK")
	}
}

func TestEvaluateEcoStatsSideCoverageGap(t *testing.T) {
	game := goodGame(24, 1)
	eco := map[uint64]*model.PlayerStats{
		// Player was seen by eco overall, but never on T (e.g. a late join whose
		// round stats never got a side assigned).
		1: {RoundsPlayed: 24, TRoundsPlayed: 0, CTRoundsPlayed: 12},
	}

	ok, reason := evaluateEcoStats(game, eco, 24)
	if ok {
		t.Fatal("missing T-side eco data must not be flagged OK")
	}
	if !strings.Contains(reason, "side-coverage-gap") {
		t.Fatalf("reason = %q, want side-coverage-gap", reason)
	}
}

func TestEvaluateEcoStatsReportsEveryFailedCondition(t *testing.T) {
	// Round divergence must not mask the coverage failures behind it.
	game := goodGame(24, 1, 2)
	eco := map[uint64]*model.PlayerStats{1: ecoPlayer()}

	ok, reason := evaluateEcoStats(game, eco, 30)
	if ok {
		t.Fatal("multiple failures must not be flagged OK")
	}
	for _, want := range []string{"coverage-gap", "round-divergence", "side-coverage-gap"} {
		if !strings.Contains(reason, want) {
			t.Fatalf("reason = %q, missing %s", reason, want)
		}
	}
}

func TestSwingDisplayRatingClamp(t *testing.T) {
	if got := swingDisplayRating(0, 10); got != 1.0 {
		t.Fatalf("zero swing = %f, want 1.0", got)
	}
	if got := swingDisplayRating(10, 10); got != 1.5 {
		t.Fatalf("high swing clamped = %f, want 1.5", got)
	}
	if got := swingDisplayRating(-10, 10); got != 0.5 {
		t.Fatalf("low swing clamped = %f, want 0.5", got)
	}
	if got := swingDisplayRating(0.4, 10); got != 1.4 {
		t.Fatalf("+4%% swing = %f, want 1.4", got)
	}
}
