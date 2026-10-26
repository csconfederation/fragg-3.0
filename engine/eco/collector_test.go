package eco

import (
	"testing"

	"github.com/csconfederation/fragg-3.0/engine/fact"
	"github.com/csconfederation/fragg-3.0/engine/round"
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
		if err := c.Fact(fact.Hurt{Snapshot: s, Attacker: &a, Player: &human, HealthDamageTaken: 10}, round.Context{}); err != nil {
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

// A death at the open's tick (time 0) is a death (#591): presence is the count, not DeathTime > 0.
func TestDeathAtTimeZero(t *testing.T) {
	c := New()
	ak := 1
	killer := fact.Player{Steam64: 1, Side: fact.SideT, Alive: true, Listed: true, Weapons: []int{ak}}
	victim := fact.Player{Steam64: 2, Side: fact.SideCT, Listed: true}
	s := fact.Snapshot{Tick: 100, T: fact.Team{Side: fact.SideT}, CT: fact.Team{Side: fact.SideCT},
		Players: []fact.Player{killer, victim}}
	c.Open(s, 2, s.Players, 1, 1)
	if err := c.Fact(fact.Kill{Snapshot: s, Killer: &killer, Victim: &victim, Weapon: &ak},
		round.Context{Expected: true, Victim: true, Killer: true, Enemy: true}); err != nil {
		t.Fatal(err)
	}
	p := c.Players([]*Round{c.Commit()})
	if p[2].Deaths != 1 || p[2].EarlyDeaths != 1 {
		t.Errorf("deaths %d, early deaths %d", p[2].Deaths, p[2].EarlyDeaths)
	}
}

// A team kill is a death the round counts (#591), with no kill for the killer.
func TestTeamKillIsADeath(t *testing.T) {
	c := New()
	ak := 7
	killer := fact.Player{Steam64: 1, Side: fact.SideT, Alive: true, Listed: true}
	victim := fact.Player{Steam64: 2, Side: fact.SideT, Listed: true}
	s := fact.Snapshot{Tick: 100, T: fact.Team{Side: fact.SideT}, CT: fact.Team{Side: fact.SideCT},
		Players: []fact.Player{killer, victim}}
	c.Open(s, 2, s.Players, 2, 0)
	s.Tick = 740
	x := round.Context{Expected: true, Victim: true, Killer: true, Before: round.State{TAlive: 2}, After: round.State{TAlive: 1}}
	if err := c.Fact(fact.Kill{Snapshot: s, Killer: &killer, Victim: &victim, Weapon: &ak}, x); err != nil {
		t.Fatal(err)
	}
	r := c.Commit()
	if v := r.stats[2]; v.Deaths != 1 || v.DeathTime != 10 {
		t.Errorf("victim deaths %d at %v s", v.Deaths, v.DeathTime)
	}
	if k := r.stats[1]; k.Kills != 0 {
		t.Errorf("killer kills %d", k.Kills)
	}
}

// Survival is final at commit (#591): a death after the win isn't a survival or a save.
func TestPostRoundDeathIsNoSurvival(t *testing.T) {
	c := New()
	ak := 7
	killer := fact.Player{Steam64: 1, Side: fact.SideT, Alive: true, Listed: true}
	victim := fact.Player{Steam64: 2, Side: fact.SideCT, Alive: true, Listed: true}
	s := fact.Snapshot{Tick: 100, T: fact.Team{Side: fact.SideT}, CT: fact.Team{Side: fact.SideCT},
		Players: []fact.Player{killer, victim}}
	c.Open(s, 2, s.Players, 1, 1)
	c.Win(fact.SideT, s, s.Players)
	x := round.Context{Expected: true, Victim: true, Killer: true, Enemy: true,
		Before: round.State{TAlive: 1, CTAlive: 1}, After: round.State{TAlive: 1}}
	if err := c.Fact(fact.Kill{Snapshot: s, Killer: &killer, Victim: &victim, Weapon: &ak}, x); err != nil {
		t.Fatal(err)
	}
	r := c.Commit()
	if v := r.stats[2]; v.Survived || v.SavedWeapons {
		t.Errorf("victim survived %v, saved %v", v.Survived, v.SavedWeapons)
	}
	if k := r.stats[1]; !k.Survived || k.SavedWeapons {
		t.Errorf("killer survived %v, saved %v", k.Survived, k.SavedWeapons)
	}
}
