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
		TProbabilitySwing:  0.6,
		CTProbabilitySwing: 0.3,
		TEcoRating:         1.2,
		CTEcoRating:        0.8,
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
}
