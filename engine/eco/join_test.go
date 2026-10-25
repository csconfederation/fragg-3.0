package eco

import (
	"testing"

	"github.com/csconfederation/fragg-3.0/engine/fact"
)

// A human who joins mid-round gets the side CSC gave them, so their side rounds count.
func TestJoinCoversTheRound(t *testing.T) {
	c := New()
	s := fact.Snapshot{T: fact.Team{Side: fact.SideT, Clan: "Alpha"}, CT: fact.Team{Side: fact.SideCT, Clan: "Bravo"},
		Players: []fact.Player{{Steam64: 1, Side: fact.SideT, Alive: true, Connected: true, Listed: true}}}
	c.Open(s, 2)
	c.Join(2, "late", false, fact.SideCT, "Bravo")
	c.Join(3, "bot", true, fact.SideCT, "Bravo")
	p := c.Players([]*Round{c.Commit()})
	if p[2] == nil || p[2].CTRoundsPlayed != 1 || p[2].RoundsPlayed != 1 || p[2].TeamName != "Bravo" {
		t.Fatalf("joiner %+v", p[2])
	}
	if p[3] != nil {
		t.Errorf("bot joined eco: %+v", p[3])
	}
}
