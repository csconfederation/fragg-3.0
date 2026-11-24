package swing

// ExitFragMode controls probability swing for kills after the round is decided.
type ExitFragMode string

const (
	ExitFragsZero ExitFragMode = "zero"
	ExitFragsTiny ExitFragMode = "tiny"
)

// Config holds tunable parameters for round swing calculation.
type Config struct {
	ExitFrags              ExitFragMode
	ResidualEnabled        bool
	ResidualMax            float64
	SavePenaltyWeight      float64
	PlantPlanterShare      float64
	DefuseDefuserShare     float64
	KillFinalHitBase       float64
	KillDamageShareWeight  float64
	TradeKillMultiplier    float64
	SurvivalCreditEnabled  bool
	SurvivalCreditMaxShare float64
	ZeroSumTolerance       float64
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
