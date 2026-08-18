package csc

// ParseHooks lets another pipeline (eco/swing) observe CSC round lifecycle
// without CSC importing that pipeline. All callbacks are optional.
type ParseHooks struct {
	// OnInitRound fires from initRound after PotentialRound is replaced.
	OnInitRound func()
	// OnRoundWinCon fires from processRoundOnWinCon after the winner is set.
	// winnerENUM is 2 (T) or 3 (CT), matching demoinfocs common.Team.
	OnRoundWinCon func(winnerENUM int)
	// OnRoundCommitted fires immediately before a round with IntegrityCheck is
	// appended to Game.Rounds. The returned value is stored on that round as
	// EcoSnapshot and rides through removeInvalidRounds.
	OnRoundCommitted func() any
}
