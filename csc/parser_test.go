package csc

import "testing"

// roundsToWin must derive the real clinching-score threshold from the
// server's mp_maxrounds cvar instead of assuming every match is MR12. A
// wrong threshold here means the RoundEnd win-condition branches in
// ProcessParser never recognize the match as decided, so processRoundFinal
// never runs for the true final round and it's silently never appended to
// game.Rounds — see fragg-3.0#23: every MR15 Combine match that ended 15-15
// permanently lost its 30th round's entire box score this way.
func TestRoundsToWin(t *testing.T) {
	tests := []struct {
		name      string
		maxRounds int32
		want      int
	}{
		{"MR15 (Combines): mp_maxrounds=30 -> win at 16", 30, 16},
		{"MR12 (league): mp_maxrounds=24 -> win at 13", 24, 13},
		{"MR8 (knife/short config): mp_maxrounds=16 -> win at 9", 16, 9},
		{"message absent (0): falls back to MR12 -> 13", 0, 13},
		{"negative (defensive): falls back to MR12 -> 13", -1, 13},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := roundsToWin(tt.maxRounds); got != tt.want {
				t.Fatalf("roundsToWin(%d) = %d, want %d", tt.maxRounds, got, tt.want)
			}
		})
	}
}

// TestRoundsToWinMatchesExistingWinConditionBranches locks roundsToWin's
// output to the exact literal thresholds the RoundEnd handler branches on
// (game.RoundsToWin == 16 / == 9 / == 13). If those branch literals ever
// change without updating roundsToWin (or vice versa), MR15/MR8 matches
// silently stop being recognized as decided again.
func TestRoundsToWinMatchesExistingWinConditionBranches(t *testing.T) {
	const (
		mr15Branch = 16
		mr8Branch  = 9
		mr12Branch = 13
	)
	if got := roundsToWin(30); got != mr15Branch {
		t.Fatalf("MR15 threshold = %d, want %d (RoundEnd handler's MR15 branch)", got, mr15Branch)
	}
	if got := roundsToWin(16); got != mr8Branch {
		t.Fatalf("MR8 threshold = %d, want %d (RoundEnd handler's MR8 branch)", got, mr8Branch)
	}
	if got := roundsToWin(24); got != mr12Branch {
		t.Fatalf("MR12 threshold = %d, want %d (RoundEnd handler's MR12 branch)", got, mr12Branch)
	}
}
