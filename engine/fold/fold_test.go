package fold

import (
	"errors"
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
