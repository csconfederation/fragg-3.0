package swing

// ExitFragMode controls probability swing for kills after the round is decided.
type ExitFragMode string

const (
	ExitFragsZero ExitFragMode = "zero"
	ExitFragsTiny ExitFragMode = "tiny"
)

// Config holds tunable parameters for round swing calculation.
type Config struct {
	ExitFrags              ExitFragMode `json:"swing_exit_frags"`
	ResidualEnabled        bool         `json:"swing_residual_enabled"`
	ResidualMax            float64      `json:"swing_residual_max"`
	SavePenaltyWeight      float64      `json:"swing_save_penalty_weight"`
	PlantPlanterShare      float64      `json:"swing_plant_planter_share"`
	DefuseDefuserShare     float64      `json:"swing_defuse_defuser_share"`
	KillFinalHitBase       float64      `json:"swing_kill_final_hit_base"`
	KillDamageShareWeight  float64      `json:"swing_kill_damage_share_weight"`
	TradeKillMultiplier    float64      `json:"swing_trade_kill_multiplier"`
	SurvivalCreditEnabled  bool         `json:"swing_survival_credit_enabled"`
	SurvivalCreditMaxShare float64      `json:"swing_survival_credit_max_share"`
	ZeroSumTolerance       float64      `json:"swing_zero_sum_tolerance"`
}

// DefaultConfig returns the recommended defaults from the swing improvement design.
func DefaultConfig() Config {
	return Config{
		ExitFrags:              ExitFragsZero,
		ResidualEnabled:        true,
		ResidualMax:            0.35,
		SavePenaltyWeight:      0.35,
		PlantPlanterShare:      0.45,
		DefuseDefuserShare:     0.60,
		KillFinalHitBase:       0.35,
		KillDamageShareWeight:  0.45,
		TradeKillMultiplier:    0.80,
		SurvivalCreditEnabled:  false,
		SurvivalCreditMaxShare: 0.05,
		ZeroSumTolerance:       0.005,
	}
}
