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
	UserID    int // the game's user ID; tells bots apart, who share Steam64 0
	Alive     bool
	Connected bool
	// Listed means the game's team member list holds this record (the oracle's first source of
	// team members; the second is every participant, for reconnects).
	Listed      bool
	Money       int
	Spent       int  // this round
	TeamStateID *int // the ID of the player's team state, nil without one (the oracle compares killer and victim by it)
	EquipValue  int  // current equipment value
	Pos         [3]float64
	// FlashDuration is the player's current flash blindness (seconds), the entity field.
	FlashDuration float64
	// Weapons holds the equipment types the player carries; set for kill victims only.
	Weapons []int
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
	Warmup       bool
	FreezeTime   bool
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

// Frame is the end of every frame. Sample reads the full snapshot at this frame and is valid only
// during the Fact call: most frames are never sampled (the oracle's early round open reads frames
// within RoundStartWindow ticks of a RoundStart, its lurk sampler one frame every 4 s), and
// reading every frame would cost about 18% of the parse.
type Frame struct {
	Tick, Frame int
	Sample      func() Snapshot
}

// RoundStartWindow is the oracle's early-open window: 1 s at its fixed 64 tick rate.
const RoundStartWindow = 64

// FreezeEnd is the end of freeze time.
type FreezeEnd struct{ Snapshot }

// RoundEnd is the game's round end. Winner is SideSpectator or lower for a draw, when the teams
// are zero values.
type RoundEnd struct {
	Snapshot
	Winner     Side
	Reason     int
	WinnerTeam Team
	LoserTeam  Team
}

// ScoreUpdated is a team's score change; Opponent is read at the same time.
type ScoreUpdated struct {
	Tick     int
	Team     Team
	Opponent Team
}

// RoundEndOfficial is the game's official round end (not sent for a demo's last round).
type RoundEndOfficial struct{ Snapshot }

// Kill is a death. Actors are nil when absent (world damage, a corrupt demo); Weapon is nil when
// the event has none.
type Kill struct {
	Snapshot
	Killer, Victim, Assister *Player
	Weapon                   *int
	Headshot                 bool
	AssistedFlash            bool
}

// Hurt is damage taken. Attacker is nil for world damage.
type Hurt struct {
	Snapshot
	Player, Attacker  *Player
	Weapon            *int
	HealthDamageTaken int // the HP the game removed (D1)
}

// Flashed is a player blinded by a flashbang; Duration is the event's blind time (seconds).
type Flashed struct {
	Tick             int
	Player, Attacker *Player
	Duration         float64
}

// BombPlanted, BombDefused and BombExplode are bomb events; Player is nil when absent.
type BombPlanted struct {
	Snapshot
	Player *Player
}
type BombDefused struct {
	Snapshot
	Player *Player
}
type BombExplode struct {
	Snapshot
	Player *Player
}

// GrenadeThrow is a grenade leaving a player's hand.
type GrenadeThrow struct {
	Tick    int
	Thrower *Player
	Weapon  *int
}

// TeamChange is a player moving to NewTeam, which is read at the event.
type TeamChange struct {
	Tick    int
	Player  Player
	NewSide Side
	NewTeam Team
}

// PlayerInfo is a userinfo update. XUID is the Steam64 the update carries. Found and Resolved are
// the player the oracle looks up for it (the participant in that userinfo slot), which can still
// be the slot's previous occupant; a bot resolves to 0. Identical re-sends aren't reported.
type PlayerInfo struct {
	Tick     int
	XUID     uint64
	Found    bool
	Resolved uint64
}

// Connect is a player's controller connecting (including a reconnect).
type Connect struct {
	Tick    int
	Steam64 uint64
}

// Disconnect is a player leaving; the snapshot is taken at the event.
type Disconnect struct {
	Snapshot
	Steam64 uint64
}

// Footstep marks movement; the oracle uses it to tell that a round is in play. CS2 demos rarely
// carry the game event.
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
func (Connect) fact()          {}
func (Disconnect) fact()       {}
func (Footstep) fact()         {}
func (GamePhase) fact()        {}
func (WinPanelMatch) fact()    {}
func (Kill) fact()             {}
func (Hurt) fact()             {}
func (Flashed) fact()          {}
func (BombPlanted) fact()      {}
func (BombDefused) fact()      {}
func (BombExplode) fact()      {}
func (GrenadeThrow) fact()     {}

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

// Terminal is how the demo stream ended. Err is the underlying error, nil when Clean. End is the
// game state after the last frame.
type Terminal struct {
	Kind  TerminalKind
	Frame int // the frame that failed to decode, for Decode
	Err   error
	End   Snapshot
}
