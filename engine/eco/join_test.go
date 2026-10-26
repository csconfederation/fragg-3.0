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
	c.Open(s, 2, s.Players, 1, 0)
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

// Rows are the round's human members: a bot or a player who isn't a member gets none (#591).
func TestRowsAreMembers(t *testing.T) {
	c := New()
	human := fact.Player{Steam64: 1, Side: fact.SideT, Alive: true, Listed: true}
	bot := fact.Player{Steam64: 0, Side: fact.SideCT, Alive: true, Listed: true, IsBot: true}
	late := fact.Player{Steam64: 9, Side: fact.SideCT, Alive: true, Listed: true}
	enemy := fact.Player{Steam64: 2, Side: fact.SideCT, Alive: true, Listed: true}
	s := fact.Snapshot{T: fact.Team{Side: fact.SideT}, CT: fact.Team{Side: fact.SideCT},
		Players: []fact.Player{human, bot, late, enemy}}
	c.Open(s, 2, []fact.Player{human, bot, enemy}, 1, 2)
	for _, a := range []fact.Player{bot, late} {
		if err := c.Fact(fact.Hurt{Snapshot: s, Attacker: &a, Player: &human, HealthDamageTaken: 10}); err != nil {
			t.Fatal(err)
		}
	}
	r := c.Commit()
	if len(r.stats) != 2 || r.stats[1] == nil || r.stats[2] == nil {
		t.Fatalf("rows %v", r.stats)
	}
	if r.stats[1].DamageTaken != 20 {
		t.Errorf("member's damage taken %d", r.stats[1].DamageTaken)
	}
}
