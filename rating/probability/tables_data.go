package probability

// DefaultTables returns probability tables with empirically-derived values
// from parsing 54,584 rounds and 390,522 kills across competitive CS2 demos.
func DefaultTables() *ProbabilityTables {
	tables := NewProbabilityTables()

	// === BASE WIN PROBABILITIES ===
	// Format: "TvCT_bombStatus" (e.g., "5v4_none", "3v2_planted")
	// Values are T-side win probability derived from empirical data

	// No bomb planted
	tables.BaseWinProb["5v5_none"] = 0.494 // 39679 / 80398
	tables.BaseWinProb["5v4_none"] = 0.525 // 17158 / 32701
	tables.BaseWinProb["5v3_none"] = 0.715 // 11957 / 16730
	tables.BaseWinProb["5v2_none"] = 0.902 // 6695 / 7425
	tables.BaseWinProb["5v1_none"] = 0.989 // 2522 / 2549
	tables.BaseWinProb["5v0_none"] = 0.990 // Forced: all CTs dead, T wins

	tables.BaseWinProb["4v5_none"] = 0.475 // 16027 / 33736
	tables.BaseWinProb["4v4_none"] = 0.473 // 9036 / 19099
	tables.BaseWinProb["4v3_none"] = 0.545 // 7495 / 13763
	tables.BaseWinProb["4v2_none"] = 0.752 // 5951 / 7910
	tables.BaseWinProb["4v1_none"] = 0.951 // 3001 / 3154
	tables.BaseWinProb["4v0_none"] = 0.990 // Forced: all CTs dead, T wins

	tables.BaseWinProb["3v5_none"] = 0.305 // 6235 / 20406
	tables.BaseWinProb["3v4_none"] = 0.393 // 6048 / 15379
	tables.BaseWinProb["3v3_none"] = 0.429 // 5565 / 12978
	tables.BaseWinProb["3v2_none"] = 0.554 // 4720 / 8518
	tables.BaseWinProb["3v1_none"] = 0.829 // 3165 / 3816
	tables.BaseWinProb["3v0_none"] = 0.990 // Forced: all CTs dead, T wins

	tables.BaseWinProb["2v5_none"] = 0.124 // 1408 / 11385
	tables.BaseWinProb["2v4_none"] = 0.231 // 2587 / 11180
	tables.BaseWinProb["2v3_none"] = 0.299 // 3190 / 10687
	tables.BaseWinProb["2v2_none"] = 0.337 // 2627 / 7786
	tables.BaseWinProb["2v1_none"] = 0.511 // 1886 / 3689
	tables.BaseWinProb["2v0_none"] = 0.990 // Forced: all CTs dead, T wins

	tables.BaseWinProb["1v5_none"] = 0.018 // 93 / 5257
	tables.BaseWinProb["1v4_none"] = 0.100 // 415 / 4162
	tables.BaseWinProb["1v3_none"] = 0.299 // 978 / 3271
	tables.BaseWinProb["1v2_none"] = 0.558 // 1229 / 2203
	tables.BaseWinProb["1v1_none"] = 0.462 // 666 / 1442
	tables.BaseWinProb["1v0_none"] = 0.990 // Forced: all CTs dead, T wins

	tables.BaseWinProb["0v5_none"] = 0.010 // Forced: all Ts dead, CT wins
	tables.BaseWinProb["0v4_none"] = 0.010 // Forced: all Ts dead, CT wins
	tables.BaseWinProb["0v3_none"] = 0.010 // Forced: all Ts dead, CT wins
	tables.BaseWinProb["0v2_none"] = 0.010 // Forced: all Ts dead, CT wins
	tables.BaseWinProb["0v1_none"] = 0.010 // Forced: all Ts dead, CT wins
	tables.BaseWinProb["0v0_none"] = 0.500 // Draw state, shouldn't occur

	// Bomb planted - T-side advantage
	tables.BaseWinProb["5v5_planted"] = 0.794 // 910 / 1146
	tables.BaseWinProb["5v4_planted"] = 0.751 // 1508 / 2007
	tables.BaseWinProb["5v3_planted"] = 0.825 // 2779 / 3369
	tables.BaseWinProb["5v2_planted"] = 0.954 // 3904 / 4091
	tables.BaseWinProb["5v1_planted"] = 0.995 // 2963 / 2978
	tables.BaseWinProb["5v0_planted"] = 0.990 // Forced: all CTs dead + bomb planted

	tables.BaseWinProb["4v5_planted"] = 0.742 // 991 / 1335
	tables.BaseWinProb["4v4_planted"] = 0.733 // 2061 / 2813
	tables.BaseWinProb["4v3_planted"] = 0.741 // 3585 / 4841
	tables.BaseWinProb["4v2_planted"] = 0.860 // 5538 / 6437
	tables.BaseWinProb["4v1_planted"] = 0.970 // 4096 / 4224
	tables.BaseWinProb["4v0_planted"] = 0.990 // Forced: all CTs dead + bomb planted (raw 11/11)

	tables.BaseWinProb["3v5_planted"] = 0.543 // 649 / 1195
	tables.BaseWinProb["3v4_planted"] = 0.669 // 2130 / 3184
	tables.BaseWinProb["3v3_planted"] = 0.683 // 4104 / 6011
	tables.BaseWinProb["3v2_planted"] = 0.729 // 6038 / 8284
	tables.BaseWinProb["3v1_planted"] = 0.860 // 4717 / 5486
	tables.BaseWinProb["3v0_planted"] = 0.990 // Forced: all CTs dead + bomb planted (raw 17/17)

	tables.BaseWinProb["2v5_planted"] = 0.241 // 201 / 835
	tables.BaseWinProb["2v4_planted"] = 0.467 // 1173 / 2510
	tables.BaseWinProb["2v3_planted"] = 0.650 // 3520 / 5418
	tables.BaseWinProb["2v2_planted"] = 0.662 // 5316 / 8034
	tables.BaseWinProb["2v1_planted"] = 0.499 // 2328 / 4663
	tables.BaseWinProb["2v0_planted"] = 0.990 // Forced: all CTs dead + bomb planted (raw 27/27)

	tables.BaseWinProb["1v5_planted"] = 0.052 // 23 / 441
	tables.BaseWinProb["1v4_planted"] = 0.162 // 214 / 1321
	tables.BaseWinProb["1v3_planted"] = 0.500 // 1395 / 2789
	tables.BaseWinProb["1v2_planted"] = 0.932 // 3787 / 4063
	tables.BaseWinProb["1v1_planted"] = 0.549 // 825 / 1503
	tables.BaseWinProb["1v0_planted"] = 0.990 // Forced: all CTs dead + bomb planted (raw 52/52)

	// 0 T alive but bomb planted - depends on defuse time
	tables.BaseWinProb["0v5_planted"] = 0.000 // 0 / 99
	tables.BaseWinProb["0v4_planted"] = 0.004 // 2 / 517
	tables.BaseWinProb["0v3_planted"] = 0.005 // 8 / 1478
	tables.BaseWinProb["0v2_planted"] = 0.020 // 56 / 2746
	tables.BaseWinProb["0v1_planted"] = 0.118 // 309 / 2620
	tables.BaseWinProb["0v0_planted"] = 1.000 // Bomb explodes, T wins (249/249)

	// === DUEL WIN RATES ===
	// Format: "attacker_vs_defender" (e.g., "rifle_vs_pistol")
	// Values represent attacker win probability from empirical data

	// Starter Pistol attacking
	tables.DuelWinRates["starter_pistol_vs_starter_pistol"] = 0.500  // 30039 / 30039 (mirror)
	tables.DuelWinRates["starter_pistol_vs_upgraded_pistol"] = 0.520 // 3335 / 6415
	tables.DuelWinRates["starter_pistol_vs_smg"] = 0.257             // 2338 / 9109
	tables.DuelWinRates["starter_pistol_vs_rifle"] = 0.248           // 3748 / 15102
	tables.DuelWinRates["starter_pistol_vs_awp"] = 0.270             // 6037 / 22368

	// Upgraded Pistol attacking
	tables.DuelWinRates["upgraded_pistol_vs_starter_pistol"] = 0.480  // 3080 / 6415
	tables.DuelWinRates["upgraded_pistol_vs_upgraded_pistol"] = 0.500 // 334 / 334 (mirror)
	tables.DuelWinRates["upgraded_pistol_vs_smg"] = 0.348             // 872 / 2508
	tables.DuelWinRates["upgraded_pistol_vs_rifle"] = 0.360           // 2532 / 7040
	tables.DuelWinRates["upgraded_pistol_vs_awp"] = 0.349             // 7108 / 20393

	// SMG attacking
	tables.DuelWinRates["smg_vs_starter_pistol"] = 0.743  // 6771 / 9109
	tables.DuelWinRates["smg_vs_upgraded_pistol"] = 0.652 // 1636 / 2508
	tables.DuelWinRates["smg_vs_smg"] = 0.500             // 4738 / 4738 (mirror)
	tables.DuelWinRates["smg_vs_rifle"] = 0.431           // 8134 / 18879
	tables.DuelWinRates["smg_vs_awp"] = 0.401             // 14262 / 35511

	// Rifle attacking
	tables.DuelWinRates["rifle_vs_starter_pistol"] = 0.752  // 11354 / 15102
	tables.DuelWinRates["rifle_vs_upgraded_pistol"] = 0.640 // 4508 / 7040
	tables.DuelWinRates["rifle_vs_smg"] = 0.569             // 10745 / 18879
	tables.DuelWinRates["rifle_vs_rifle"] = 0.500           // 24356 / 24356 (mirror)
	tables.DuelWinRates["rifle_vs_awp"] = 0.468             // 48771 / 104315

	// AWP/Full Buy attacking
	tables.DuelWinRates["awp_vs_starter_pistol"] = 0.730  // 16331 / 22368
	tables.DuelWinRates["awp_vs_upgraded_pistol"] = 0.651 // 13285 / 20393
	tables.DuelWinRates["awp_vs_smg"] = 0.599             // 21249 / 35511
	tables.DuelWinRates["awp_vs_rifle"] = 0.532           // 55544 / 104315
	tables.DuelWinRates["awp_vs_awp"] = 0.500             // 89415 / 89415 (mirror)

	// === MAP T-SIDE WIN RATES ===
	// Empirically derived from demo data

	tables.MapAdjustments["de_ancient"] = 0.513  // 4120 / 8027
	tables.MapAdjustments["de_anubis"] = 0.564   // 2623 / 4652 (T-sided)
	tables.MapAdjustments["de_dust2"] = 0.519    // 3822 / 7366
	tables.MapAdjustments["de_inferno"] = 0.512  // 3848 / 7517
	tables.MapAdjustments["de_mirage"] = 0.498   // 3711 / 7457
	tables.MapAdjustments["de_nuke"] = 0.480     // 3832 / 7984 (CT-sided)
	tables.MapAdjustments["de_overpass"] = 0.488 // 2665 / 5457
	tables.MapAdjustments["de_train"] = 0.456    // 2793 / 6124 (CT-sided)

	return tables
}

// NormalizationConstants for converting swing to rating contribution.
const (
	// SwingToRatingScale converts average swing per round to a ~1.0 centered rating.
	// A player with +4% avg swing should get ~1.40 rating contribution.
	// A player with -3% avg swing should get ~0.70 rating contribution.
	SwingToRatingScale = 10.0

	// SwingRatingBaseline is the baseline for swing rating (average = 1.0).
	SwingRatingBaseline = 1.0

	// MinSwingRating is the minimum possible swing rating component.
	MinSwingRating = 0.40

	// MaxSwingRating is the maximum possible swing rating component.
	MaxSwingRating = 1.80
)
