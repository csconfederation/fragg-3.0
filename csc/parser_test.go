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

// TestMatchIsDecidedMR8NoTieCase mirrors the MR12 fix: csc-configs' 1v1
// config (gamemode_competitive_server.cfg) sets mp_overtime_enable=1 with
// mp_overtime_limit=0 -- unlimited overtime halves -- so an 8-8 tie is no
// more automatically final than MR12's 12-12 is. An earlier version of this
// test asserted the opposite ("MR8 has no overtime"); that was wrong and,
// per review, is worth pinning down explicitly rather than just deleting.
func TestMatchIsDecidedMR8NoTieCase(t *testing.T) {
	if matchIsDecided(9, 8, 8) {
		t.Fatal("8-8 under MR8 (roundsToWin=9) must not be eagerly decided -- the 1v1 config allows unlimited overtime, same as MR12")
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

// TestShouldFinalizeViaFallback covers the four guards from the fragg-3.0#25
// review: a genuine, complete, level, previously-uncommitted final round is
// the only case that finalizes. Each other test flips exactly one guard to
// confirm it alone is enough to block finalizing -- these map directly to
// the review's four scenarios (truncated match, double-processing, mid-round
// EOF) plus the tie-detection case itself.
func TestShouldFinalizeViaFallback(t *testing.T) {
	tests := []struct {
		name             string
		isGameLive       bool
		alreadyCommitted bool
		roundEndFired    bool
		ctScore, tScore  int
		want             bool
	}{
		{
			name: "genuine level final round -- the fix's actual target", isGameLive: true,
			alreadyCommitted: false, roundEndFired: true, ctScore: 15, tScore: 15, want: true,
		},
		{
			name: "already decided elsewhere -- match already ended, never re-finalize", isGameLive: false,
			alreadyCommitted: false, roundEndFired: true, ctScore: 15, tScore: 15, want: false,
		},
		{
			name: "already committed via RoundEndOfficial -- would double-process the same round", isGameLive: true,
			alreadyCommitted: true, roundEndFired: true, ctScore: 15, tScore: 15, want: false,
		},
		{
			name: "round never concluded -- demo cut off mid-round, not mid-match-boundary", isGameLive: true,
			alreadyCommitted: false, roundEndFired: false, ctScore: 15, tScore: 15, want: false,
		},
		{
			name: "asymmetric score -- an unclinched, non-level score can only mean a truncated match", isGameLive: true,
			alreadyCommitted: false, roundEndFired: true, ctScore: 13, tScore: 12, want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldFinalizeViaFallback(tt.isGameLive, tt.alreadyCommitted, tt.roundEndFired, tt.ctScore, tt.tScore); got != tt.want {
				t.Fatalf("shouldFinalizeViaFallback(%v, %v, %v, %d, %d) = %v, want %v",
					tt.isGameLive, tt.alreadyCommitted, tt.roundEndFired, tt.ctScore, tt.tScore, got, tt.want)
			}
		})
	}
}
