package probability

// DefaultTables returns probability tables measured on CSC demos: 6,388 S18-S21 league (BO1/BO3,
// playoffs) and Combine maps, 140,335 rounds and 986,908 kills (engine/cmd/probtables;
// Core-planning#305).
func DefaultTables() *ProbabilityTables {
	tables := NewProbabilityTables()

	// === BASE WIN PROBABILITIES ===
	// Format: "TvCT_bombStatus" (e.g., "5v4_none", "3v2_planted"), T-side win probability. A decided
	// round (a side eliminated, except Ts dead with the bomb planted) never reads the table.

	// No bomb planted
	tables.BaseWinProb["5v5_none"] = 0.489 // 68350 / 139877
	tables.BaseWinProb["5v4_none"] = 0.707 // 45143 / 63837
	tables.BaseWinProb["5v3_none"] = 0.887 // 24960 / 28125
	tables.BaseWinProb["5v2_none"] = 0.978 // 11749 / 12017
	tables.BaseWinProb["5v1_none"] = 0.999 // 4755 / 4760

	tables.BaseWinProb["4v5_none"] = 0.298 // 22353 / 75085
	tables.BaseWinProb["4v4_none"] = 0.510 // 35326 / 69258
	tables.BaseWinProb["4v3_none"] = 0.748 // 33062 / 44212
	tables.BaseWinProb["4v2_none"] = 0.927 // 21323 / 23003
	tables.BaseWinProb["4v1_none"] = 0.994 // 9508 / 9564

	tables.BaseWinProb["3v5_none"] = 0.129 // 4950 / 38466
	tables.BaseWinProb["3v4_none"] = 0.281 // 14865 / 52878
	tables.BaseWinProb["3v3_none"] = 0.513 // 22334 / 43545
	tables.BaseWinProb["3v2_none"] = 0.782 // 20324 / 25985
	tables.BaseWinProb["3v1_none"] = 0.964 // 10924 / 11332

	tables.BaseWinProb["2v5_none"] = 0.028 // 564 / 19912
	tables.BaseWinProb["2v4_none"] = 0.090 // 3200 / 35421
	tables.BaseWinProb["2v3_none"] = 0.228 // 7966 / 34870
	tables.BaseWinProb["2v2_none"] = 0.487 // 11104 / 22818
	tables.BaseWinProb["2v1_none"] = 0.816 // 8373 / 10262

	tables.BaseWinProb["1v5_none"] = 0.002 // 23 / 10767
	tables.BaseWinProb["1v4_none"] = 0.009 // 209 / 23012
	tables.BaseWinProb["1v3_none"] = 0.038 // 1001 / 26062
	tables.BaseWinProb["1v2_none"] = 0.135 // 2491 / 18515
	tables.BaseWinProb["1v1_none"] = 0.416 // 3492 / 8396

	// Bomb planted - T-side advantage
	tables.BaseWinProb["5v5_planted"] = 0.770 // 1080 / 1402
	tables.BaseWinProb["5v4_planted"] = 0.859 // 2573 / 2997
	tables.BaseWinProb["5v3_planted"] = 0.947 // 4668 / 4931
	tables.BaseWinProb["5v2_planted"] = 0.994 // 5577 / 5610
	tables.BaseWinProb["5v1_planted"] = 0.999 // 5111 / 5118

	tables.BaseWinProb["4v5_planted"] = 0.511 // 740 / 1447
	tables.BaseWinProb["4v4_planted"] = 0.666 // 3514 / 5275
	tables.BaseWinProb["4v3_planted"] = 0.852 // 9424 / 11057
	tables.BaseWinProb["4v2_planted"] = 0.969 // 15111 / 15592
	tables.BaseWinProb["4v1_planted"] = 0.998 // 15397 / 15427

	tables.BaseWinProb["3v5_planted"] = 0.236 // 285 / 1206
	tables.BaseWinProb["3v4_planted"] = 0.407 // 2252 / 5536
	tables.BaseWinProb["3v3_planted"] = 0.651 // 9120 / 14006
	tables.BaseWinProb["3v2_planted"] = 0.881 // 19690 / 22346
	tables.BaseWinProb["3v1_planted"] = 0.988 // 24611 / 24911

	tables.BaseWinProb["2v5_planted"] = 0.065 // 55 / 845
	tables.BaseWinProb["2v4_planted"] = 0.147 // 657 / 4462
	tables.BaseWinProb["2v3_planted"] = 0.338 // 4279 / 12650
	tables.BaseWinProb["2v2_planted"] = 0.644 // 14136 / 21941
	tables.BaseWinProb["2v1_planted"] = 0.916 // 24090 / 26300

	tables.BaseWinProb["1v5_planted"] = 0.002 // 1 / 598
	tables.BaseWinProb["1v4_planted"] = 0.027 // 83 / 3103
	tables.BaseWinProb["1v3_planted"] = 0.084 // 774 / 9160
	tables.BaseWinProb["1v2_planted"] = 0.272 // 4593 / 16891
	tables.BaseWinProb["1v1_planted"] = 0.631 // 13196 / 20929

	// 0 T alive but bomb planted - depends on defuse time
	tables.BaseWinProb["0v5_planted"] = 0.000 // 0 / 346
	tables.BaseWinProb["0v4_planted"] = 0.004 // 6 / 1645
	tables.BaseWinProb["0v3_planted"] = 0.009 // 43 / 4748
	tables.BaseWinProb["0v2_planted"] = 0.043 // 356 / 8237
	tables.BaseWinProb["0v1_planted"] = 0.243 // 1927 / 7943

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
	tables.DuelWinRates["starter_pistol_vs_smg"] = 0.265             // 5580 / 21051
	tables.DuelWinRates["starter_pistol_vs_rifle"] = 0.245           // 9409 / 38344
	tables.DuelWinRates["starter_pistol_vs_awp"] = 0.262             // 14954 / 57185

	// Upgraded Pistol attacking
	tables.DuelWinRates["upgraded_pistol_vs_starter_pistol"] = 0.481  // 8316 / 17301
	tables.DuelWinRates["upgraded_pistol_vs_upgraded_pistol"] = 0.500 // mirror
	tables.DuelWinRates["upgraded_pistol_vs_smg"] = 0.365             // 1958 / 5368
	tables.DuelWinRates["upgraded_pistol_vs_rifle"] = 0.358           // 5720 / 15986
	tables.DuelWinRates["upgraded_pistol_vs_awp"] = 0.341             // 17909 / 52498

	// SMG attacking
	tables.DuelWinRates["smg_vs_starter_pistol"] = 0.735  // 15471 / 21051
	tables.DuelWinRates["smg_vs_upgraded_pistol"] = 0.635 // 3410 / 5368
	tables.DuelWinRates["smg_vs_smg"] = 0.500             // mirror
	tables.DuelWinRates["smg_vs_rifle"] = 0.431           // 18813 / 43653
	tables.DuelWinRates["smg_vs_awp"] = 0.398             // 34803 / 87518

	// Rifle attacking
	tables.DuelWinRates["rifle_vs_starter_pistol"] = 0.755  // 28935 / 38344
	tables.DuelWinRates["rifle_vs_upgraded_pistol"] = 0.642 // 10266 / 15986
	tables.DuelWinRates["rifle_vs_smg"] = 0.569             // 24840 / 43653
	tables.DuelWinRates["rifle_vs_rifle"] = 0.500           // mirror
	tables.DuelWinRates["rifle_vs_awp"] = 0.463             // 120685 / 260788

	// AWP/Full Buy attacking
	tables.DuelWinRates["awp_vs_starter_pistol"] = 0.738  // 42231 / 57185
	tables.DuelWinRates["awp_vs_upgraded_pistol"] = 0.659 // 34589 / 52498
	tables.DuelWinRates["awp_vs_smg"] = 0.602             // 52715 / 87518
	tables.DuelWinRates["awp_vs_rifle"] = 0.537           // 140103 / 260788
	tables.DuelWinRates["awp_vs_awp"] = 0.500             // mirror

	// === MAP T-SIDE WIN RATES ===
	// A map's factor is its T win rate / MapBaseline (the all-maps rate BaseWinProb already carries).
	tables.MapBaseline = 0.489 // 68571 / 140335

	tables.MapAdjustments["de_ancient"] = 0.489  // 11568 / 23633 (CT-sided)
	tables.MapAdjustments["de_anubis"] = 0.515   // 7871 / 15287 (T-sided)
	tables.MapAdjustments["de_cache"] = 0.489    // 2269 / 4560 = 0.498: factor 1.018 (95% CI 0.988-1.047), shrunk to 1.000 (N_med 19243.5; docs/rating.md)
	tables.MapAdjustments["de_dust2"] = 0.508    // 10020 / 19712 (T-sided)
	tables.MapAdjustments["de_inferno"] = 0.500  // 9588 / 19170
	tables.MapAdjustments["de_mirage"] = 0.472   // 10828 / 22946 (CT-sided)
	tables.MapAdjustments["de_nuke"] = 0.467     // 9024 / 19317 (CT-sided)
	tables.MapAdjustments["de_overpass"] = 0.476 // 5637 / 11833 (CT-sided)
	tables.MapAdjustments["de_train"] = 0.456    // 1766 / 3877 (CT-sided)

	return tables
}
