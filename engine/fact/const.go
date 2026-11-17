package fact

// Game constants shared by the engine's packages. Where the oracle (csc/parser.go) declares one
// in several places, this is the single definition.
const (
	TickRate = 64 // server ticks per second
	MR       = 12 // half the regulation rounds (MR12)

	// FirstPistolRound and SecondPistolRound are the match's pistol rounds (D13).
	FirstPistolRound  = 1
	SecondPistolRound = MR + 1
)

// The demoinfocs equipment types the engine tests.
const (
	EqP2000      = 1
	EqRevolver   = 10
	EqAWP        = 309
	EqBomb       = 404
	EqKnife      = 405
	EqMolotov    = 502
	EqIncendiary = 503
	EqFlash      = 504
	EqSmoke      = 505
	EqHE         = 506
)

// ReasonTargetBombed is RoundEnd.Reason when the bomb exploded.
const ReasonTargetBombed = 1

var roundEndNames = map[int]string{
	0:  "StillInProgress", //base values
	1:  "TargetBombed",
	2:  "VIPEscaped",
	3:  "VIPKilled",
	4:  "TerroristsEscaped",
	5:  "CTStoppedEscape",
	6:  "TerroristsStopped",
	7:  "BombDefused",
	8:  "CTWin",
	9:  "TerroristsWin",
	10: "Draw",
	11: "HostagesRescued",
	12: "TargetSaved",
	13: "HostagesNotRescued",
	14: "TerroristsNotEscaped",
	15: "VIPNotEscaped",
	16: "GameStart",
	17: "TerroristsSurrender",
	18: "CTSurrender",
	19: "TerroristsPlanted",
	20: "CTsReachedHostage",
}

// RoundEndName is the oracle's name for a RoundEnd.Reason, "" for an unknown one.
func RoundEndName(reason int) string { return roundEndNames[reason] }
