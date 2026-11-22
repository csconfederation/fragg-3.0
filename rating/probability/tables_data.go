package probability

// DefaultTables returns probability tables measured on CSC demos: 6,389 S18-S21 league (BO1/BO3,
// playoffs) and Combine maps, 140,364 rounds and 987,108 kills (engine/cmd/probtables;
// Core-planning#305).
func DefaultTables() *ProbabilityTables {
	tables := NewProbabilityTables()

	// === BASE WIN PROBABILITIES ===
	// Format: "TvCT_bombStatus" (e.g., "5v4_none", "3v2_planted"), T-side win probability. A decided
	// round (a side eliminated, except Ts dead with the bomb planted) never reads the table.

	// No bomb planted
	tables.BaseWinProb["5v5_none"] = 0.489 // 68365 / 139906
	tables.BaseWinProb["5v4_none"] = 0.707 // 45155 / 63854
	tables.BaseWinProb["5v3_none"] = 0.887 // 24966 / 28131
	tables.BaseWinProb["5v2_none"] = 0.978 // 11751 / 12019
	tables.BaseWinProb["5v1_none"] = 0.999 // 4756 / 4761

	tables.BaseWinProb["4v5_none"] = 0.298 // 22356 / 75097
	tables.BaseWinProb["4v4_none"] = 0.510 // 35333 / 69274
	tables.BaseWinProb["4v3_none"] = 0.748 // 33072 / 44223
	tables.BaseWinProb["4v2_none"] = 0.927 // 21330 / 23010
	tables.BaseWinProb["4v1_none"] = 0.994 // 9511 / 9567

	tables.BaseWinProb["3v5_none"] = 0.129 // 4951 / 38472
	tables.BaseWinProb["3v4_none"] = 0.281 // 14867 / 52890
	tables.BaseWinProb["3v3_none"] = 0.513 // 22340 / 43557
	tables.BaseWinProb["3v2_none"] = 0.782 // 20331 / 25993
	tables.BaseWinProb["3v1_none"] = 0.964 // 10928 / 11336

	tables.BaseWinProb["2v5_none"] = 0.028 // 564 / 19915
	tables.BaseWinProb["2v4_none"] = 0.090 // 3200 / 35427
	tables.BaseWinProb["2v3_none"] = 0.228 // 7967 / 34874
	tables.BaseWinProb["2v2_none"] = 0.487 // 11105 / 22820
	tables.BaseWinProb["2v1_none"] = 0.816 // 8374 / 10263

	tables.BaseWinProb["1v5_none"] = 0.002 // 23 / 10769
	tables.BaseWinProb["1v4_none"] = 0.009 // 209 / 23019
	tables.BaseWinProb["1v3_none"] = 0.038 // 1001 / 26066
	tables.BaseWinProb["1v2_none"] = 0.135 // 2491 / 18515
	tables.BaseWinProb["1v1_none"] = 0.416 // 3492 / 8396

	// Bomb planted - T-side advantage
	tables.BaseWinProb["5v5_planted"] = 0.770 // 1080 / 1402
	tables.BaseWinProb["5v4_planted"] = 0.859 // 2574 / 2998
	tables.BaseWinProb["5v3_planted"] = 0.947 // 4669 / 4932
	tables.BaseWinProb["5v2_planted"] = 0.994 // 5577 / 5610
	tables.BaseWinProb["5v1_planted"] = 0.999 // 5111 / 5118

	tables.BaseWinProb["4v5_planted"] = 0.511 // 740 / 1447
	tables.BaseWinProb["4v4_planted"] = 0.666 // 3514 / 5275
	tables.BaseWinProb["4v3_planted"] = 0.852 // 9425 / 11058
	tables.BaseWinProb["4v2_planted"] = 0.969 // 15113 / 15594
	tables.BaseWinProb["4v1_planted"] = 0.998 // 15400 / 15430

	tables.BaseWinProb["3v5_planted"] = 0.236 // 285 / 1206
	tables.BaseWinProb["3v4_planted"] = 0.407 // 2252 / 5536
	tables.BaseWinProb["3v3_planted"] = 0.651 // 9121 / 14009
	tables.BaseWinProb["3v2_planted"] = 0.881 // 19695 / 22352
	tables.BaseWinProb["3v1_planted"] = 0.988 // 24619 / 24919

	tables.BaseWinProb["2v5_planted"] = 0.065 // 55 / 845
	tables.BaseWinProb["2v4_planted"] = 0.147 // 657 / 4462
	tables.BaseWinProb["2v3_planted"] = 0.338 // 4279 / 12652
	tables.BaseWinProb["2v2_planted"] = 0.644 // 14138 / 21946
	tables.BaseWinProb["2v1_planted"] = 0.916 // 24097 / 26307

	tables.BaseWinProb["1v5_planted"] = 0.002 // 1 / 598
	tables.BaseWinProb["1v4_planted"] = 0.027 // 83 / 3103
	tables.BaseWinProb["1v3_planted"] = 0.084 // 774 / 9161
	tables.BaseWinProb["1v2_planted"] = 0.272 // 4594 / 16896
	tables.BaseWinProb["1v1_planted"] = 0.631 // 13201 / 20935

	// 0 T alive but bomb planted - depends on defuse time
	tables.BaseWinProb["0v5_planted"] = 0.000 // 0 / 346
	tables.BaseWinProb["0v4_planted"] = 0.004 // 6 / 1645
	tables.BaseWinProb["0v3_planted"] = 0.009 // 43 / 4748
	tables.BaseWinProb["0v2_planted"] = 0.043 // 356 / 8240
	tables.BaseWinProb["0v1_planted"] = 0.243 // 1928 / 7945

	// Empirical cells can violate alive-count / plant monotonicity due to
	// selection bias and small samples. Enforce invariants so a kill or plant
	// never lowers the beneficiary's modeled win probability.
	EnforceMonotonicity(tables)

	// === DUEL WIN RATES ===
	// Format: "attacker_vs_defender" (e.g., "rifle_vs_pistol")
	// Values represent attacker win probability from empirical data

	// Starter Pistol attacking
	tables.DuelWinRates["starter_pistol_vs_starter_pistol"] = 0.500  // mirror
	tables.DuelWinRates["starter_pistol_vs_upgraded_pistol"] = 0.519 // 8985 / 17301
	tables.DuelWinRates["starter_pistol_vs_smg"] = 0.265             // 5580 / 21054
	tables.DuelWinRates["starter_pistol_vs_rifle"] = 0.245           // 9409 / 38349
	tables.DuelWinRates["starter_pistol_vs_awp"] = 0.261             // 14958 / 57203

	// Upgraded Pistol attacking
	tables.DuelWinRates["upgraded_pistol_vs_starter_pistol"] = 0.481  // 8316 / 17301
	tables.DuelWinRates["upgraded_pistol_vs_upgraded_pistol"] = 0.500 // mirror
	tables.DuelWinRates["upgraded_pistol_vs_smg"] = 0.365             // 1958 / 5369
	tables.DuelWinRates["upgraded_pistol_vs_rifle"] = 0.358           // 5720 / 15987
	tables.DuelWinRates["upgraded_pistol_vs_awp"] = 0.341             // 17911 / 52505

	// SMG attacking
	tables.DuelWinRates["smg_vs_starter_pistol"] = 0.735  // 15474 / 21054
	tables.DuelWinRates["smg_vs_upgraded_pistol"] = 0.635 // 3411 / 5369
	tables.DuelWinRates["smg_vs_smg"] = 0.500             // mirror
	tables.DuelWinRates["smg_vs_rifle"] = 0.431           // 18817 / 43660
	tables.DuelWinRates["smg_vs_awp"] = 0.398             // 34812 / 87535

	// Rifle attacking
	tables.DuelWinRates["rifle_vs_starter_pistol"] = 0.755  // 28940 / 38349
	tables.DuelWinRates["rifle_vs_upgraded_pistol"] = 0.642 // 10267 / 15987
	tables.DuelWinRates["rifle_vs_smg"] = 0.569             // 24843 / 43660
	tables.DuelWinRates["rifle_vs_rifle"] = 0.500           // mirror
	tables.DuelWinRates["rifle_vs_awp"] = 0.463             // 120706 / 260824

	// AWP/Full Buy attacking
	tables.DuelWinRates["awp_vs_starter_pistol"] = 0.739  // 42245 / 57203
	tables.DuelWinRates["awp_vs_upgraded_pistol"] = 0.659 // 34594 / 52505
	tables.DuelWinRates["awp_vs_smg"] = 0.602             // 52723 / 87535
	tables.DuelWinRates["awp_vs_rifle"] = 0.537           // 140118 / 260824
	tables.DuelWinRates["awp_vs_awp"] = 0.500             // mirror

	// === MAP T-SIDE WIN RATES ===
	// A map's factor is its T win rate / MapBaseline (the all-maps rate BaseWinProb already carries).
	tables.MapBaseline = 0.489 // 68586 / 140364

	tables.MapAdjustments["de_ancient"] = 0.489  // 11568 / 23633 (CT-sided)
	tables.MapAdjustments["de_anubis"] = 0.515   // 7871 / 15287 (T-sided)
	tables.MapAdjustments["de_cache"] = 0.489    // 2269 / 4560 = 0.498: factor 1.018 (95% CI 0.988-1.047), shrunk to 1.000 (N_med 19258; docs/rating.md)
	tables.MapAdjustments["de_dust2"] = 0.508    // 10020 / 19712 (T-sided)
	tables.MapAdjustments["de_inferno"] = 0.500  // 9603 / 19199
	tables.MapAdjustments["de_mirage"] = 0.472   // 10828 / 22946 (CT-sided)
	tables.MapAdjustments["de_nuke"] = 0.467     // 9024 / 19317 (CT-sided)
	tables.MapAdjustments["de_overpass"] = 0.476 // 5637 / 11833 (CT-sided)
	tables.MapAdjustments["de_train"] = 0.456    // 1766 / 3877 (CT-sided)

	return tables
}
