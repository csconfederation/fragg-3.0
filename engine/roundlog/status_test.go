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
	alpha, bravo int  // rounds won
	money        *int // every player's money, when not the default
	warmup       bool // the snapshots' Warmup flag
	reason       int  // the next round end's reason, 9 by default
}

// restore sets the game's round counter back (or forward) with the scores of its rounds.
func (s *sim) restore(played, alpha, bravo int) { s.played, s.alpha, s.bravo = played, alpha, bravo }

func (s *sim) setMoney(m int) { s.money = &m }

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
	if s.money != nil {
		money = *s.money
	}
	snap := fact.Snapshot{Tick: s.tick, RoundsPlayed: s.played, Warmup: s.warmup,
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
func (s *sim) open() { s.openWith(false) }

// openWith is open, after a frame that opens the round early when early is set.
func (s *sim) openWith(early bool) {
	s.tick += 100
	s.add(fact.RoundStart{Tick: s.tick, RoundsPlayed: s.played})
	if early {
		snap := s.snapshot(s.played + 1)
		snap.Tick = s.tick + 2
		s.facts = append(s.facts, Event{match.Event{Fact: fact.Frame{Tick: s.tick + 2}, Sample: &snap}})
	}
	s.tick += 100
	s.add(fact.FreezeEnd{Snapshot: s.snapshot(s.played + 1)})
}

// end ends the open round, won by Alpha, with an official end unless it decides the match.
func (s *sim) end(official bool) { s.endBy(official, true) }

func (s *sim) endBy(official, alphaWins bool) {
	num := s.played + 1
	t, ct := "Alpha", "Bravo"
	if num > 12 {
		t, ct = ct, t
	}
	tSide, ctSide := fact.Team{Side: fact.SideT, ID: 2, Clan: t}, fact.Team{Side: fact.SideCT, ID: 3, Clan: ct}
	side, winner, loser := fact.SideT, tSide, ctSide
	if (t == "Alpha") != alphaWins {
		side, winner, loser = fact.SideCT, ctSide, tSide
	}
	s.tick += 1000
	s.played++
	if alphaWins {
		s.alpha++
	} else {
		s.bravo++
	}
	wins := map[string]int{"Alpha": s.alpha, "Bravo": s.bravo}
	winner.Score, loser.Score = wins[winner.Clan], wins[loser.Clan]
	reason := 9
	if s.reason != 0 {
		reason, s.reason = s.reason, 0
	}
	s.add(fact.ScoreUpdated{Tick: s.tick, Team: winner, Opponent: loser})
	s.add(fact.RoundEnd{Snapshot: s.snapshot(num), Winner: side, Reason: reason, WinnerTeam: winner, LoserTeam: loser})
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
	s.deciding()
}

// deciding plays the round that completes Alpha's 13-0, without an official end, then the game end.
func (s *sim) deciding() {
	s.open()
	s.end(false)
	s.gameEnd()
}

func (s *sim) gameEnd() {
	s.add(fact.GamePhase{Tick: s.tick, Phase: fact.GamePhaseEnded}, fact.WinPanelMatch{Tick: s.tick})
}

// playAlt plays n rounds with an official end each, Alpha taking the odd ones.
func (s *sim) playAlt(n int) {
	for i := 0; i < n; i++ {
		s.open()
		s.endBy(true, i%2 == 0)
	}
}

func (s *sim) terminal(kind string) Terminal {
	return Terminal{Kind: kind, End: fact.Snapshot{Tick: s.tick + 500, RoundsPlayed: s.played,
		T: fact.Team{Side: fact.SideT, Score: s.played}, CT: fact.Team{Side: fact.SideCT}}}
}

// statuses are the fixtures. Each Want is written here from the lifecycle's rules (match.keep,
// Finish's precedence), and checked against the build they were added on (parser-v6 eac6b04).
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
		Want:     Want{Outcome: "Ended", Rounds: "1-13", Scores: "Alpha 13, Bravo 0"}})

	s = &sim{}
	s.play(3)
	out = append(out, &Status{Name: "truncated-unfinished",
		Why:      "three rounds, then the stream ends before the match does (D8)",
		Facts:    s.facts,
		Terminal: s.terminal("Truncated"),
		Want:     Want{Outcome: "Unfinished", Detail: "demo stopped (truncated) after round 3", Rounds: "1-3", Scores: "Alpha 3, Bravo 0"}})

	out = append(out, &Status{Name: "invalid-file",
		Why:      "no facts and a file that isn't a demo",
		Terminal: Terminal{Kind: "InvalidFileType", Err: "not a CS2 demo"},
		Want:     Want{Error: "not a CS2 demo"}})

	// ended is what a 13-0 for Alpha gives.
	ended := Want{Outcome: "Ended", Rounds: "1-13", Scores: "Alpha 13, Bravo 0"}
	add := func(name, why string, s *sim, term Terminal, want Want) {
		out = append(out, &Status{Name: name, Why: why, Facts: s.facts, Terminal: term, Want: want})
	}

	// Go-live: the first round open where both first members have 800 money. Before it, rounds don't
	// commit or score; the Warmup flag isn't read (D62).
	s = &sim{}
	s.setMoney(4000)
	s.open()
	s.end(true)
	s.money, s.played, s.alpha = nil, 0, 0
	s.match13()
	add("go-live-first-800", "a played round on 4000 money before the first 800/800 open: no go-live, nothing committed", s, s.terminal("Clean"), ended)

	s = &sim{warmup: true}
	s.match13()
	add("go-live-ignores-warmup", "the 800/800 opens carry the Warmup flag and go live anyway (D62)", s, s.terminal("Clean"), ended)

	s = &sim{}
	s.openWith(true)
	s.end(true)
	s.play(11)
	s.deciding()
	add("go-live-at-early-open", "the first 800/800 open is the early open at a frame; the freeze-end reopen replaces it", s, s.terminal("Clean"), ended)

	// Knife before the match is live: nothing starts, and the next 800/800 open is the go-live.
	s = &sim{}
	s.setMoney(0)
	s.open()
	s.end(true)
	s.money, s.played, s.alpha = nil, 0, 0
	s.match13()
	add("knife-before-live", "a knife round (0/0 at freeze end) before go-live is not a round; the 800/800 open after it goes live", s, s.terminal("Clean"), ended)

	s = &sim{}
	s.setMoney(0)
	s.openWith(true)
	s.end(true)
	s.money, s.played, s.alpha = nil, 0, 0
	s.match13()
	add("knife-before-live-early-open", "as knife-before-live, the knife round also opened early at a frame", s, s.terminal("Clean"), ended)

	// Knife while live (D58): the round being opened is the knife round, and the map waits for an
	// 800/800 open. The knife round, with or without an early open, and the rounds after it don't
	// commit or score; the committed rounds before it stay.
	s = &sim{}
	s.play(3)
	s.setMoney(0)
	s.open()
	s.end(true)
	add("knife-while-live-no-commit", "0/0 at freeze end while live, no early open: the knife round 4 doesn't commit or score", s, s.terminal("Truncated"),
		Want{Outcome: "Unfinished", Detail: "demo stopped (truncated) after round 3", Rounds: "1-3", Scores: "Alpha 3, Bravo 0"})

	s = &sim{}
	s.play(3)
	s.setMoney(0)
	s.openWith(true)
	s.end(true)
	add("knife-while-live-early-open", "0/0 at freeze end while live after an early open: the freeze-end reopen is the knife round, and it doesn't commit", s, s.terminal("Truncated"),
		Want{Outcome: "Unfinished", Detail: "demo stopped (truncated) after round 3", Rounds: "1-3", Scores: "Alpha 3, Bravo 0"})

	s = &sim{}
	s.play(3)
	s.setMoney(0)
	s.open()
	s.end(true)
	s.money = nil
	s.play(2)
	add("knife-while-live-later-rounds", "after a live knife, rounds that don't open at 800/800 money don't commit either", s, s.terminal("Truncated"),
		Want{Outcome: "Unfinished", Detail: "demo stopped (truncated) after round 3", Rounds: "1-3", Scores: "Alpha 3, Bravo 0"})

	s = &sim{}
	s.play(3)
	s.setMoney(0)
	s.open()
	s.end(true)
	s.money = nil
	s.restore(0, 0, 0)
	s.match13()
	add("knife-while-live-restart", "after a live knife the next 800/800 open restarts the match: scores reset, the new rounds supersede the old", s, s.terminal("Clean"), ended)

	// D12: a restore replays from an earlier round; committing round n removes rounds n and up.
	s = &sim{}
	s.play(5)
	s.restore(2, 2, 0)
	s.play(10)
	s.deciding()
	add("restore-supersedes", "rounds 1-5 played, restored to after round 2 and replayed: the replay's rounds replace 3-5 (D12)", s, s.terminal("Clean"), ended)

	s = &sim{}
	s.play(5)
	s.restore(2, 2, 0)
	s.play(1)
	s.restore(4, 4, 0)
	s.play(8)
	s.deciding()
	add("restore-gap-score-only", "the replay commits round 3, then the counter jumps to 4: round 4 is never committed (D9)", s, s.terminal("Clean"),
		Want{Outcome: "ScoreOnly", Detail: "round 4 is missing", Rounds: "1-3,5-13", Scores: "Alpha 13, Bravo 0"})

	// The deciding round won by the bomb closes (D59) at the first of
	// the official end, the win panel and the end of the stream (D15).
	s = &sim{}
	s.play(12)
	s.open()
	s.reason = fact.ReasonTargetBombed
	s.end(true)
	add("bomb-deciding-official-end", "the bomb wins the deciding round; the official end commits it", s, s.terminal("Clean"), ended)

	s = &sim{}
	s.play(12)
	s.open()
	s.reason = fact.ReasonTargetBombed
	s.end(false)
	add("bomb-deciding-stream-end", "the bomb wins the deciding round; no official end or game end, so Finish commits it", s, s.terminal("Clean"), ended)

	s = &sim{}
	s.play(12)
	s.open()
	s.reason = fact.ReasonTargetBombed
	s.end(false)
	s.gameEnd()
	add("bomb-deciding-win-panel", "the bomb wins the deciding round; the win panel commits it", s, s.terminal("Clean"), ended)

	// The game's end can arrive before the final RoundEnd (fact.GamePhase).
	s = &sim{}
	s.play(12)
	s.open()
	s.gameEnd()
	s.end(false)
	add("game-end-before-final-round-end", "game phase Ended and the win panel, then the deciding round's RoundEnd: it still scores and commits", s, s.terminal("Clean"), ended)

	// A game end that isn't the clinch (D56) ends the map: the round in play closes, or is rejected.
	s = &sim{}
	s.play(7)
	s.open()
	s.reason = 18 // CTSurrender
	s.end(false)
	s.gameEnd()
	add("surrender-game-end", "CT surrenders in round 8 of an undecided match: the game ends, and round 8 closes at the win panel", s, s.terminal("Clean"),
		Want{Outcome: "Ended", Rounds: "1-8", Scores: "Alpha 8, Bravo 0"})

	s = &sim{}
	s.play(7)
	s.add(fact.GamePhase{Tick: s.tick, Phase: fact.GamePhaseEnded})
	add("admin-end-game-phase", "game phase Ended between rounds of an undecided match, no win panel: Ended", s, s.terminal("Clean"),
		Want{Outcome: "Ended", Rounds: "1-7", Scores: "Alpha 7, Bravo 0"})

	s = &sim{}
	s.play(7)
	s.open()
	s.add(fact.GamePhase{Tick: s.tick, Phase: fact.GamePhaseEnded})
	add("game-end-open-round-rejected", "game phase Ended in an open round that never gets its RoundEnd: the round is rejected, the map is Ended", s, s.terminal("Truncated"),
		Want{Outcome: "Ended", Rounds: "1-7", Scores: "Alpha 7, Bravo 0"})

	s = &sim{}
	s.play(7)
	s.open()
	s.add(fact.GamePhase{Tick: s.tick, Phase: fact.GamePhaseEnded})
	s.restore(0, 0, 0)
	s.play(2)
	add("game-end-open-round-replaced", "game phase Ended in an open round, then a new round opens: the open round is rejected, the map stays Ended and the new rounds are ignored", s, s.terminal("Clean"),
		Want{Outcome: "Ended", Rounds: "1-7", Scores: "Alpha 7, Bravo 0"})

	// Ties (D63): the game ends at a level score, which ends the map; the last round has no official end. A Combine can end
	// 15-15 after one overtime (D63).
	tied := func(s *sim) Terminal {
		t := s.terminal("Clean")
		t.End.T.Score, t.End.CT.Score = 15, 15
		return t
	}
	s = &sim{}
	s.playAlt(29)
	s.open()
	s.endBy(false, false)
	s.gameEnd()
	add("tie-game-end", "15-15 after round 30, game end, no official end for round 30: the tie fallback commits it", s, tied(s),
		Want{Outcome: "Ended", Rounds: "1-30", Scores: "Alpha 15, Bravo 15"})

	s = &sim{}
	s.playAlt(29)
	s.open()
	s.endBy(false, false)
	add("tie-without-game-end", "the same level score with no game end: not a tie, the demo just stopped", s, tied(s),
		Want{Outcome: "Unfinished", Detail: "demo stopped (clean) after round 29", Rounds: "1-29", Scores: "Alpha 15, Bravo 15"})

	// Streams that end badly.
	s = &sim{}
	s.playAlt(29)
	s.open()
	s.endBy(false, false)
	s.gameEnd()
	term := tied(s)
	term.Kind, term.Err = "Decode", "decode failed after the game end"
	add("tie-game-end-then-decode-error", "the tie facts, then a decode failure: the game had ended, so the map is Ended (D56)", s, term,
		Want{Outcome: "Ended", Rounds: "1-30", Scores: "Alpha 15, Bravo 15"})

	s = &sim{}
	s.play(3)
	term = s.terminal("Decode")
	term.Err = "decode failed mid-match"
	add("decode-mid-match", "three rounds, then a decode failure before the end", s, term, Want{Error: "decode failed mid-match"})

	s = &sim{}
	s.match13()
	term = s.terminal("Decode")
	term.Err = "decode failed after the clinch"
	add("decode-after-clinch", "the match ended by the clinch, then a decode failure: still Ended", s, term, ended)

	s = &sim{}
	s.open()
	add("truncated-no-rounds", "go-live and one open round, then the stream stops: nothing committed", s, s.terminal("Truncated"),
		Want{Outcome: "NoRounds", Scores: "Alpha 0, Bravo 0"})

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
