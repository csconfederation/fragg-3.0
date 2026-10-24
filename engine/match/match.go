// Package match runs the round lifecycle over facts: when the match goes live, which rounds commit,
// how the match ends, and the outcome (Core-planning#301, engine core design). The lifecycle ports
// the oracle's (fragg csc/parser.go at tag oracle-v6port-rc.1) so rounds, sides, scores, and
// player sets match it; the outcome rules (D8, D9, D12) are new.
package match

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/csconfederation/fragg-3.0/engine/fact"
)

// mr is half the regulation rounds (MR12).
const mr = 12

// Outcome is what the demo holds.
type Outcome uint8

const (
	Ended      Outcome = iota + 1 // the match ended and rounds 1..N committed
	Partial                       // the match ended, but the committed rounds don't cover it (D9)
	Unfinished                    // rounds were played, but the demo stopped before the match ended (D8)
	NoRounds                      // nothing committed
)

func (o Outcome) String() string {
	return [...]string{"", "Ended", "Partial", "Unfinished", "NoRounds"}[o]
}

// InvariantError is a state the lifecycle doesn't support. The worker reports it as a 500.
type InvariantError struct {
	Tick   int
	Reason string
}

func (e *InvariantError) Error() string {
	return fmt.Sprintf("match invariant at tick %d: %s", e.Tick, e.Reason)
}

// Member is a player in a round, as the round opened (or joined it alive).
type Member struct {
	Steam64 uint64
	Name    string
	IsBot   bool
	Side    fact.Side
	TeamID  int
	Clan    string
}

// Round is one round the lifecycle opened.
type Round struct {
	Num          int
	StartTick    int
	EndTick      int
	Winner       fact.Side
	WinnerClan   string
	Reason       int
	Knife        bool
	Pistol       bool
	InitT        int
	InitCT       int
	Members      map[uint64]Member
	committed    bool
	sideClans    map[fact.Side]string
	integrityEnd bool
}

// Result is the lifecycle's output.
type Result struct {
	Outcome Outcome
	// Detail says why the outcome isn't Ended (the partial or unfinished reason).
	Detail string
	// Rounds are the committed rounds in order.
	Rounds []*Round
	// Scores are the team scores at the decision, by clan.
	Scores      map[string]int
	WinnerClan  string
	TotalRounds int
	Map         string
	ServerName  string
	// HeaderMatchID is the match ID in the server name, 0 if none (D11).
	HeaderMatchID int
	// Reconnected holds players who connected again after disconnecting.
	Reconnected map[uint64]bool
	GameEnded   bool
}

// Match is a fact.Sink. Feed it every fact, then call Finish once.
type Match struct {
	hasGameStarted, isGameLive, inRound, postWinCon bool
	didRoundEndFire, didScoreUpdateFire, haveInit   bool
	integrityStart, integrityEnd, integrityOfficial int
	roundStartedAt                                  int
	result                                          string
	winnerClan                                      string
	totalRounds                                     int
	teams                                           map[string]int // clan -> tracked score; nil before the first start
	potential                                       *Round
	rounds                                          []*Round // in commit order
	connectedAfterStart                             map[uint64]bool
	reconnectedOracle                               map[uint64]bool // the oracle's ReconnectedPlayers, which drives membership
	disconnected, reconnected                       map[uint64]bool
	gameEnded                                       bool
	mapName, serverName                             string
	err                                             *InvariantError
}

// New returns a lifecycle before any fact.
func New() *Match {
	return &Match{
		integrityStart: -1, integrityEnd: -1, integrityOfficial: -1,
		potential:           &Round{Members: map[uint64]Member{}},
		connectedAfterStart: map[uint64]bool{},
		reconnectedOracle:   map[uint64]bool{},
		disconnected:        map[uint64]bool{},
		reconnected:         map[uint64]bool{},
	}
}

func (m *Match) fail(tick int, format string, a ...any) error {
	if m.err == nil {
		m.err = &InvariantError{Tick: tick, Reason: fmt.Sprintf(format, a...)}
	}
	return m.err
}

// Fact applies one fact. It returns an *InvariantError on an unsupported state, after which the
// lifecycle ignores further facts.
func (m *Match) Fact(f fact.Fact) error {
	if m.err != nil {
		return m.err
	}
	switch f := f.(type) {
	case fact.Header:
		m.serverName = f.ServerName
	case fact.ServerInfo:
		m.mapName = f.Map
	case fact.RoundStart:
		m.roundStartedAt = f.Tick
	case fact.Frame:
		if m.roundStartedAt > 0 && m.roundStartedAt+fact.RoundStartWindow > f.Tick && !m.haveInit {
			return m.open(f.Snapshot, false)
		}
	case fact.FreezeEnd:
		return m.open(f.Snapshot, true)
	case fact.RoundEnd:
		return m.roundEnd(f)
	case fact.ScoreUpdated:
		if m.isGameLive {
			m.didScoreUpdateFire = true
			team, err := m.teamName(f.Tick, f.Team.Clan, f.Team.Side)
			if err != nil {
				return err
			}
			opp, err := m.teamName(f.Tick, f.Opponent.Clan, f.Opponent.Side)
			if err != nil {
				return err
			}
			if err := m.setScore(f.Tick, team, f.Team.Score); err != nil {
				return err
			}
			return m.setScore(f.Tick, opp, f.Opponent.Score)
		}
	case fact.RoundEndOfficial:
		if !m.didRoundEndFire {
			m.integrityEnd--
		}
		if m.isGameLive && m.integrityEnd == f.RoundsPlayed {
			return m.final(f.Tick, f.RoundsPlayed, false)
		}
	case fact.TeamChange:
		if m.isGameLive && m.inRound && f.NewSide > fact.SideSpectator {
			if _, ok := m.potential.Members[f.Player.Steam64]; !ok && f.Player.Alive {
				clan, err := m.teamName(f.Tick, f.NewTeam.Clan, f.NewTeam.Side)
				if err != nil {
					return err
				}
				m.potential.Members[f.Player.Steam64] = Member{Steam64: f.Player.Steam64, Name: f.Player.Name,
					IsBot: f.Player.IsBot, Side: f.NewTeam.Side, TeamID: f.NewTeam.ID, Clan: clan}
			}
		}
	case fact.PlayerInfo:
		if f.Found {
			m.reconnectedOracle[f.Resolved] = true
			if m.inRound && m.isGameLive {
				m.connectedAfterStart[f.Resolved] = true
			}
		}
	case fact.Connect:
		if m.disconnected[f.Steam64] {
			m.reconnected[f.Steam64] = true
		}
	case fact.Disconnect:
		if m.reconnectedOracle[f.Steam64] {
			m.reconnectedOracle[f.Steam64] = false
		}
		m.disconnected[f.Steam64] = true
	case fact.Footstep:
		if m.isGameLive {
			m.inRound = true
		}
	case fact.GamePhase:
		if f.Phase == fact.GamePhaseEnded {
			m.gameEnded = true
		}
	case fact.WinPanelMatch:
		m.gameEnded = true
	}
	return nil
}

// open is the oracle's round open: at a frame within the early window, or at freeze end (which
// also detects knife rounds and always reopens).
func (m *Match) open(s fact.Snapshot, freezeEnd bool) error {
	pistol := false
	t, ct := m.members(s, fact.SideT), m.members(s, fact.SideCT)
	if len(t) != 0 && len(ct) != 0 {
		tMoney, ctMoney := t[0].Money+t[0].Spent, ct[0].Money+ct[0].Spent
		switch {
		case tMoney == 800 && ctMoney == 800:
			if !m.hasGameStarted {
				if err := m.startGame(s); err != nil {
					return err
				}
			}
			pistol = true
		case freezeEnd && tMoney == 0 && ctMoney == 0:
			m.potential.Knife = true
			m.hasGameStarted = false
		}
	}
	if !m.isGameLive {
		return nil
	}
	if freezeEnd {
		m.inRound = true
	}
	if err := m.initRound(s); err != nil {
		return err
	}
	if !freezeEnd {
		m.haveInit = true
	}
	m.potential.Pistol = pistol
	return nil
}

func (m *Match) startGame(s fact.Snapshot) error {
	m.hasGameStarted, m.isGameLive = true, true
	m.teams = map[string]int{}
	for _, team := range []fact.Team{s.T, s.CT} {
		name, err := m.teamName(s.Tick, team.Clan, team.Side)
		if err != nil {
			return err
		}
		m.teams[name] = 0
	}
	m.result = ""
	return nil
}

// initRound opens a new round. As the oracle, it clears the late-connect filter before reading the
// rosters, so they can differ from the ones the money check used.
func (m *Match) initRound(s fact.Snapshot) error {
	m.connectedAfterStart = map[uint64]bool{}
	t, ct := m.members(s, fact.SideT), m.members(s, fact.SideCT)
	m.integrityStart = s.RoundsPlayed + 1
	r := &Round{Num: m.integrityStart, StartTick: s.Tick, Members: map[uint64]Member{}, InitT: len(t), InitCT: len(ct),
		sideClans: map[fact.Side]string{}}
	for _, team := range []struct {
		state   fact.Team
		players []fact.Player
	}{{s.T, t}, {s.CT, ct}} {
		// The oracle names teams before the new round replaces the old one.
		clan, err := m.teamName(s.Tick, team.state.Clan, team.state.Side)
		if err != nil {
			return err
		}
		r.sideClans[team.state.Side] = clan
		for _, p := range team.players {
			r.Members[p.Steam64] = Member{Steam64: p.Steam64, Name: p.Name, IsBot: p.IsBot, Side: team.state.Side,
				TeamID: team.state.ID, Clan: clan}
		}
	}
	m.potential = r
	m.postWinCon, m.didRoundEndFire, m.didScoreUpdateFire, m.haveInit = false, false, false, false
	m.roundStartedAt = 0
	return nil
}

// members is the oracle's getTeamMembers over a snapshot: listed records on the side, minus players
// who connected after the round started, then each reconnected player's last record replacing
// their listed one. Reconnected players go in Steam64 order (the oracle's map order is random).
func (m *Match) members(s fact.Snapshot, side fact.Side) []fact.Player {
	var out []fact.Player
	for _, p := range s.Players {
		if p.Listed && p.Side == side && !m.connectedAfterStart[p.Steam64] {
			out = append(out, p)
		}
	}
	for _, id := range slices.Sorted(maps.Keys(m.reconnectedOracle)) {
		if !m.reconnectedOracle[id] {
			continue
		}
		for _, p := range s.Players {
			if m.connectedAfterStart[p.Steam64] || p.Steam64 != id || p.Side != side {
				continue
			}
			if i := slices.IndexFunc(out, func(q fact.Player) bool { return q.Steam64 == id }); i >= 0 {
				out = slices.Delete(out, i, i+1)
			}
			out = append(out, p)
		}
	}
	return out
}

func (m *Match) roundEnd(f fact.RoundEnd) error {
	if !m.isGameLive {
		return nil
	}
	m.didRoundEndFire = true
	if f.Winner < fact.SideT || m.integrityStart != f.RoundsPlayed {
		return nil
	}
	m.potential.Winner = f.Winner
	m.potential.Reason = f.Reason
	winner, err := m.teamName(f.Tick, f.WinnerTeam.Clan, f.WinnerTeam.Side)
	if err != nil {
		return err
	}
	loser, err := m.teamName(f.Tick, f.LoserTeam.Clan, f.LoserTeam.Side)
	if err != nil {
		return err
	}
	m.integrityEnd = f.RoundsPlayed
	m.totalRounds = f.RoundsPlayed
	m.postWinCon = true
	m.potential.WinnerClan = winner
	if !m.potential.Knife && !m.didScoreUpdateFire {
		if err := m.setScore(f.Tick, winner, m.teams[winner]+1); err != nil {
			return err
		}
	}
	w, okW := m.teams[winner]
	l, okL := m.teams[loser]
	if !okW || !okL {
		return m.fail(f.Tick, "round %d: team %q or %q not in the match", f.RoundsPlayed, winner, loser)
	}
	if matchIsDecided(mr+1, w, l) {
		m.winnerClan = winner
		return m.final(f.Tick, f.RoundsPlayed, true)
	}
	return nil
}

func (m *Match) setScore(tick int, team string, score int) error {
	if _, ok := m.teams[team]; !ok {
		return m.fail(tick, "score for team %q, which isn't in the match", team)
	}
	m.teams[team] = score
	return nil
}

// final is the oracle's processRoundFinal: commit the pending round if it ran uninterrupted, and
// end the match on the last round.
func (m *Match) final(tick, roundsPlayed int, last bool) error {
	m.inRound = false
	m.potential.EndTick = tick
	m.integrityOfficial = roundsPlayed
	if m.integrityStart == m.integrityEnd && m.integrityEnd == m.integrityOfficial && !m.potential.committed {
		r := m.potential
		if r.WinnerClan != r.sideClans[fact.SideT] && r.WinnerClan != r.sideClans[fact.SideCT] {
			// The oracle panics here (TeamStats[WinnerClanName] is nil).
			return m.fail(tick, "round %d winner %q isn't one of its teams %v", r.Num, r.WinnerClan, r.sideClans)
		}
		if n := len(m.rounds); n > 0 && r.Num <= m.rounds[n-1].Num {
			return m.fail(tick, "round %d committed after round %d (no in-demo redo path, D12)", r.Num, m.rounds[n-1].Num)
		}
		r.committed = true
		m.rounds = append(m.rounds, r)
	}
	if last {
		m.totalRounds = roundsPlayed
		m.isGameLive = false
		m.result = "Ended"
	}
	return nil
}

// teamName is the oracle's validateTeamName: a clan name, cleaned of a "[tag] " prefix, or a
// side-based name when the demo has no clan names.
func (m *Match) teamName(tick int, clan string, side fact.Side) (string, error) {
	if _, noClans := m.teams["StartedT"]; clan != "" && !noClans {
		if !strings.HasPrefix(clan, "[") {
			return clan, nil
		}
		parts := strings.Split(clan, "] ")
		if len(parts) < 2 {
			return "", m.fail(tick, "clan %q has a tag but no name", clan)
		}
		name := parts[1]     // the oracle's split: "[t] A] B" is "A"
		if len(clan) == 31 { // truncated by the game: match a known team that contains it
			for _, team := range slices.Sorted(maps.Keys(m.teams)) {
				if strings.Contains(team, name) {
					return team, nil
				}
			}
		}
		return name, nil
	}
	round := m.potential.Num
	swap := false
	if round >= mr+1 && round <= mr*2+3 {
		swap = true
	} else if round >= mr*2+4 && (round-(mr*2+4))/6%2 != 0 {
		swap = true
	}
	switch {
	case side == fact.SideT && !swap, side == fact.SideCT && swap:
		return "StartedT", nil
	case side == fact.SideCT && !swap, side == fact.SideT && swap:
		return "StartedCT", nil
	default:
		return "SPECs", nil
	}
}

// matchIsDecided is the oracle's clinch rule for MR12 (13 to win, OT blocks of 6); a level score is
// left to the tie fallback. See fragg-3.0#23.
func matchIsDecided(roundsToWin, winner, loser int) bool {
	if roundsToWin != mr+1 {
		return false
	}
	if winner == mr+1 && loser < mr {
		return true
	}
	if winner > mr {
		overtime := (winner+loser-mr*2-1)/6 + 1
		return (winner-mr-1)/3 == overtime
	}
	return false
}

var headerMatchID = regexp.MustCompile(`^CSC Match ?- (\d+) \|`)

// Finish ends the lifecycle with how the demo stream ended and returns the outcome. Precedence:
// an invariant failure, then an invalid file, then a decode or other failure before the match
// ended, then the outcome.
func (m *Match) Finish(t fact.Terminal) (*Result, error) {
	if m.err != nil {
		return nil, m.err
	}
	if t.Kind == fact.InvalidFileType {
		return nil, t.Err
	}
	// The oracle's tie fallback (fragg#23/#25), which now also needs the game's own match end.
	if m.isGameLive && !m.potential.committed && m.integrityStart == m.integrityEnd &&
		t.Kind != fact.Decode && t.End.CT.Score == t.End.T.Score && m.gameEnded {
		if err := m.final(t.End.Tick, t.End.RoundsPlayed, true); err != nil {
			return nil, err
		}
	}
	ended := m.result == "Ended"
	if !ended && (t.Kind == fact.Decode || t.Kind == fact.Failed || t.Kind == fact.Stopped) {
		return nil, t.Err
	}

	res := &Result{Map: m.mapName, ServerName: m.serverName, WinnerClan: m.winnerClan, TotalRounds: m.totalRounds,
		GameEnded: m.gameEnded, Scores: maps.Clone(m.teams), Reconnected: m.reconnected}
	if g := headerMatchID.FindStringSubmatch(m.serverName); g != nil {
		res.HeaderMatchID, _ = strconv.Atoi(g[1])
	}
	for _, r := range m.rounds {
		if !r.Knife {
			res.Rounds = append(res.Rounds, r)
		}
	}
	switch {
	case len(res.Rounds) == 0:
		res.Outcome = NoRounds
	case !ended && m.gameEnded:
		return nil, &InvariantError{Tick: t.End.Tick, Reason: "the game ended, but not by our match-end rule"}
	case !ended:
		res.Outcome, res.Detail = Unfinished, fmt.Sprintf("demo stopped (%s) after round %d", kind(t.Kind), res.Rounds[len(res.Rounds)-1].Num)
	default:
		res.Outcome, res.Detail = Ended, ""
		if d := coverage(res.Rounds, res.Scores); d != "" {
			res.Outcome, res.Detail = Partial, d
		}
	}
	return res, nil
}

// coverage reports why the committed rounds don't cover the whole match, or "".
func coverage(rounds []*Round, scores map[string]int) string {
	if rounds[0].Num != 1 {
		return fmt.Sprintf("first committed round is %d", rounds[0].Num)
	}
	for i := 1; i < len(rounds); i++ {
		if rounds[i].Num != rounds[i-1].Num+1 {
			return fmt.Sprintf("round %d is missing", rounds[i-1].Num+1)
		}
	}
	sum := 0
	for _, s := range scores {
		sum += s
	}
	if sum != len(rounds) {
		return fmt.Sprintf("scores sum to %d, but %d rounds committed", sum, len(rounds))
	}
	return ""
}

func kind(k fact.TerminalKind) string {
	return [...]string{"clean", "truncated", "decode", "invalid file", "stopped", "failed"}[k]
}
