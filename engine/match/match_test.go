package match

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/csconfederation/fragg-3.0/engine/fact"
)

// sim builds the facts of a two-team match in CS2 dispatch order. Alpha starts T; sides swap after
// round 12 and every 3 OT rounds.
type sim struct {
	facts        []fact.Fact
	tick, played int
	score        map[string]int
	money        int // money+spent at the next open; 0 means 800 on pistol rounds, 4000 otherwise
}

func newSim() *sim { return &sim{score: map[string]int{"Alpha": 0, "Bravo": 0}} }

func (s *sim) sides(num int) (t, ct string) {
	swap := num > mr && num <= mr*2
	if num > mr*2 {
		swap = (num-mr*2-1)/3%2 == 0
	}
	if swap {
		return "Bravo", "Alpha"
	}
	return "Alpha", "Bravo"
}

func (s *sim) snapshot(num int) fact.Snapshot {
	t, ct := s.sides(num)
	money := s.money
	if money == 0 {
		money = 4000
		if num == 1 || num == mr+1 {
			money = 800
		}
	}
	snap := fact.Snapshot{Tick: s.tick, RoundsPlayed: s.played,
		T:  fact.Team{Side: fact.SideT, ID: 2, Clan: t, Score: s.score[t]},
		CT: fact.Team{Side: fact.SideCT, ID: 3, Clan: ct, Score: s.score[ct]}}
	for i := uint64(1); i <= 10; i++ {
		side := fact.SideT
		if (i > 5) != (t == "Bravo") {
			side = fact.SideCT
		}
		snap.Players = append(snap.Players, fact.Player{Steam64: i, Name: string(rune('a' + i)), Side: side,
			Alive: true, Connected: true, Listed: true, Money: money, Spent: 0})
	}
	return snap
}

func (s *sim) add(f ...fact.Fact) { s.facts = append(s.facts, f...) }

// open starts round num at freeze end.
func (s *sim) open() int {
	num := s.played + 1
	s.tick += 100
	s.add(fact.RoundStart{Tick: s.tick, RoundsPlayed: s.played})
	s.tick += 100
	s.add(fact.FreezeEnd{Snapshot: s.snapshot(num)})
	return num
}

// end ends the open round won by clan, with the score update first (CS2 order).
func (s *sim) end(winner string, official bool) {
	num := s.played + 1
	t, ct := s.sides(num)
	loser, side := "Alpha", fact.SideT
	if winner == "Alpha" {
		loser = "Bravo"
	}
	if winner == ct {
		side = fact.SideCT
	}
	s.tick += 1000
	s.score[winner]++
	s.played++
	s.add(fact.ScoreUpdated{Tick: s.tick, Team: s.team(winner, t), Opponent: s.team(loser, t)})
	s.add(fact.RoundEnd{Tick: s.tick, RoundsPlayed: s.played, Winner: side, Reason: 9,
		WinnerTeam: s.team(winner, t), LoserTeam: s.team(loser, t)})
	if official {
		s.tick += 300
		s.add(fact.RoundEndOfficial{Tick: s.tick, RoundsPlayed: s.played})
	}
}

func (s *sim) team(clan, tClan string) fact.Team {
	if clan == tClan {
		return fact.Team{Side: fact.SideT, ID: 2, Clan: clan, Score: s.score[clan]}
	}
	return fact.Team{Side: fact.SideCT, ID: 3, Clan: clan, Score: s.score[clan]}
}

// play opens and ends a round; the official end is skipped once a team reaches 13 in regulation,
// as the game does for the deciding round.
func (s *sim) play(winner string) {
	s.open()
	s.end(winner, true)
}

func (s *sim) gameEnded() {
	s.add(fact.GamePhase{Tick: s.tick, Phase: fact.GamePhaseEnded}, fact.WinPanelMatch{Tick: s.tick})
}

func (s *sim) run(t *testing.T, term fact.Terminal) (*Result, error) {
	t.Helper()
	m := New()
	for _, f := range s.facts {
		if err := m.Fact(f); err != nil {
			return m.Finish(term)
		}
	}
	term.End = fact.Snapshot{Tick: s.tick + 500, RoundsPlayed: s.played,
		T: fact.Team{Side: fact.SideT, Score: s.score["Alpha"]}, CT: fact.Team{Side: fact.SideCT, Score: s.score["Bravo"]}}
	return m.Finish(term)
}

// frame is a Frame whose sample is s.
func frame(s fact.Snapshot) fact.Frame {
	return fact.Frame{Tick: s.Tick, Frame: s.Frame, Sample: func() fact.Snapshot { return s }}
}

func nums(rs []*Round) []int {
	var out []int
	for _, r := range rs {
		out = append(out, r.Num)
	}
	return out
}

func want(t *testing.T, res *Result, err error, outcome Outcome, rounds int) {
	t.Helper()
	if err != nil {
		t.Fatalf("error %v", err)
	}
	if res.Outcome != outcome || len(res.Rounds) != rounds {
		t.Fatalf("outcome %v (%s), %d rounds %v; want %v, %d", res.Outcome, res.Detail, len(res.Rounds), nums(res.Rounds), outcome, rounds)
	}
}

// regulation plays a 13-n win for Alpha.
func regulation(s *sim, alphaLoses int) {
	for i := 0; i < 13+alphaLoses; i++ {
		w := "Alpha"
		if i < alphaLoses {
			w = "Bravo"
		}
		if i == 13+alphaLoses-1 {
			s.open()
			s.end(w, false)
			s.gameEnded()
			return
		}
		s.play(w)
	}
}

func TestDecidedWin(t *testing.T) {
	s := newSim()
	s.add(fact.Header{ServerName: "CSC Match - 9984 | de_cache Map 0 | A vs B | Powered by Dathost"}, fact.ServerInfo{Map: "de_cache"})
	regulation(s, 4)
	res, err := s.run(t, fact.Terminal{Kind: fact.Clean})
	want(t, res, err, Ended, 17)
	if res.Scores["Alpha"] != 13 || res.Scores["Bravo"] != 4 || res.WinnerClan != "Alpha" || res.TotalRounds != 17 {
		t.Errorf("scores %v winner %q total %d", res.Scores, res.WinnerClan, res.TotalRounds)
	}
	if res.HeaderMatchID != 9984 || res.Map != "de_cache" {
		t.Errorf("header id %d map %q", res.HeaderMatchID, res.Map)
	}
	if r := res.Rounds[0]; !r.Pistol || len(r.Members) != 10 || r.InitT != 5 || r.Members[1].Clan != "Alpha" || r.Members[1].Side != fact.SideT {
		t.Errorf("round 1 %+v", r)
	}
	if r := res.Rounds[12]; r.Members[1].Side != fact.SideCT || !r.Pistol {
		t.Errorf("round 13 after the swap: %+v", r.Members[1])
	}
}

func TestKnifeRoundBeforeLive(t *testing.T) {
	s := newSim()
	knife := s.snapshot(1)
	for i := range knife.Players {
		knife.Players[i].Money = 0
	}
	s.add(fact.RoundStart{Tick: 50}, fact.FreezeEnd{Snapshot: knife})
	s.tick += 1000
	s.played = 1
	s.add(fact.RoundEnd{Tick: s.tick, RoundsPlayed: 1, Winner: fact.SideT, WinnerTeam: fact.Team{Side: fact.SideT, Clan: "Alpha"}})
	s.played = 0 // the game restarts for the live match
	regulation(s, 0)
	res, err := s.run(t, fact.Terminal{Kind: fact.Clean})
	want(t, res, err, Ended, 13)
}

func TestOvertimeWin(t *testing.T) {
	s := newSim()
	for i := 0; i < 24; i++ {
		w := "Alpha"
		if i%2 == 1 {
			w = "Bravo"
		}
		s.play(w)
	}
	for i := 0; i < 4; i++ { // 16-12 in OT1: Alpha takes 4 of 4
		if i == 3 {
			s.open()
			s.end("Alpha", false)
			s.gameEnded()
			break
		}
		s.play("Alpha")
	}
	res, err := s.run(t, fact.Terminal{Kind: fact.Clean})
	want(t, res, err, Ended, 28)
	if res.Scores["Alpha"] != 16 || res.WinnerClan != "Alpha" {
		t.Errorf("scores %v winner %q", res.Scores, res.WinnerClan)
	}
}

func tied(s *sim) {
	for i := 0; i < 30; i++ {
		w := "Alpha"
		if i%2 == 1 {
			w = "Bravo"
		}
		if i == 29 {
			s.open()
			s.end(w, false) // the game doesn't send RoundEndOfficial for the last round
			return
		}
		s.play(w)
	}
}

func TestTieNeedsGameEnded(t *testing.T) {
	s := newSim()
	tied(s)
	s.gameEnded()
	res, err := s.run(t, fact.Terminal{Kind: fact.Clean})
	want(t, res, err, Ended, 30)
	if res.WinnerClan != "" || res.Scores["Alpha"] != 15 {
		t.Errorf("winner %q scores %v", res.WinnerClan, res.Scores)
	}

	s = newSim()
	tied(s)
	res, err = s.run(t, fact.Terminal{Kind: fact.Truncated})
	want(t, res, err, Unfinished, 29)
}

func TestGameEndedBeforeDecidingRoundEnd(t *testing.T) {
	s := newSim()
	for i := 0; i < 12; i++ {
		s.play("Alpha")
	}
	s.open()
	s.gameEnded() // the phase change arrives before the deciding RoundEnd
	s.end("Alpha", false)
	res, err := s.run(t, fact.Terminal{Kind: fact.Clean})
	want(t, res, err, Ended, 13)
}

func TestTerminals(t *testing.T) {
	half := func() *sim {
		s := newSim()
		for i := 0; i < 6; i++ {
			s.play("Alpha")
		}
		return s
	}
	res, err := half().run(t, fact.Terminal{Kind: fact.Truncated})
	want(t, res, err, Unfinished, 6)
	res, err = half().run(t, fact.Terminal{Kind: fact.Clean})
	want(t, res, err, Unfinished, 6)

	decode := errors.New("decode")
	if _, err := half().run(t, fact.Terminal{Kind: fact.Decode, Err: decode}); !errors.Is(err, decode) {
		t.Errorf("decode before the end: error %v", err)
	}
	s := newSim()
	regulation(s, 0)
	res, err = s.run(t, fact.Terminal{Kind: fact.Decode, Err: decode})
	want(t, res, err, Ended, 13)

	invalid := errors.New("invalid")
	if _, err := newSim().run(t, fact.Terminal{Kind: fact.InvalidFileType, Err: invalid}); !errors.Is(err, invalid) {
		t.Errorf("invalid file: error %v", err)
	}
	res, err = newSim().run(t, fact.Terminal{Kind: fact.Clean})
	want(t, res, err, NoRounds, 0)
}

func TestPartial(t *testing.T) {
	// A recording that starts at round 5 (restored server): the score is the game's.
	s := newSim()
	s.played, s.score["Alpha"], s.score["Bravo"] = 4, 2, 2
	s.money = 3000
	for i := 0; i < 11; i++ {
		if i == 10 {
			s.open()
			s.end("Alpha", false)
			s.gameEnded()
			break
		}
		s.play("Alpha")
	}
	res, err := s.run(t, fact.Terminal{Kind: fact.Clean})
	// No 800/800 open, so the match never goes live: the oracle needs a pistol round to start.
	want(t, res, err, NoRounds, 0)

	// The same restore going live (pistol money at the first open): rounds 5..13 commit.
	s = newSim()
	s.played, s.score["Alpha"] = 4, 4
	s.money = 800
	for i := 0; i < 9; i++ {
		if i == 8 {
			s.open()
			s.end("Alpha", false)
			s.gameEnded()
			break
		}
		s.play("Alpha")
	}
	res, err = s.run(t, fact.Terminal{Kind: fact.Clean})
	want(t, res, err, Partial, 9)
	if !strings.Contains(res.Detail, "first committed round is 5") || res.Scores["Alpha"] != 13 {
		t.Errorf("detail %q scores %v", res.Detail, res.Scores)
	}

	// Rounds 1..13 commit, but the game's score counts a round the demo doesn't hold.
	s = newSim()
	s.score["Bravo"] = 1
	regulation(s, 0)
	res, err = s.run(t, fact.Terminal{Kind: fact.Clean})
	want(t, res, err, Partial, 13)
	if !strings.Contains(res.Detail, "scores sum to 14, but 13 rounds committed") {
		t.Errorf("detail %q", res.Detail)
	}

	// A round that fails integrity leaves a gap.
	s = newSim()
	for i := 0; i < 13; i++ {
		if i == 5 {
			s.open()
			s.end("Alpha", false)
			s.add(fact.RoundEndOfficial{Tick: s.tick + 1, RoundsPlayed: s.played + 1}) // official for the wrong round
			continue
		}
		if i == 12 {
			s.open()
			s.end("Alpha", false)
			s.gameEnded()
			break
		}
		s.play("Alpha")
	}
	res, err = s.run(t, fact.Terminal{Kind: fact.Clean})
	want(t, res, err, Partial, 12)
	if !strings.Contains(res.Detail, "round 6 is missing") {
		t.Errorf("detail %q", res.Detail)
	}
}

func TestScoreNormalization(t *testing.T) {
	// Without ScoreUpdated, the winner's tracked score goes up by one.
	s := newSim()
	for i := 0; i < 13; i++ {
		s.open()
		s.tick += 1000
		s.played++
		tClan, _ := s.sides(s.played)
		s.score["Alpha"]++
		s.add(fact.RoundEnd{Tick: s.tick, RoundsPlayed: s.played, Winner: s.team("Alpha", tClan).Side,
			WinnerTeam: s.team("Alpha", tClan), LoserTeam: s.team("Bravo", tClan)})
		if i < 12 {
			s.add(fact.RoundEndOfficial{Tick: s.tick + 1, RoundsPlayed: s.played})
		}
	}
	res, err := s.run(t, fact.Terminal{Kind: fact.Clean})
	if err != nil {
		t.Fatal(err)
	}
	if res.Outcome != Ended || res.Scores["Alpha"] != 13 || res.Scores["Bravo"] != 0 {
		t.Errorf("outcome %v (%s), scores %v", res.Outcome, res.Detail, res.Scores)
	}
}

func TestLateScoreUpdated(t *testing.T) {
	// The score update arrives after the round end: the round end counts the win, the update sets
	// the same score, so nothing is counted twice.
	s := newSim()
	for i := 0; i < 13; i++ {
		s.open()
		s.tick += 1000
		s.played++
		tClan, _ := s.sides(s.played)
		s.score["Alpha"]++
		s.add(fact.RoundEnd{Tick: s.tick, RoundsPlayed: s.played, Winner: s.team("Alpha", tClan).Side,
			WinnerTeam: s.team("Alpha", tClan), LoserTeam: s.team("Bravo", tClan)},
			fact.ScoreUpdated{Tick: s.tick, Team: s.team("Alpha", tClan), Opponent: s.team("Bravo", tClan)})
		if i < 12 {
			s.add(fact.RoundEndOfficial{Tick: s.tick + 1, RoundsPlayed: s.played})
		}
	}
	res, err := s.run(t, fact.Terminal{Kind: fact.Clean})
	want(t, res, err, Ended, 13)
	if res.Scores["Alpha"] != 13 || res.Scores["Bravo"] != 0 {
		t.Errorf("scores %v", res.Scores)
	}
}

func TestWinnerMustBeARoundTeam(t *testing.T) {
	s := newSim()
	s.play("Alpha")
	s.open()
	s.tick += 1000
	s.played++
	// The game names a winner that isn't either team of this round (but is in the match's map).
	s.add(fact.RoundEnd{Tick: s.tick, RoundsPlayed: s.played, Winner: fact.SideT,
		WinnerTeam: fact.Team{Side: fact.SideT, Clan: "Alpha"}, LoserTeam: fact.Team{Side: fact.SideCT, Clan: "Bravo"}},
		fact.RoundEndOfficial{Tick: s.tick + 1, RoundsPlayed: s.played})
	m := New()
	var err error
	for _, f := range s.facts[:len(s.facts)-2] {
		if err = m.Fact(f); err != nil {
			t.Fatal(err)
		}
	}
	m.potential.sideClans[fact.SideT] = "StartedT" // the round opened with an empty clan name
	for _, f := range s.facts[len(s.facts)-2:] {
		err = m.Fact(f)
	}
	var inv *InvariantError
	if !errors.As(err, &inv) || !strings.Contains(inv.Reason, "isn't one of its teams") {
		t.Errorf("error %v", err)
	}
}

func TestInvariantWinsOverTruncation(t *testing.T) {
	s := newSim()
	for i := 0; i < 5; i++ {
		s.play("Alpha")
	}
	s.played = 2
	s.play("Bravo")
	_, err := s.run(t, fact.Terminal{Kind: fact.Truncated, Err: errors.New("eof")})
	var inv *InvariantError
	if !errors.As(err, &inv) {
		t.Errorf("error %v, want the invariant", err)
	}
}

func TestReconnectReplacesListedRecord(t *testing.T) {
	m := New()
	m.reconnectedOracle[1] = true
	s := fact.Snapshot{Players: []fact.Player{
		{Steam64: 1, Side: fact.SideT, Listed: true, Money: 800},
		{Steam64: 2, Side: fact.SideT, Listed: true, Money: 900},
		{Steam64: 1, Side: fact.SideT, Listed: false, Money: 1000}, // the new record after a reconnect
		{Steam64: 3, Side: fact.SideCT, Listed: true},
	}}
	got := m.members(s, fact.SideT)
	if len(got) != 2 || got[0].Steam64 != 2 || got[1].Steam64 != 1 || got[1].Money != 1000 {
		t.Errorf("members %+v", got)
	}
	m.connectedAfterStart[2] = true
	if got := m.members(s, fact.SideT); len(got) != 1 || got[0].Steam64 != 1 {
		t.Errorf("members after a late connect %+v", got)
	}
}

func TestPlayerInfoMarksLateConnect(t *testing.T) {
	s := newSim()
	s.open()
	m := New()
	for _, f := range append(s.facts, fact.PlayerInfo{Found: true, Resolved: 3}) {
		if err := m.Fact(f); err != nil {
			t.Fatal(err)
		}
	}
	if !m.connectedAfterStart[3] || !m.reconnectedOracle[3] {
		t.Errorf("late connect %v reconnected %v", m.connectedAfterStart, m.reconnectedOracle)
	}
	// Until the next round opens, a late connect is left out of both member sources.
	got := m.members(s.snapshot(1), fact.SideT)
	if len(got) != 4 || slices.ContainsFunc(got, func(p fact.Player) bool { return p.Steam64 == 3 }) {
		t.Errorf("members %+v", got)
	}
}

func TestRoundRegressionIsInvariant(t *testing.T) {
	s := newSim()
	for i := 0; i < 5; i++ {
		s.play("Alpha")
	}
	s.played = 2 // an in-demo restore back to round 3
	s.play("Bravo")
	_, err := s.run(t, fact.Terminal{Kind: fact.Clean})
	var inv *InvariantError
	if !errors.As(err, &inv) || !strings.Contains(inv.Reason, "round 3 committed after round 5") {
		t.Errorf("error %v", err)
	}
}

func TestGameEndedWithoutOurEndIsInvariant(t *testing.T) {
	s := newSim()
	for i := 0; i < 6; i++ {
		s.play("Alpha")
	}
	s.gameEnded() // e.g. a surrender
	_, err := s.run(t, fact.Terminal{Kind: fact.Clean})
	var inv *InvariantError
	if !errors.As(err, &inv) {
		t.Errorf("error %v", err)
	}
}

func TestEarlyOpen(t *testing.T) {
	// The round opens at a frame in the window; with no freeze end, that round commits.
	s := newSim()
	s.tick = 102
	snap := s.snapshot(1)
	s.add(fact.RoundStart{Tick: 100}, frame(fact.Snapshot{Tick: 101}), frame(snap))
	s.end("Alpha", true)
	m := New()
	for _, f := range s.facts {
		if err := m.Fact(f); err != nil {
			t.Fatal(err)
		}
	}
	if len(m.rounds) != 1 || m.rounds[0].StartTick != snap.Tick || !m.rounds[0].Pistol {
		t.Fatalf("rounds %+v", m.rounds)
	}
	// A freeze end after the early open reopens the round; that one commits.
	s = newSim()
	s.tick = 102
	early := s.snapshot(1)
	s.add(fact.RoundStart{Tick: 100}, frame(early))
	s.tick = 400
	s.add(fact.FreezeEnd{Snapshot: s.snapshot(1)})
	s.end("Alpha", true)
	m = New()
	for _, f := range s.facts {
		if err := m.Fact(f); err != nil {
			t.Fatal(err)
		}
	}
	if len(m.rounds) != 1 || m.rounds[0].StartTick != 400 {
		t.Fatalf("rounds %+v", m.rounds)
	}

	// A frame after the window doesn't open.
	m = New()
	late := s.snapshot(1)
	late.Tick = 100 + fact.RoundStartWindow
	for _, f := range []fact.Fact{fact.RoundStart{Tick: 100}, frame(late)} {
		_ = m.Fact(f)
	}
	if m.isGameLive {
		t.Error("a frame outside the window opened the match")
	}
}

func TestMidRoundJoin(t *testing.T) {
	s := newSim()
	s.open()
	s.add(
		fact.Footstep{Tick: s.tick},
		// An absent player joins CT alive: added with the event's team.
		fact.TeamChange{Tick: s.tick, Player: fact.Player{Steam64: 42, Name: "late", Alive: true}, NewSide: fact.SideCT,
			NewTeam: fact.Team{Side: fact.SideCT, ID: 3, Clan: "Bravo"}},
		// An existing T player switching to CT keeps their entry.
		fact.TeamChange{Tick: s.tick, Player: fact.Player{Steam64: 1, Name: "b", Alive: true}, NewSide: fact.SideCT,
			NewTeam: fact.Team{Side: fact.SideCT, ID: 3, Clan: "Bravo"}},
	)
	s.end("Alpha", true)
	m := New()
	for _, f := range s.facts {
		if err := m.Fact(f); err != nil {
			t.Fatal(err)
		}
	}
	r := m.rounds[0]
	if got := r.Members[42]; got.Side != fact.SideCT || got.Clan != "Bravo" {
		t.Errorf("late joiner %+v", got)
	}
	if got := r.Members[1]; got.Side != fact.SideT || got.Clan != "Alpha" {
		t.Errorf("side switcher %+v", got)
	}
}

func TestReconnect(t *testing.T) {
	m := New()
	for _, f := range []fact.Fact{
		fact.Connect{Steam64: 7},
		fact.Disconnect{Steam64: 7},
		fact.Connect{Steam64: 7},
		fact.Disconnect{Steam64: 8},
	} {
		_ = m.Fact(f)
	}
	res, _ := m.Finish(fact.Terminal{Kind: fact.Clean})
	if !res.Reconnected[7] || res.Reconnected[8] || len(res.Reconnected) != 1 {
		t.Errorf("reconnected %v", res.Reconnected)
	}
}

// recorder logs the lifecycle calls a collector sees.
type recorder struct{ log []string }

func (c *recorder) Open(*Match, *Round, fact.Snapshot, bool) error { return nil }
func (c *recorder) RoundEnd(*Match, fact.RoundEnd) error           { return nil }
func (c *recorder) Win(*Match, *Round, fact.RoundEnd) error        { return nil }
func (c *recorder) Join(*Match, *Round, Member) error              { return nil }
func (c *recorder) Commit(_ *Match, r *Round, tick int) error {
	c.log = append(c.log, "commit")
	return nil
}
func (c *recorder) Fact(_ *Match, f fact.Fact) error {
	if h, ok := f.(fact.Hurt); ok {
		c.log = append(c.log, fmt.Sprintf("hurt %d", *h.Weapon))
	}
	return nil
}

// The bomb's damage arrives after the deciding round's RoundEnd; the round commits after it, at the
// win panel or, without one, at the demo's end. Other late facts, such as world kills on the win
// panel, stay out (D15).
func TestDecidingRoundCommitsAfterLateFacts(t *testing.T) {
	for _, tc := range []struct {
		name     string
		winPanel bool
		term     fact.TerminalKind
	}{{"win panel", true, fact.Clean}, {"stream error, no win panel", false, fact.Decode}} {
		t.Run(tc.name, func(t *testing.T) {
			s := newSim()
			for i := 0; i < 12; i++ {
				s.play("Alpha")
			}
			s.open()
			s.end("Alpha", false)
			end := s.facts[len(s.facts)-1].(fact.RoundEnd)
			end.Reason = reasonTargetBombed
			s.facts[len(s.facts)-1] = end
			bomb, world := eqBomb, 407
			s.add(fact.Hurt{Snapshot: fact.Snapshot{Tick: s.tick + 18}, Weapon: &bomb},
				fact.Hurt{Snapshot: fact.Snapshot{Tick: s.tick + 100}, Weapon: &world})
			if tc.winPanel {
				s.tick += 200
				s.gameEnded()
			}
			rec := &recorder{}
			m := New(rec)
			for _, f := range s.facts {
				if err := m.Fact(f); err != nil {
					t.Fatal(err)
				}
			}
			res, err := m.Finish(fact.Terminal{Kind: tc.term, Err: errors.New("stream"),
				End: fact.Snapshot{Tick: s.tick + 500, RoundsPlayed: 13}})
			want(t, res, err, Ended, 13)
			if got := strings.Join(rec.log[len(rec.log)-2:], ","); got != "hurt 404,commit" {
				t.Errorf("last calls %v; want the hurt before the commit", rec.log[len(rec.log)-2:])
			}
			if r := res.Rounds[12]; r.EndTick != end.Tick {
				t.Errorf("end tick %d; want the RoundEnd's %d", r.EndTick, end.Tick)
			}
		})
	}
}
