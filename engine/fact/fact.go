// Package fact holds the facts an ingest emits and match consumes. It is backend-neutral: no
// demoinfocs type appears here, so a CS:GO (v4) ingest can emit the same facts later
// (Core-planning#575). Each fact carries the game state it needs, read when the event fired.
package fact

// Side is a team side, numbered as in the game (and demoinfocs' common.Team).
type Side uint8

const (
	SideUnassigned Side = 0
	SideSpectator  Side = 1
	SideT          Side = 2
	SideCT         Side = 3
)

// Player is one participant as observed at the fact's tick.
type Player struct {
	Steam64   uint64
	Name      string
	Side      Side
	IsBot     bool
	Alive     bool
	Connected bool
	// Listed means the game's team member list holds this record (the oracle's first source of
	// team members; the second is every participant, for reconnects).
	Listed bool
	Money  int
	Spent  int // this round
}

// Team is one team as observed at the fact's tick.
type Team struct {
	Side  Side
	ID    int
	Clan  string
	Score int
}

// Snapshot is the game state at a tick. Players holds every T and CT participant record in the
// game's participant order, duplicates included: a player who reconnected can have an old and a
// new record, and the oracle keeps the last one.
type Snapshot struct {
	Tick         int
	RoundsPlayed int
	T, CT        Team
	Players      []Player
}

// Fact is one observation from the demo, in dispatch order.
type Fact interface{ fact() }

// Header is the demo file header. ServerName is the server's hostname, which carries the match
// ID on CSC servers since S15.
type Header struct{ ServerName string }

// ServerInfo carries the map name.
type ServerInfo struct{ Map string }

// RoundStart is the game's round start.
type RoundStart struct{ Tick, RoundsPlayed int }

// Frame is a frame end within RoundStartWindow ticks of a RoundStart, for the oracle's early
// round open.
type Frame struct{ Snapshot }

// RoundStartWindow is the oracle's early-open window: 1 s at its fixed 64 tick rate.
const RoundStartWindow = 64

// FreezeEnd is the end of freeze time.
type FreezeEnd struct{ Snapshot }

// RoundEnd is the game's round end. Winner is SideSpectator or lower for a draw, when the teams
// are zero values.
type RoundEnd struct {
	Tick, RoundsPlayed int
	Winner             Side
	Reason             int
	WinnerTeam         Team
	LoserTeam          Team
}

// ScoreUpdated is a team's score change; Opponent is read at the same time.
type ScoreUpdated struct {
	Tick     int
	Team     Team
	Opponent Team
}

// RoundEndOfficial is the game's official round end (not sent for a demo's last round).
type RoundEndOfficial struct{ Tick, RoundsPlayed int }

// TeamChange is a player moving to NewTeam, which is read at the event.
type TeamChange struct {
	Tick    int
	Player  Player
	NewSide Side
	NewTeam Team
}

// PlayerInfo is a player's info update (a connect or reconnect).
type PlayerInfo struct {
	Tick    int
	Steam64 uint64
}

// Disconnect is a player leaving; the snapshot is taken at the event.
type Disconnect struct {
	Snapshot
	Steam64 uint64
}

// Footstep marks movement; the oracle uses it to tell that a round is in play. At most one per tick.
type Footstep struct{ Tick int }

// GamePhase is a game phase change (5 is game ended). It can arrive before the RoundEnd of the
// round that ended the game.
type GamePhase struct{ Tick, Phase int }

// GamePhaseEnded is the game phase for a finished match.
const GamePhaseEnded = 5

// WinPanelMatch is the end-of-match win panel.
type WinPanelMatch struct{ Tick int }

func (Header) fact()           {}
func (ServerInfo) fact()       {}
func (RoundStart) fact()       {}
func (Frame) fact()            {}
func (FreezeEnd) fact()        {}
func (RoundEnd) fact()         {}
func (ScoreUpdated) fact()     {}
func (RoundEndOfficial) fact() {}
func (TeamChange) fact()       {}
func (PlayerInfo) fact()       {}
func (Disconnect) fact()       {}
func (Footstep) fact()         {}
func (GamePhase) fact()        {}
func (WinPanelMatch) fact()    {}

// Sink receives facts in order. A non-nil error stops the parse; the ingest returns it.
type Sink interface {
	Fact(Fact) error
}

// TerminalKind is how the demo stream ended.
type TerminalKind uint8

const (
	Clean           TerminalKind = iota // the demo ended normally
	Truncated                           // the stream ended early
	Decode                              // data couldn't be decoded; events before it are valid
	InvalidFileType                     // not a CS2 demo
	Stopped                             // the sink returned an error, or the context was canceled
	Failed                              // any other error
)

// Terminal is how the demo stream ended. Err is the underlying error, nil when Clean.
type Terminal struct {
	Kind  TerminalKind
	Frame int // the frame that failed to decode, for Decode
	Err   error
}
