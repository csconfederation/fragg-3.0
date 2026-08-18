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
// output to the exact literal thresholds matchIsDecided switches on
// (16 / 9 / 13). If those literals ever change without updating roundsToWin
// (or vice versa), MR15/MR8 matches silently stop being recognized as
// decided again.
func TestRoundsToWinMatchesExistingWinConditionBranches(t *testing.T) {
	const (
		mr15Branch = 16
		mr8Branch  = 9
		mr12Branch = 13
	)
	if got := roundsToWin(30); got != mr15Branch {
		t.Fatalf("MR15 threshold = %d, want %d (matchIsDecided's MR15 case)", got, mr15Branch)
	}
	if got := roundsToWin(16); got != mr8Branch {
		t.Fatalf("MR8 threshold = %d, want %d (matchIsDecided's MR8 case)", got, mr8Branch)
	}
	if got := roundsToWin(24); got != mr12Branch {
		t.Fatalf("MR12 threshold = %d, want %d (matchIsDecided's MR12 case)", got, mr12Branch)
	}
}

// TestMatchIsDecidedMR15Tie is the regression test for the actual bug: a
// 15-15 tie (the outcome all 6 known-affected prod matches hit) must be
// recognized as deciding the match. Before this fix, the MR15 branch had a
// normal-win case and an OT-win case but no tie case, so this returned
// false, processRoundFinal() never ran for the true final round, and it was
// never appended to game.Rounds -- silently dropping every stat in it.
func TestMatchIsDecidedMR15Tie(t *testing.T) {
	if !matchIsDecided(16, 15, 15) {
		t.Fatal("15-15 under MR15 (roundsToWin=16) must be decided (a tie) -- this is the fragg-3.0#23 bug")
	}
}

// TestMatchIsDecidedMR15 covers the full MR15 decision surface: not just the
// tie case above, but that normal wins, OT wins, and genuinely undecided
// mid-match scores (which an MR15 match passes through on the way to 30
// rounds, e.g. 13-9) are still classified correctly.
func TestMatchIsDecidedMR15(t *testing.T) {
	tests := []struct {
		name                    string
		winnerScore, loserScore int
		want                    bool
	}{
		{"regulation win 16-14", 16, 14, true},
		{"regulation win 16-0", 16, 0, true},
		{"first OT set win 19-16", 19, 16, true},
		{"second OT set win 22-19", 22, 19, true},
		{"tie 15-15", 15, 15, true},
		{"mid-match score, not decided", 13, 9, false},
		{"one round from clinching, not decided", 15, 12, false},
		{"one round from tie, not decided", 15, 14, false},
		{"mid-OT, not decided", 17, 16, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchIsDecided(16, tt.winnerScore, tt.loserScore); got != tt.want {
				t.Fatalf("matchIsDecided(16, %d, %d) = %v, want %v", tt.winnerScore, tt.loserScore, got, tt.want)
			}
		})
	}
}

// TestMatchIsDecidedMR12NoTieCase locks in existing (pre-fix) behavior: MR12
// has never had a tie case, and this fix must not add one -- league matches
// always play overtime to a real decision (per team confirmation), so a
// 12-12 MR12 scoreline is mid-match, not a valid terminal state.
func TestMatchIsDecidedMR12NoTieCase(t *testing.T) {
	if matchIsDecided(13, 12, 12) {
		t.Fatal("12-12 under MR12 (roundsToWin=13) must not be decided -- MR12 has no tie case")
	}
	if !matchIsDecided(13, 13, 11) {
		t.Fatal("13-11 under MR12 must be a normal win")
	}
	if !matchIsDecided(13, 16, 13) {
		t.Fatal("16-13 under MR12 must be an OT win")
	}
}

// TestMatchIsDecidedMR8Tie protects the MR8 tie case (pre-existing, not
// touched by this fix) from regressing while matchIsDecided was extracted
// from the inline RoundEnd handler.
func TestMatchIsDecidedMR8Tie(t *testing.T) {
	if !matchIsDecided(9, 8, 8) {
		t.Fatal("8-8 under MR8 (roundsToWin=9) must be decided (a tie)")
	}
	if !matchIsDecided(9, 9, 7) {
		t.Fatal("9-7 under MR8 must be a normal win")
	}
}

// TestMatchIsDecidedUnknownFormat defends against a roundsToWin value this
// parser doesn't recognize ever being silently treated as decided.
func TestMatchIsDecidedUnknownFormat(t *testing.T) {
	if matchIsDecided(0, 15, 15) {
		t.Fatal("an unrecognized roundsToWin must never be treated as decided")
	}
	if matchIsDecided(20, 20, 18) {
		t.Fatal("an unrecognized roundsToWin must never be treated as decided")
	}
}
