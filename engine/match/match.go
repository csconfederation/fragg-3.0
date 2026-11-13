// Package match runs the round lifecycle over facts: when the match goes live, which rounds commit,
// how the match ends, and the outcome (Core-planning#301, engine core design). The lifecycle ports
// the oracle's (fragg csc/parser.go at tag oracle-v6port-rc.1) so rounds, sides, scores, and
// player sets match it; the outcome rules (D8, D9) are new.
package match

import (
	"errors"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/csconfederation/fragg-3.0/engine/fact"
)

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
	Num        int
	StartTick  int
	EndTick    int
	WinnerClan string
	Knife      bool
	Pistol     bool
	// FreezeEnd is the round opening at freeze end, after the buy; a round can commit from an
	// earlier open without one.
	FreezeEnd bool
	InitT     int
	InitCT    int
	// TClan and CTClan are the round's team names by side, as the oracle names them at the open.
	TClan, CTClan string
	Members       map[uint64]Member
	committed     bool
	sideClans     map[fact.Side]string
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
	// HeaderMatchID is the match ID in the server name, 0 if none (D11).
	HeaderMatchID int
}

// Stats computes stats over the lifecycle (engine/round's Deriver). Match calls it where the oracle's
// CSC pipeline runs, after updating its own state. Every hook is called in fact order, and an error
// is kept as the match's first error.
type Stats interface {
	// Open is a round open while the match is live; roster is the open's members.
	Open(r *Round, s fact.Snapshot, roster Roster) error
	Event(e Event) error
	// Commit is before the round is added to the committed rounds.
	Commit(r *Round, tick int) error
}

// Event is one fact with what the lifecycle resolved for it.
type Event struct {
	// Fact is as ingested. A Frame has Sample nil: the lurk-cadence frames carry theirs in Sample.
	Fact   fact.Fact
	Sample *fact.Snapshot
	// Roster is the members after the lifecycle applied the fact: Kill, Bomb*, Disconnect, RoundEnd
	// and a sampled Frame.
	Roster Roster
	Live   bool // the match is live (the oracle's IsGameLive)
	// Dropped is a fact the deciding round's wait filters out (D15); it reaches no stat.
	Dropped bool
	// Join is a TeamChange that added this member to the open round.
	Join *Member
	// Win is a RoundEnd that ran the win path; Winner is the round's winner clan then.
	Win    bool
	Winner string
	// Final is the lifecycle's round end processing (final), whether or not it commits.
	Final bool
}

// Roster is Members(s, side) as indices into the snapshot's Players, in members order.
type Roster struct{ T, CT []int }

// Members is the players of side in s, as Match.Members returned them when the roster was taken.
func (r Roster) Members(s fact.Snapshot, side fact.Side) []fact.Player {
	idx := r.T
	if side == fact.SideCT {
		idx = r.CT
	}
	var out []fact.Player
	for _, i := range idx {
		out = append(out, s.Players[i])
	}
	return out
}

// Match is a fact.Sink. Feed it every fact, then call Finish once.
type Match struct {
	stats                                           Stats
	hasGameStarted, isGameLive, inRound             bool
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
	reconnected                                     map[uint64]bool // the oracle's ReconnectedPlayers, which drives membership
	gameEnded                                       bool
	pendingFinal                                    *fact.RoundEnd // the deciding round, until it commits (D15)
	lurkClock                                       int            // the tick of the last lurk-cadence frame
	mapName, serverName                             string
	err                                             *InvariantError
}

// New returns a lifecycle before any fact, with an optional stats collector.
func New(stats ...Stats) *Match {
	m := &Match{
		integrityStart: -1, integrityEnd: -1, integrityOfficial: -1,
		potential:           &Round{Members: map[uint64]Member{}},
		connectedAfterStart: map[uint64]bool{},
		reconnected:         map[uint64]bool{},
	}
	if len(stats) > 0 {
		m.stats = stats[0]
	}
	return m
}

// lurkTicks is the lurk sampler's cadence: 4 s at the oracle's tick rate.
const lurkTicks = 4 * fact.TickRate

// collect passes a stat fact to the stats, after the lifecycle's own update. Only a lurk-cadence
// frame is passed, with its sample.
func (m *Match) collect(f fact.Fact) error {
	if m.stats == nil {
		return nil
	}
	e := Event{Fact: f, Live: m.isGameLive, Dropped: m.pendingFinal != nil && !bombFact(f)}
	if !e.Dropped {
		switch f := f.(type) {
		case fact.Frame:
			if !e.Live || !m.inRound || m.lurkClock+lurkTicks >= f.Tick {
				return nil
			}
			m.lurkClock = f.Tick
			s := f.Sample()
			f.Sample = nil
			e.Fact, e.Sample, e.Roster = f, &s, m.roster(s)
		case fact.Kill:
			if e.Live {
				e.Roster = m.roster(f.Snapshot)
			}
		case fact.BombPlanted:
			e.Roster = m.roster(f.Snapshot)
		case fact.BombDefused:
			e.Roster = m.roster(f.Snapshot)
		case fact.BombExplode:
			e.Roster = m.roster(f.Snapshot)
		case fact.Disconnect:
			if e.Live {
				e.Roster = m.roster(f.Snapshot)
			}
		}
	} else if _, ok := f.(fact.Frame); ok {
		return nil
	}
	return m.event(e)
}

func (m *Match) event(e Event) error {
	if err := m.stats.Event(e); err != nil {
		return m.keep(err)
	}
	return nil
}

// keep records a collector's error as the match's first error.
func (m *Match) keep(err error) error {
	if m.err == nil {
		var inv *InvariantError
		if !errors.As(err, &inv) {
			inv = &InvariantError{Reason: err.Error()}
		}
		m.err = inv
	}
	return m.err
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
		return m.collect(f)
	case fact.RoundStart:
		m.roundStartedAt = f.Tick
	case fact.Frame:
		if m.roundStartedAt > 0 && m.roundStartedAt+fact.RoundStartWindow > f.Tick && !m.haveInit {
			if err := m.open(f.Sample(), false); err != nil {
				return err
			}
		}
		return m.collect(f)
	case fact.Kill, fact.Hurt, fact.Flashed, fact.BombPlanted, fact.BombDefused, fact.BombExplode, fact.GrenadeThrow:
		return m.collect(f)
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
		if m.pendingFinal != nil {
			return m.flush()
		}
		if !m.didRoundEndFire {
			m.integrityEnd--
		}
		if m.isGameLive && m.integrityEnd == f.RoundsPlayed {
			return m.final(f.Tick, f.RoundsPlayed, false)
		}
	case fact.TeamChange:
		var join *Member
		if m.isGameLive && m.inRound && m.pendingFinal == nil && f.NewSide > fact.SideSpectator {
			if _, ok := m.potential.Members[f.Player.Steam64]; !ok && f.Player.Alive {
				clan, err := m.teamName(f.Tick, f.NewTeam.Clan, f.NewTeam.Side)
				if err != nil {
					return err
				}
				mem := Member{Steam64: f.Player.Steam64, Name: f.Player.Name,
					IsBot: f.Player.IsBot, Side: f.NewTeam.Side, TeamID: f.NewTeam.ID, Clan: clan}
				m.potential.Members[f.Player.Steam64] = mem
				join = &mem
			}
		}
		if m.stats != nil {
			return m.event(Event{Fact: f, Live: m.isGameLive, Join: join})
		}
	case fact.PlayerInfo:
		if f.Found {
			m.reconnected[f.Resolved] = true
			if m.inRound && m.isGameLive {
				m.connectedAfterStart[f.Resolved] = true
			}
			if m.stats != nil {
				return m.event(Event{Fact: f, Live: m.isGameLive})
			}
		}
	case fact.Disconnect:
		if m.reconnected[f.Steam64] {
			m.reconnected[f.Steam64] = false
		}
		return m.collect(f)
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
		return m.flush()
	}
	return nil
}

// open is the oracle's round open: at a frame within the early window, or at freeze end (which
// also detects knife rounds and always reopens).
func (m *Match) open(s fact.Snapshot, freezeEnd bool) error {
	if err := m.flush(); err != nil {
		return err
	}
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
	m.potential.Pistol, m.potential.FreezeEnd = pistol, freezeEnd
	if m.stats != nil {
		m.lurkClock = 0
		if err := m.stats.Open(m.potential, s, m.roster(s)); err != nil {
			return m.keep(err)
		}
	}
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
		if team.state.Side == fact.SideT {
			r.TClan = clan
		} else {
			r.CTClan = clan
		}
		for _, p := range team.players {
			r.Members[p.Steam64] = Member{Steam64: p.Steam64, Name: p.Name, IsBot: p.IsBot, Side: team.state.Side,
				TeamID: team.state.ID, Clan: clan}
		}
	}
	m.potential = r
	m.didRoundEndFire, m.didScoreUpdateFire, m.haveInit = false, false, false
	m.roundStartedAt = 0
	return nil
}

// members is the oracle's getTeamMembers over a snapshot: listed records on the side, minus players
// who connected after the round started, then each reconnected player's last record replacing
// their listed one. Reconnected players go in Steam64 order (the oracle's map order is random).
func (m *Match) members(s fact.Snapshot, side fact.Side) []fact.Player {
	var out []fact.Player
	for _, i := range m.memberIdx(s, side) {
		out = append(out, s.Players[i])
	}
	return out
}

// memberIdx is members as indices into s.Players.
func (m *Match) memberIdx(s fact.Snapshot, side fact.Side) []int {
	var out []int
	for i, p := range s.Players {
		if p.Listed && p.Side == side && !m.connectedAfterStart[p.Steam64] {
			out = append(out, i)
		}
	}
	for _, id := range slices.Sorted(maps.Keys(m.reconnected)) {
		if !m.reconnected[id] {
			continue
		}
		for i, p := range s.Players {
			if m.connectedAfterStart[p.Steam64] || p.Steam64 != id || p.Side != side {
				continue
			}
			if j := slices.IndexFunc(out, func(k int) bool { return s.Players[k].Steam64 == id }); j >= 0 {
				out = slices.Delete(out, j, j+1)
			}
			out = append(out, i)
		}
	}
	return out
}

func (m *Match) roster(s fact.Snapshot) Roster {
	return Roster{T: m.memberIdx(s, fact.SideT), CT: m.memberIdx(s, fact.SideCT)}
}

func (m *Match) roundEnd(f fact.RoundEnd) error {
	if err := m.flush(); err != nil {
		return err
	}
	if !m.isGameLive {
		return nil
	}
	m.didRoundEndFire = true
	if f.Winner < fact.SideT || m.integrityStart != f.RoundsPlayed {
		return m.roundEndEvent(f, false)
	}
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
	m.potential.WinnerClan = winner
	if !m.potential.Knife && !m.didScoreUpdateFire {
		if err := m.setScore(f.Tick, winner, m.teams[winner]+1); err != nil {
			return err
		}
	}
	if err := m.roundEndEvent(f, true); err != nil {
		return err
	}
	w, okW := m.teams[winner]
	l, okL := m.teams[loser]
	if !okW || !okL {
		return m.fail(f.Tick, "round %d: team %q or %q not in the match", f.RoundsPlayed, winner, loser)
	}
	if matchIsDecided(fact.MR+1, w, l) {
		m.winnerClan = winner
		if f.Reason != fact.ReasonTargetBombed {
			return m.final(f.Tick, f.RoundsPlayed, true)
		}
		// The explosion's damage and kills arrive after RoundEnd: the deciding round commits at the
		// win panel or the demo's end, and only those facts reach it meanwhile (D15).
		m.pendingFinal = &f
	}
	return nil
}

// roundEndEvent passes a live RoundEnd to the stats; win is whether the win path ran for it.
func (m *Match) roundEndEvent(f fact.RoundEnd, win bool) error {
	if m.stats == nil {
		return nil
	}
	e := Event{Fact: f, Live: true, Roster: m.roster(f.Snapshot), Win: win}
	if win {
		e.Winner = m.potential.WinnerClan
	}
	return m.event(e)
}

func (m *Match) setScore(tick int, team string, score int) error {
	if _, ok := m.teams[team]; !ok {
		return m.fail(tick, "score for team %q, which isn't in the match", team)
	}
	m.teams[team] = score
	return nil
}

// bombFact reports whether f is the bomb's damage or kill.
func bombFact(f fact.Fact) bool {
	var w *int
	switch f := f.(type) {
	case fact.Hurt:
		w = f.Weapon
	case fact.Kill:
		w = f.Weapon
	}
	return w != nil && *w == fact.EqBomb
}

// flush commits the pending deciding round and ends the match.
func (m *Match) flush() error {
	f := m.pendingFinal
	if f == nil {
		return nil
	}
	m.pendingFinal = nil
	return m.final(f.Tick, f.RoundsPlayed, true)
}

// final is the oracle's processRoundFinal: commit the pending round if it ran uninterrupted, and
// end the match on the last round.
func (m *Match) final(tick, roundsPlayed int, last bool) error {
	m.inRound = false
	m.potential.EndTick = tick
	m.integrityOfficial = roundsPlayed
	if m.stats != nil {
		if err := m.event(Event{Final: true, Live: m.isGameLive}); err != nil {
			return err
		}
	}
	if m.integrityStart == m.integrityEnd && m.integrityEnd == m.integrityOfficial && !m.potential.committed {
		r := m.potential
		if r.WinnerClan != r.sideClans[fact.SideT] && r.WinnerClan != r.sideClans[fact.SideCT] {
			// The oracle panics here (TeamStats[WinnerClanName] is nil).
			return m.fail(tick, "round %d winner %q isn't one of its teams %v", r.Num, r.WinnerClan, r.sideClans)
		}
		// An in-demo restore (D12) replays from round r.Num: the rounds committed from it on are
		// superseded. The oracle's removeInvalidRounds agrees when the replay commits every number;
		// if it skips one, the gap is left for Finish to report as Partial (D9).
		m.rounds = slices.DeleteFunc(m.rounds, func(c *Round) bool { return c.Num >= r.Num })
		if m.stats != nil {
			if err := m.stats.Commit(r, tick); err != nil {
				return m.keep(err)
			}
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
		if len(parts) < 2 || parts[1] == "" {
			return clan, nil // "[Name]" or "[tag] ": no tag to strip (the oracle indexes past the split, or names the team "")
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
	if round >= fact.MR+1 && round <= fact.MR*2+3 {
		swap = true
	} else if round >= fact.MR*2+4 && (round-(fact.MR*2+4))/6%2 != 0 {
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
	if roundsToWin != fact.MR+1 {
		return false
	}
	if winner == fact.MR+1 && loser < fact.MR {
		return true
	}
	if winner > fact.MR {
		overtime := (winner+loser-fact.MR*2-1)/6 + 1
		return (winner-fact.MR-1)/3 == overtime
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
	if err := m.flush(); err != nil {
		return nil, err
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

	res := &Result{Map: m.mapName, WinnerClan: m.winnerClan, TotalRounds: m.totalRounds, Scores: maps.Clone(m.teams)}
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
