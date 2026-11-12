package roundlog

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/csconfederation/fragg-3.0/engine/fact"
	"github.com/csconfederation/fragg-3.0/engine/match"
)

var update = flag.Bool("update", false, "rewrite testdata/status from the builders below")

// sim builds the facts of a two-team match in CS2 dispatch order: Alpha is T in round 1, and the
// sides swap after round 12. Players 1-5 are Alpha's, 6-10 Bravo's.
type sim struct {
	facts        []Event
	tick, played int
}

func (s *sim) add(f ...fact.Fact) {
	for _, f := range f {
		s.facts = append(s.facts, Event{match.Event{Fact: f}})
	}
}

func (s *sim) snapshot(num int) fact.Snapshot {
	t, ct := "Alpha", "Bravo"
	if num > 12 {
		t, ct = ct, t
	}
	money := 4000
	if num == 1 || num == 13 {
		money = 800
	}
	snap := fact.Snapshot{Tick: s.tick, RoundsPlayed: s.played,
		T: fact.Team{Side: fact.SideT, ID: 2, Clan: t}, CT: fact.Team{Side: fact.SideCT, ID: 3, Clan: ct}}
	for i := uint64(1); i <= 10; i++ {
		side := fact.SideT
		if (i > 5) != (t == "Bravo") {
			side = fact.SideCT
		}
		id := 2
		if side == fact.SideCT {
			id = 3
		}
		snap.Players = append(snap.Players, fact.Player{Steam64: i, Name: fmt.Sprintf("p%d", i), Side: side, Alive: true,
			Connected: true, Listed: true, Money: money, TeamStateID: &id, UserID: int(i)})
	}
	return snap
}

func (s *sim) player(steam uint64) *fact.Player {
	snap := s.snapshot(s.played + 1)
	return &snap.Players[steam-1]
}

// open starts the next round at freeze end.
func (s *sim) open() {
	s.tick += 100
	s.add(fact.RoundStart{Tick: s.tick, RoundsPlayed: s.played})
	s.tick += 100
	s.add(fact.FreezeEnd{Snapshot: s.snapshot(s.played + 1)})
}

// end ends the open round, won by Alpha, with an official end unless it decides the match.
func (s *sim) end(official bool) {
	num := s.played + 1
	t, ct := "Alpha", "Bravo"
	if num > 12 {
		t, ct = ct, t
	}
	side, winner, loser := fact.SideT, fact.Team{Side: fact.SideT, ID: 2, Clan: t}, fact.Team{Side: fact.SideCT, ID: 3, Clan: ct}
	if t != "Alpha" {
		side, winner, loser = fact.SideCT, fact.Team{Side: fact.SideCT, ID: 3, Clan: ct}, fact.Team{Side: fact.SideT, ID: 2, Clan: t}
	}
	s.tick += 1000
	s.played++
	winner.Score, loser.Score = s.played, 0
	s.add(fact.ScoreUpdated{Tick: s.tick, Team: winner, Opponent: loser})
	s.add(fact.RoundEnd{Snapshot: s.snapshot(num), Winner: side, Reason: 9, WinnerTeam: winner, LoserTeam: loser})
	if official {
		s.tick += 300
		s.add(fact.RoundEndOfficial{Snapshot: fact.Snapshot{Tick: s.tick, RoundsPlayed: s.played}})
	}
}

func (s *sim) play(n int) {
	for i := 0; i < n; i++ {
		s.open()
		s.end(true)
	}
}

// match13 plays Alpha's 13-0, the deciding round without an official end, and the game end.
func (s *sim) match13() {
	s.play(12)
	s.open()
	s.end(false)
	s.add(fact.GamePhase{Tick: s.tick, Phase: fact.GamePhaseEnded}, fact.WinPanelMatch{Tick: s.tick})
}

func (s *sim) terminal(kind string) Terminal {
	return Terminal{Kind: kind, End: fact.Snapshot{Tick: s.tick + 500, RoundsPlayed: s.played,
		T: fact.Team{Side: fact.SideT, Score: s.played}, CT: fact.Team{Side: fact.SideCT}}}
}

// statuses are the fixtures. Each Want is written here from the lifecycle's rules (match.keep,
// Finish's precedence), and checked against the baseline build (fragg ca9fe6b) when added.
func statuses() []*Status {
	var out []*Status

	// A deriver error is the match's first error, and facts after it, including a later lifecycle
	// error (a score for a team that isn't in the match), don't replace it.
	s := &sim{}
	s.open()
	s.add(fact.BombPlanted{Snapshot: fact.Snapshot{Tick: s.tick + 50}})
	s.add(fact.ScoreUpdated{Tick: s.tick + 60, Team: fact.Team{Clan: "Ghosts", Side: fact.SideT}, Opponent: fact.Team{Clan: "Bravo", Side: fact.SideCT}})
	out = append(out, &Status{Name: "deriver-error-before-lifecycle-error",
		Why:      "bomb planted by nobody in a live undecided round (round step); a later unknown-team score must not replace it",
		Facts:    s.facts,
		Terminal: s.terminal("Truncated"),
		Want:     Want{Error: "match invariant at tick 0: bomb planted by nobody"}})

	// Eco's check runs on non-live facts too, after the match has ended.
	s = &sim{}
	s.match13()
	s.add(fact.BombPlanted{Snapshot: fact.Snapshot{Tick: s.tick + 10, RoundsPlayed: 13}})
	out = append(out, &Status{Name: "post-match-bomb-error",
		Why:      "bomb planted by nobody after the match ended (eco step, not live): an error, not a payload",
		Facts:    s.facts,
		Terminal: s.terminal("Clean"),
		Want:     Want{Error: fmt.Sprintf("match invariant at tick 0: bomb planted by nobody at tick %d", s.tick+10)}})

	// The round opens early at a frame, then reopens at freeze end; a grenade without a weapon in
	// the reopened round is CSC's error.
	s = &sim{tick: 102}
	pistol := s.snapshot(1)
	s.add(fact.RoundStart{Tick: 100})
	s.facts = append(s.facts, Event{match.Event{Fact: fact.Frame{Tick: 102}, Sample: &pistol}})
	s.tick = 400
	s.add(fact.FreezeEnd{Snapshot: s.snapshot(1)})
	s.add(fact.GrenadeThrow{Tick: 500, Thrower: s.player(1)})
	out = append(out, &Status{Name: "grenade-without-weapon-reopened",
		Why:      "an early open, then a freeze-end reopen; a live grenade with no weapon is the CSC step's error",
		Facts:    s.facts,
		Terminal: s.terminal("Truncated"),
		Want:     Want{Error: "match invariant at tick 0: grenade without a weapon at tick 500"}})

	// Non-live facts, lifecycle facts and a clean end.
	s = &sim{}
	rifle := 7
	snap := s.snapshot(1)
	snap.Warmup = true
	s.add(fact.Header{ServerName: "CSC Match - 1 | Alpha vs Bravo"}, fact.ServerInfo{Map: "de_mirage"}, fact.Connect{Tick: 1, Steam64: 1},
		fact.Footstep{Tick: 2}, fact.Hurt{Snapshot: snap, Player: s.player(1), Attacker: s.player(6), Weapon: &rifle, HealthDamageTaken: 10},
		fact.Kill{Snapshot: snap, Killer: s.player(6), Victim: s.player(1), Weapon: &rifle},
		fact.Flashed{Tick: 3, Player: s.player(1), Attacker: s.player(6), Duration: 2})
	s.match13()
	out = append(out, &Status{Name: "ended-with-non-live-facts",
		Why:      "warmup facts before the start, ScoreUpdated and RoundEndOfficial each round, the game end, a clean terminal",
		Facts:    s.facts,
		Terminal: s.terminal("Clean"),
		Want:     Want{Outcome: "Ended"}})

	s = &sim{}
	s.play(3)
	out = append(out, &Status{Name: "truncated-unfinished",
		Why:      "three rounds, then the stream ends before the match does (D8)",
		Facts:    s.facts,
		Terminal: s.terminal("Truncated"),
		Want:     Want{Outcome: "Unfinished", Detail: "demo stopped (truncated) after round 3"}})

	out = append(out, &Status{Name: "invalid-file",
		Why:      "no facts and a file that isn't a demo",
		Terminal: Terminal{Kind: "InvalidFileType", Err: "not a CS2 demo"},
		Want:     Want{Error: "not a CS2 demo"}})

	for _, st := range out {
		st.Version = Version
	}
	return out
}

func TestStatuses(t *testing.T) {
	for _, st := range statuses() {
		t.Run(st.Name, func(t *testing.T) {
			path := filepath.Join("testdata", "status", st.Name+".json.gz")
			want, err := EncodeStatus(st)
			if err != nil {
				t.Fatal(err)
			}
			if *update {
				gz, err := Gzip(want)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, gz, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			raw := read(t, path)
			if !bytes.Equal(raw, want) {
				t.Fatal("the fixture differs from its builder; run go test ./engine/roundlog -run TestStatuses -update")
			}
			got, err := DecodeStatus(raw)
			if err != nil {
				t.Fatal(err)
			}
			res, err := got.Run()
			if err != nil {
				t.Fatal(err)
			}
			if res != got.Want {
				t.Errorf("Finish gave %+v; want %+v", res, got.Want)
			}
		})
	}
}
