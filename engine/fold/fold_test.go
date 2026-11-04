package fold

import (
	"errors"
	"math"
	"testing"

	"github.com/csconfederation/fragg-3.0/engine/cscstats"
	"github.com/csconfederation/fragg-3.0/engine/eco"
	"github.com/csconfederation/fragg-3.0/engine/match"
)

// Round records are T or CT; anything else would leak into the CT rows, so it's an invariant.
func TestSideIsTOrCT(t *testing.T) {
	g := &cscstats.Game{Rounds: []*cscstats.Round{{RoundNum: 4, EndingTick: 900,
		PlayerStats: map[uint64]*cscstats.PlayerStats{1: {SteamID: "1", Side: 1}}}}}
	var inv *match.InvariantError
	if err := Game(g, eco.New(), nil); !errors.As(err, &inv) || inv.Tick != 900 {
		t.Fatalf("err %v; want an invariant error at the round's end", err)
	}
}

// atd averages time to death over the rounds a player died in (D20).
func TestAtdIsPerDeath(t *testing.T) {
	rows := map[uint64]*cscstats.PlayerStats{
		1: {Rounds: 4, Deaths: 2, TicksAlive: 64 * 30 * 2}, // died twice at 30 s, survived twice
		2: {Rounds: 4, TicksAlive: 0},                      // never died
	}
	derive(rows, 64)
	if rows[1].Atd != 30 || rows[2].Atd != 0 {
		t.Errorf("atd %d, %d; want 30, 0", rows[1].Atd, rows[2].Atd)
	}
}

// Each whole-match row gets rounds, wins and a rating per buy of its team; a full buy against an eco
// also counts as anti-eco, and a round without a known buy counts in none (Core-planning#590).
func TestBuyScopes(t *testing.T) {
	type side struct {
		clan string
		buy  cscstats.Buy
	}
	round := func(num int, a, b side, aWon bool) *cscstats.Round {
		rf := map[bool]int{true: 1, false: 0}
		return &cscstats.Round{RoundNum: int8(num), InitTerroristCount: 1, InitCTerroristCount: 1,
			PlayerStats: map[uint64]*cscstats.PlayerStats{
				1: {SteamID: "1", Side: 2, TeamClanName: a.clan, Kills: 1, Damage: 100, RF: rf[aWon], KastRounds: 1, WinPoints: 1},
				2: {SteamID: "2", Side: 3, TeamClanName: b.clan, Deaths: 1, RF: rf[!aWon]}},
			TeamStats: map[string]*cscstats.TeamStats{a.clan: {Buy: a.buy}, b.clan: {Buy: b.buy}}}
	}
	g := &cscstats.Game{TickRate: 64, Rounds: []*cscstats.Round{
		round(1, side{"A", cscstats.BuyPistol}, side{"B", cscstats.BuyPistol}, true),
		round(2, side{"A", cscstats.BuyFull}, side{"B", cscstats.BuyEco}, true),
		round(3, side{"A", cscstats.BuyForce}, side{"B", cscstats.BuyFull}, false),
		round(4, side{"A", cscstats.BuyUnknown}, side{"B", cscstats.BuyFull}, true),
	}}
	if err := Game(g, eco.New(), nil); err != nil {
		t.Fatal(err)
	}
	a, b := g.TotalPlayerStats[1], g.TotalPlayerStats[2]
	type counts struct{ pistol, pistolW, eco, ecoW, force, forceW, full, fullW, anti, antiW int }
	got := func(p *cscstats.PlayerStats) counts {
		return counts{p.PistolRounds, p.PistolRoundsWon, p.EcoBuyRounds, p.EcoBuyRoundsWon, p.ForceBuyRounds,
			p.ForceBuyRoundsWon, p.FullBuyRounds, p.FullBuyRoundsWon, p.AntiEcoRounds, p.AntiEcoRoundsWon}
	}
	if c := got(a); c != (counts{1, 1, 0, 0, 1, 0, 1, 1, 1, 1}) {
		t.Errorf("A %+v", c)
	}
	if c := got(b); c != (counts{1, 0, 1, 0, 0, 0, 2, 1, 0, 0}) {
		t.Errorf("B %+v", c)
	}
	for _, r := range []float64{a.PistolRoundRating, a.FullBuyRating, a.ForceBuyRating, a.AntiEcoRating, b.EcoBuyRating, b.FullBuyRating} {
		if math.IsNaN(r) || math.IsInf(r, 0) || r < 0 {
			t.Errorf("rating %v", r)
		}
	}
	if a.EcoBuyRating != 0 {
		t.Errorf("A eco rating %v without eco rounds", a.EcoBuyRating)
	}

	// A scope holding every round is rated exactly as the whole match.
	g = &cscstats.Game{TickRate: 64, Rounds: []*cscstats.Round{
		round(2, side{"A", cscstats.BuyFull}, side{"B", cscstats.BuyFull}, true),
		round(3, side{"A", cscstats.BuyFull}, side{"B", cscstats.BuyFull}, false),
		round(4, side{"A", cscstats.BuyFull}, side{"B", cscstats.BuyFull}, true),
	}}
	if err := Game(g, eco.New(), nil); err != nil {
		t.Fatal(err)
	}
	for _, p := range g.TotalPlayerStats {
		if p.FullBuyRating != p.Rating || p.FullBuyRounds != p.Rounds {
			t.Errorf("player %s: full-buy rating %v over %d rounds, match rating %v over %d", p.SteamID, p.FullBuyRating, p.FullBuyRounds, p.Rating, p.Rounds)
		}
	}
}
