package csc

import "testing"

// TestMatchIsDecidedMR12RegulationTieIsNotEagerlyDecided is the regression
// test for the actual bug (fragg-3.0#23): a bare MR12 regulation tie (12-12)
// must NOT be treated as decided by matchIsDecided, because whether it's the
// match's real final state depends on whether overtime is then played --
// League/Playoff matches replay OT blocks until decided, Combines play at
// most one OT block and settle for a tie if it's level too, and neither
// policy is recoverable from the demo itself. Before this fix, the RoundEnd
// handler had no tie case for MR12 at all, so a genuine final 12-12 (no OT)
// silently lost its last round -- confirmed against real prod demos, all of
// which actually reach 15-15 (12-12 regulation + one level OT block) with no
// CCSUsrMsg_MatchEndConditions message ever present to distinguish them from
// a League match. The fix is the post-parse fallback in ProcessParser, not a
// tie literal here.
func TestMatchIsDecidedMR12RegulationTieIsNotEagerlyDecided(t *testing.T) {
	if matchIsDecided(13, 12, 12) {
		t.Fatal("12-12 under MR12 (roundsToWin=13) must not be eagerly decided -- whether it's final depends on overtime policy, which matchIsDecided can't know")
	}
}

// TestMatchIsDecidedMR12OTBlockLevelIsNotEagerlyDecided covers the other
// real-world shape: a single MR12 overtime block (12-12 -> 15-15) that
// itself ends level. This is exactly what all 6 known-affected prod matches
// hit. Not decided here either -- caught by the post-parse fallback.
func TestMatchIsDecidedMR12OTBlockLevelIsNotEagerlyDecided(t *testing.T) {
	if matchIsDecided(13, 15, 15) {
		t.Fatal("15-15 under MR12 (roundsToWin=13, one OT block played) must not be eagerly decided -- League/Playoff matches can still continue to a second OT block")
	}
}

// TestMatchIsDecidedMR12Wins covers the unambiguous MR12 win cases: a
// regulation clinch and an overtime-block clinch. These are always safe to
// decide immediately, regardless of match type or overtime policy.
func TestMatchIsDecidedMR12Wins(t *testing.T) {
	tests := []struct {
		name                    string
		winnerScore, loserScore int
		want                    bool
	}{
		{"regulation win 13-11", 13, 11, true},
		{"regulation win 13-0", 13, 0, true},
		{"first OT block win 16-13", 16, 13, true},
		{"second OT block win 19-16", 19, 16, true},
		{"mid-match score, not decided", 10, 7, false},
		{"one round from clinching, not decided", 12, 9, false},
		{"one round from regulation tie, not decided", 12, 11, false},
		{"mid-OT-block, not decided", 14, 13, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchIsDecided(13, tt.winnerScore, tt.loserScore); got != tt.want {
				t.Fatalf("matchIsDecided(13, %d, %d) = %v, want %v", tt.winnerScore, tt.loserScore, got, tt.want)
			}
		})
	}
}

// TestMatchIsDecidedMR8Tie protects MR8's existing tie case (pre-existing,
// not touched by this fix) from regressing while matchIsDecided was
// extracted from the inline RoundEnd handler. Unlike MR12/MR15, MR8
// (knife/short configs) has no overtime, so a bare tie there really is
// always final and is safe to decide eagerly.
func TestMatchIsDecidedMR8Tie(t *testing.T) {
	if !matchIsDecided(9, 8, 8) {
		t.Fatal("8-8 under MR8 (roundsToWin=9) must be decided (a tie) -- MR8 has no overtime")
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
