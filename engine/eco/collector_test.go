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
		if err := c.Fact(fact.Hurt{Snapshot: s, Attacker: &a, Player: &human, HealthDamageTaken: 10}, round.Context{Enemy: true}); err != nil {
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
	c.Win(fact.SideT, s, s.Players, round.State{})
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

// A winner who disconnected without dying won the round: no save (#591, review50).
func TestDisconnectedWinnerIsNoSave(t *testing.T) {
	c := New()
	winner := fact.Player{Steam64: 1, Side: fact.SideT, Alive: true, Listed: true}
	loser := fact.Player{Steam64: 2, Side: fact.SideCT, Alive: true, Listed: true}
	s := fact.Snapshot{Tick: 100, T: fact.Team{Side: fact.SideT}, CT: fact.Team{Side: fact.SideCT},
		Players: []fact.Player{winner, loser}}
	c.Open(s, 2, s.Players, 1, 1)
	s.Players = []fact.Player{loser}
	c.Win(fact.SideT, s, s.Players, round.State{})
	r := c.Commit()
	if w := r.stats[1]; !w.TeamWon || w.SavedWeapons {
		t.Errorf("winner won %v, saved %v", w.TeamWon, w.SavedWeapons)
	}
	if l := r.stats[2]; l.TeamWon || !l.SavedWeapons {
		t.Errorf("loser won %v, saved %v", l.TeamWon, l.SavedWeapons)
	}
}

// Eco takes engine/round's trades (D22): every avenged death is traded, and each trading kill counts.
func TestTradesFromRound(t *testing.T) {
	c := New()
	ak := 7
	a, b := fact.Player{Steam64: 1, Side: fact.SideT, Listed: true}, fact.Player{Steam64: 2, Side: fact.SideT, Listed: true}
	d, e := fact.Player{Steam64: 3, Side: fact.SideT, Listed: true}, fact.Player{Steam64: 11, Side: fact.SideCT, Listed: true}
	s := fact.Snapshot{Tick: 0, T: fact.Team{Side: fact.SideT}, CT: fact.Team{Side: fact.SideCT},
		Players: []fact.Player{a, b, d, e}}
	c.Open(s, 2, s.Players, 3, 1)
	kill := func(tick int, k, v fact.Player, avenged ...round.Death) {
		s.Tick = tick
		x := round.Context{Expected: true, Victim: true, Killer: true, Enemy: true, Avenged: avenged}
		if err := c.Fact(fact.Kill{Snapshot: s, Killer: &k, Victim: &v, Weapon: &ak}, x); err != nil {
			t.Fatal(err)
		}
	}
	kill(640, e, a)
	kill(704, e, b)
	kill(768, d, e, round.Death{Steam: 1, Tick: 640}, round.Death{Steam: 2, Tick: 704}) // 2 s after a's death
	r := c.Commit()
	if !r.stats[1].Traded || !r.stats[2].Traded || !r.stats[1].SavedByTeammate {
		t.Errorf("traded %v %v", r.stats[1].Traded, r.stats[2].Traded)
	}
	if k := r.stats[3]; k.TradeKills != 1 || k.FastTrades != 0 || !k.SavedTeammate {
		t.Errorf("trader: trade kills %d, fast %d, saved %v", k.TradeKills, k.FastTrades, k.SavedTeammate)
	}
}

// A bot killing a bot isn't a suicide, though both are Steam64 0: its trade counts (review52).
func TestBotTradesBot(t *testing.T) {
	c := New()
	ak := 7
	h := fact.Player{Steam64: 11, Side: fact.SideCT, Listed: true}
	tb := fact.Player{Steam64: 0, Side: fact.SideT, IsBot: true, UserID: 5, Listed: true}
	cb := fact.Player{Steam64: 0, Side: fact.SideCT, IsBot: true, UserID: 6, Listed: true}
	s := fact.Snapshot{T: fact.Team{Side: fact.SideT}, CT: fact.Team{Side: fact.SideCT}, Players: []fact.Player{h, tb, cb}}
	c.Open(s, 2, s.Players, 1, 2)
	x := round.Context{Expected: true, Victim: true, Killer: true, Enemy: true}
	s.Tick = 640
	if err := c.Fact(fact.Kill{Snapshot: s, Killer: &tb, Victim: &h, Weapon: &ak}, x); err != nil {
		t.Fatal(err)
	}
	x.Avenged = []round.Death{{Steam: 11, Tick: 640}}
	s.Tick = 700
	if err := c.Fact(fact.Kill{Snapshot: s, Killer: &cb, Victim: &tb, Weapon: &ak}, x); err != nil {
		t.Fatal(err)
	}
	if r := c.Commit(); !r.stats[11].Traded {
		t.Error("the human's death isn't traded")
	}
}

// Clutch attempts are the round's slots (#591, D23), however the side got to its last player.
func TestClutchFromRound(t *testing.T) {
	c := New()
	tp := fact.Player{Steam64: 1, Side: fact.SideT, Alive: true, Listed: true}
	ctp := fact.Player{Steam64: 11, Side: fact.SideCT, Alive: true, Listed: true}
	s := fact.Snapshot{T: fact.Team{Side: fact.SideT}, CT: fact.Team{Side: fact.SideCT}, Players: []fact.Player{tp, ctp}}
	c.Open(s, 2, s.Players, 1, 1)
	c.Win(fact.SideT, s, s.Players, round.State{TClutch: round.Clutch{Size: 2, Steam: 1}})
	r := c.Commit()
	if p := r.stats[1]; !p.ClutchAttempt || p.ClutchSize != 2 || !p.ClutchWon {
		t.Errorf("clutch attempt %v size %d won %v", p.ClutchAttempt, p.ClutchSize, p.ClutchWon)
	}
	if r.stats[11].ClutchAttempt {
		t.Error("CT clutch without a slot")
	}
}

// Flash stats are CSC's (#591): only counting flashes, and flash assists from kills, not every flash.
func TestFlashesFromRound(t *testing.T) {
	c := New()
	ak := 7
	a, b := fact.Player{Steam64: 1, Side: fact.SideT, Alive: true, Listed: true}, fact.Player{Steam64: 2, Side: fact.SideT, Alive: true, Listed: true}
	v := fact.Player{Steam64: 11, Side: fact.SideCT, Alive: true, Listed: true}
	s := fact.Snapshot{T: fact.Team{Side: fact.SideT}, CT: fact.Team{Side: fact.SideCT}, Players: []fact.Player{a, b, v}}
	c.Open(s, 2, s.Players, 2, 1)
	for _, counted := range []bool{false, true} {
		if err := c.Fact(fact.Flashed{Tick: 10, Attacker: &a, Player: &v, Duration: 2}, round.Context{Flash: counted}); err != nil {
			t.Fatal(err)
		}
	}
	x := round.Context{Expected: true, Victim: true, Killer: true, Enemy: true, Assist: true, FlashAssister: 1, FlashAssisted: true}
	if err := c.Fact(fact.Kill{Snapshot: s, Killer: &b, Victim: &v, Assister: &a, Weapon: &ak}, x); err != nil {
		t.Fatal(err)
	}
	r := c.Commit().stats[1]
	if r.EnemiesFlashed != 1 || r.EnemyFlashDuration != 2 || r.FlashAssists != 1 {
		t.Errorf("enemies flashed %d for %v s, flash assists %d", r.EnemiesFlashed, r.EnemyFlashDuration, r.FlashAssists)
	}
}
