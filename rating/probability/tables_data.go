package probability

// DefaultTables returns probability tables measured on CSC demos: 3,131 S19-S20 league (BO1/BO3,
// playoffs) and Combine maps, 68,872 rounds and 483,374 kills (engine/cmd/probtables;
// Core-planning#305).
func DefaultTables() *ProbabilityTables {
	tables := NewProbabilityTables()

	// === BASE WIN PROBABILITIES ===
	// Format: "TvCT_bombStatus" (e.g., "5v4_none", "3v2_planted")
	// Values are T-side win probability derived from empirical data

	// No bomb planted
	tables.BaseWinProb["5v5_none"] = 0.489 // 33562 / 68597
	tables.BaseWinProb["5v4_none"] = 0.708 // 22192 / 31342
	tables.BaseWinProb["5v3_none"] = 0.891 // 12281 / 13788
	tables.BaseWinProb["5v2_none"] = 0.978 // 5783 / 5912
	tables.BaseWinProb["5v1_none"] = 0.999 // 2341 / 2343
	tables.BaseWinProb["5v0_none"] = 0.990 // Forced: all CTs dead, T wins

	tables.BaseWinProb["4v5_none"] = 0.298 // 10959 / 36813
	tables.BaseWinProb["4v4_none"] = 0.511 // 17290 / 33868
	tables.BaseWinProb["4v3_none"] = 0.751 // 16241 / 21616
	tables.BaseWinProb["4v2_none"] = 0.929 // 10443 / 11238
	tables.BaseWinProb["4v1_none"] = 0.993 // 4693 / 4724
	tables.BaseWinProb["4v0_none"] = 0.990 // Forced: all CTs dead, T wins

	tables.BaseWinProb["3v5_none"] = 0.130 // 2462 / 18968
	tables.BaseWinProb["3v4_none"] = 0.280 // 7230 / 25835
	tables.BaseWinProb["3v3_none"] = 0.515 // 10895 / 21155
	tables.BaseWinProb["3v2_none"] = 0.785 // 9848 / 12539
	tables.BaseWinProb["3v1_none"] = 0.965 // 5255 / 5443
	tables.BaseWinProb["3v0_none"] = 0.990 // Forced: all CTs dead, T wins

	tables.BaseWinProb["2v5_none"] = 0.028 // 275 / 9854
	tables.BaseWinProb["2v4_none"] = 0.086 // 1504 / 17441
	tables.BaseWinProb["2v3_none"] = 0.224 // 3821 / 17068
	tables.BaseWinProb["2v2_none"] = 0.483 // 5336 / 11052
	tables.BaseWinProb["2v1_none"] = 0.813 // 3947 / 4855
	tables.BaseWinProb["2v0_none"] = 0.990 // Forced: all CTs dead, T wins

	tables.BaseWinProb["1v5_none"] = 0.002 // 9 / 5290
	tables.BaseWinProb["1v4_none"] = 0.008 // 88 / 11305
	tables.BaseWinProb["1v3_none"] = 0.037 // 469 / 12753
	tables.BaseWinProb["1v2_none"] = 0.130 // 1158 / 8933
	tables.BaseWinProb["1v1_none"] = 0.401 // 1597 / 3982
	tables.BaseWinProb["1v0_none"] = 0.990 // Forced: all CTs dead, T wins

	tables.BaseWinProb["0v5_none"] = 0.010 // Forced: all Ts dead, CT wins
	tables.BaseWinProb["0v4_none"] = 0.010 // Forced: all Ts dead, CT wins
	tables.BaseWinProb["0v3_none"] = 0.010 // Forced: all Ts dead, CT wins
	tables.BaseWinProb["0v2_none"] = 0.010 // Forced: all Ts dead, CT wins
	tables.BaseWinProb["0v1_none"] = 0.010 // Forced: all Ts dead, CT wins
	tables.BaseWinProb["0v0_none"] = 0.500 // Draw state, shouldn't occur

	// Bomb planted - T-side advantage
	tables.BaseWinProb["5v5_planted"] = 0.785 // 557 / 710
	tables.BaseWinProb["5v4_planted"] = 0.856 // 1301 / 1519
	tables.BaseWinProb["5v3_planted"] = 0.946 // 2352 / 2486
	tables.BaseWinProb["5v2_planted"] = 0.994 // 2748 / 2764
	tables.BaseWinProb["5v1_planted"] = 0.998 // 2512 / 2517
	tables.BaseWinProb["5v0_planted"] = 0.990 // Forced: all CTs dead + bomb planted

	tables.BaseWinProb["4v5_planted"] = 0.512 // 363 / 709
	tables.BaseWinProb["4v4_planted"] = 0.663 // 1749 / 2639
	tables.BaseWinProb["4v3_planted"] = 0.856 // 4785 / 5589
	tables.BaseWinProb["4v2_planted"] = 0.971 // 7590 / 7814
	tables.BaseWinProb["4v1_planted"] = 0.998 // 7615 / 7630
	tables.BaseWinProb["4v0_planted"] = 0.990 // Forced: all CTs dead + bomb planted

	tables.BaseWinProb["3v5_planted"] = 0.237 // 135 / 569
	tables.BaseWinProb["3v4_planted"] = 0.415 // 1145 / 2756
	tables.BaseWinProb["3v3_planted"] = 0.657 // 4520 / 6881
	tables.BaseWinProb["3v2_planted"] = 0.886 // 9738 / 10991
	tables.BaseWinProb["3v1_planted"] = 0.989 // 12245 / 12383
	tables.BaseWinProb["3v0_planted"] = 0.990 // Forced: all CTs dead + bomb planted

	tables.BaseWinProb["2v5_planted"] = 0.071 // 29 / 407
	tables.BaseWinProb["2v4_planted"] = 0.146 // 315 / 2161
	tables.BaseWinProb["2v3_planted"] = 0.341 // 2116 / 6200
	tables.BaseWinProb["2v2_planted"] = 0.645 // 6962 / 10793
	tables.BaseWinProb["2v1_planted"] = 0.915 // 11807 / 12904
	tables.BaseWinProb["2v0_planted"] = 0.990 // Forced: all CTs dead + bomb planted

	tables.BaseWinProb["1v5_planted"] = 0.003 // 1 / 293
	tables.BaseWinProb["1v4_planted"] = 0.027 // 40 / 1504
	tables.BaseWinProb["1v3_planted"] = 0.087 // 384 / 4437
	tables.BaseWinProb["1v2_planted"] = 0.274 // 2229 / 8141
	tables.BaseWinProb["1v1_planted"] = 0.627 // 6328 / 10091
	tables.BaseWinProb["1v0_planted"] = 0.990 // Forced: all CTs dead + bomb planted

	// 0 T alive but bomb planted - depends on defuse time
	tables.BaseWinProb["0v5_planted"] = 0.000 // 0 / 160
	tables.BaseWinProb["0v4_planted"] = 0.005 // 4 / 783
	tables.BaseWinProb["0v3_planted"] = 0.007 // 17 / 2340
	tables.BaseWinProb["0v2_planted"] = 0.040 // 157 / 3940
	tables.BaseWinProb["0v1_planted"] = 0.220 // 825 / 3742
	tables.BaseWinProb["0v0_planted"] = 1.000 // Bomb explodes, T wins

	// Empirical cells can violate alive-count / plant monotonicity due to
	// selection bias and small samples. Enforce invariants so a kill or plant
	// never lowers the beneficiary's modeled win probability.
	EnforceMonotonicity(tables)

	// === DUEL WIN RATES ===
	// Format: "attacker_vs_defender" (e.g., "rifle_vs_pistol")
	// Values represent attacker win probability from empirical data

	// Starter Pistol attacking
	tables.DuelWinRates["starter_pistol_vs_starter_pistol"] = 0.500  // mirror
	tables.DuelWinRates["starter_pistol_vs_upgraded_pistol"] = 0.523 // 4422 / 8458
	tables.DuelWinRates["starter_pistol_vs_smg"] = 0.265             // 2788 / 10534
	tables.DuelWinRates["starter_pistol_vs_rifle"] = 0.242           // 4584 / 18946
	tables.DuelWinRates["starter_pistol_vs_awp"] = 0.258             // 7250 / 28147

	// Upgraded Pistol attacking
	tables.DuelWinRates["upgraded_pistol_vs_starter_pistol"] = 0.477  // 4036 / 8458
	tables.DuelWinRates["upgraded_pistol_vs_upgraded_pistol"] = 0.500 // mirror
	tables.DuelWinRates["upgraded_pistol_vs_smg"] = 0.366             // 941 / 2570
	tables.DuelWinRates["upgraded_pistol_vs_rifle"] = 0.354           // 2764 / 7805
	tables.DuelWinRates["upgraded_pistol_vs_awp"] = 0.345             // 8908 / 25850

	// SMG attacking
	tables.DuelWinRates["smg_vs_starter_pistol"] = 0.735  // 7746 / 10534
	tables.DuelWinRates["smg_vs_upgraded_pistol"] = 0.634 // 1629 / 2570
	tables.DuelWinRates["smg_vs_smg"] = 0.500             // mirror
	tables.DuelWinRates["smg_vs_rifle"] = 0.427           // 8989 / 21035
	tables.DuelWinRates["smg_vs_awp"] = 0.399             // 17203 / 43121

	// Rifle attacking
	tables.DuelWinRates["rifle_vs_starter_pistol"] = 0.758  // 14362 / 18946
	tables.DuelWinRates["rifle_vs_upgraded_pistol"] = 0.646 // 5041 / 7805
	tables.DuelWinRates["rifle_vs_smg"] = 0.573             // 12046 / 21035
	tables.DuelWinRates["rifle_vs_rifle"] = 0.500           // mirror
	tables.DuelWinRates["rifle_vs_awp"] = 0.462             // 58773 / 127221

	// AWP/Full Buy attacking
	tables.DuelWinRates["awp_vs_starter_pistol"] = 0.742  // 20897 / 28147
	tables.DuelWinRates["awp_vs_upgraded_pistol"] = 0.655 // 16942 / 25850
	tables.DuelWinRates["awp_vs_smg"] = 0.601             // 25918 / 43121
	tables.DuelWinRates["awp_vs_rifle"] = 0.538           // 68448 / 127221
	tables.DuelWinRates["awp_vs_awp"] = 0.500             // mirror

	// === MAP T-SIDE WIN RATES ===
	// A map's factor is its T win rate / MapBaseline (the all-maps rate BaseWinProb already carries).
	tables.MapBaseline = 0.489 // 33703 / 68872

	tables.MapAdjustments["de_ancient"] = 0.489  // 5466 / 11179 (CT-sided)
	tables.MapAdjustments["de_anubis"] = 0.526   // 4776 / 9086 (T-sided)
	tables.MapAdjustments["de_dust2"] = 0.509    // 4895 / 9626
	tables.MapAdjustments["de_inferno"] = 0.505  // 4838 / 9576
	tables.MapAdjustments["de_mirage"] = 0.465   // 5268 / 11324 (CT-sided)
	tables.MapAdjustments["de_nuke"] = 0.463     // 4451 / 9613 (CT-sided)
	tables.MapAdjustments["de_overpass"] = 0.473 // 3597 / 7597 (CT-sided)
	tables.MapAdjustments["de_train"] = 0.473    // 412 / 871

	return tables
}
