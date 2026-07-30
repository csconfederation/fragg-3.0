package export

import (
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
