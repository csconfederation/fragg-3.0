package fold

import (
	"errors"
	"math"
	"reflect"
	"testing"

	"github.com/csconfederation/fragg-3.0/engine/fact"
	"github.com/csconfederation/fragg-3.0/engine/match"
	"github.com/csconfederation/fragg-3.0/engine/row"
)

// A round of two one-player teams, A on T and B on CT; a wins says whether A did.
func round(num int, a, b row.Buy, aWon bool) *row.Round {
	rf := map[bool]int{true: 1, false: 0}
	return &row.Round{
		Meta: row.Meta{RoundNum: num, InitTerroristCount: 1, InitCTCount: 1, EndingTick: 900},
		Players: map[uint64]*row.Player{
			1: {ID: row.ID{Name: "a", SteamID: "1", TeamClanName: "A", Side: 2}, Combat: row.Combat{Kills: 1, EcoKills: 1, Damage: 100},
				Objectives: row.Objectives{RF: rf[aWon], WinPoints: 1}, KAST: row.KAST{KastRounds: 1}},
			2: {ID: row.ID{Name: "b", SteamID: "2", TeamClanName: "B", Side: 3}, Combat: row.Combat{Deaths: 1},
				Objectives: row.Objectives{RF: rf[!aWon]}},
		},
		Teams: map[fact.Side]*row.Team{
			fact.SideT:  {Clan: "A", TeamEconomy: row.TeamEconomy{Buy: a}},
			fact.SideCT: {Clan: "B", TeamEconomy: row.TeamEconomy{Buy: b}},
		},
	}
}

// Round records are T or CT; anything else would leak into the CT rows, so it's an invariant.
func TestSideIsTOrCT(t *testing.T) {
	r := round(4, row.BuyFull, row.BuyFull, true)
	r.Players[1].Side = 1
	var inv *match.InvariantError
	if _, err := Fold([]*row.Round{r}); !errors.As(err, &inv) || inv.Tick != 900 {
		t.Fatalf("err %v; want an invariant error at the round's end", err)
	}
}

// atd averages time to death over the rounds a player died in (D20).
func TestAtdIsPerDeath(t *testing.T) {
	rows := map[uint64]*row.Player{
		1: {ID: row.ID{Rounds: 4}, Combat: row.Combat{Deaths: 2}, Survival: row.Survival{TicksAlive: 64 * 30 * 2}}, // died twice at 30 s, survived twice
		2: {ID: row.ID{Rounds: 4}},                                                                                 // never died
	}
	derive(rows)
	if rows[1].Atd != 30 || rows[2].Atd != 0 {
		t.Errorf("atd %d, %d; want 30, 0", rows[1].Atd, rows[2].Atd)
	}
	if rows[2].Tr != 0.5 {
		t.Errorf("tr %v with no deaths, want 0.50", rows[2].Tr)
	}
}

// Each whole-match row gets rounds, wins and a rating per buy of its team; a full buy against an eco
// also counts as anti-eco, and a round without a known buy counts in none (Core-planning#590).
func TestBuyScopes(t *testing.T) {
	m, err := Fold([]*row.Round{
		round(1, row.BuyPistol, row.BuyPistol, true),
		round(2, row.BuyFull, row.BuyEco, true),
		round(3, row.BuyForce, row.BuyFull, false),
		round(4, row.BuyUnknown, row.BuyFull, true),
	})
	if err != nil {
		t.Fatal(err)
	}
	a, b := m.Whole[1], m.Whole[2]
	type counts struct{ pistol, pistolW, eco, ecoW, force, forceW, full, fullW, anti, antiW int }
	got := func(p *row.Player) counts {
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
	m, err = Fold([]*row.Round{
		round(2, row.BuyFull, row.BuyFull, true),
		round(3, row.BuyFull, row.BuyFull, false),
		round(4, row.BuyFull, row.BuyFull, true),
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range m.Whole {
		if p.FullBuyRating != p.Rating || p.FullBuyRounds != p.Rounds {
			t.Errorf("player %s: full-buy rating %v over %d rounds, match rating %v over %d", p.SteamID, p.FullBuyRating, p.FullBuyRounds, p.Rating, p.Rounds)
		}
	}
}

// The sides split by the round's side, and a row takes its name and team from its latest round.
func TestSidesAndIdentity(t *testing.T) {
	r1, r2 := round(1, row.BuyPistol, row.BuyPistol, true), round(2, row.BuyFull, row.BuyFull, false)
	r2.Players[1].Name, r2.Players[1].TeamClanName, r2.Players[1].Side = "renamed", "C", 3
	m, err := Fold([]*row.Round{r1, r2})
	if err != nil {
		t.Fatal(err)
	}
	w := m.Whole[1]
	if w.Name != "renamed" || w.TeamClanName != "C" || w.Rounds != 2 || w.Side != total {
		t.Errorf("whole %+v", w.ID)
	}
	if m.T[1].Rounds != 1 || m.CT[1].Rounds != 1 || m.T[1].Side != 2 || m.CT[1].Side != 3 {
		t.Errorf("sides: T %d CT %d", m.T[1].Rounds, m.CT[1].Rounds)
	}
	if w.TRounds != 1 || w.CtRounds != 1 || w.TKills != 1 {
		t.Errorf("projection %+v", w.Projection)
	}
}

// D43: the whole row's HLTV rating uses the exact survival count, as a side row does (15/22*22
// is 14.999..., which a rate round trip would truncate to 14).
func TestHLTVSurvivals(t *testing.T) {
	mk := func() *row.Player { return &row.Player{ID: row.ID{Rounds: 22}} }
	m := &Match{Whole: map[uint64]*row.Player{1: mk()}, T: map[uint64]*row.Player{1: mk()}, CT: map[uint64]*row.Player{}}
	tallies := map[*row.Player]*tally{m.Whole[1]: {survived: 15}, m.T[1]: {survived: 15}}
	m.eco(1, tallies)
	if m.Whole[1].HLTVRating != m.T[1].HLTVRating {
		t.Errorf("whole %v, side %v: both rows should count 15 survivals", m.Whole[1].HLTVRating, m.T[1].HLTVRating)
	}
}

// Every field of a Player is summed, computed by the fold, or round-only: a new field of another
// kind must be tagged, or sum panics.
func TestSummable(t *testing.T) {
	paths := plan(reflect.TypeFor[row.Player]())
	if len(paths) == 0 {
		t.Error("nothing is summed")
	}
	plan(reflect.TypeFor[row.Team]())
}
