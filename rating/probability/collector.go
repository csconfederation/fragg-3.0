package probability

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
)

// DataCollector collects probability data from demo parsing.
// Used in cumulative mode to build empirical probability tables.
type DataCollector struct {
	mu            sync.Mutex
	data          *CollectedData
	pendingStates []string // State keys captured during round, attributed at round end
}

// CollectedData holds all collected probability data.
type CollectedData struct {
	StateOutcomes map[string]*StateOutcomeData `json:"state_outcomes"`
	DuelOutcomes  map[string]*DuelOutcomeData  `json:"duel_outcomes"`
	MapData       map[string]*MapData          `json:"map_data"`
	TotalRounds   int                          `json:"total_rounds"`
	TotalKills    int                          `json:"total_kills"`
}

// StateOutcomeData tracks win/loss for a specific game state.
type StateOutcomeData struct {
	TWins  int `json:"t_wins"`
	CTWins int `json:"ct_wins"`
}

// DuelOutcomeData tracks duel outcomes between equipment categories.
type DuelOutcomeData struct {
	AttackerWins int `json:"attacker_wins"`
	DefenderWins int `json:"defender_wins"`
}

// MapData tracks T/CT win rates per map.
type MapData struct {
	TWins  int `json:"t_wins"`
	CTWins int `json:"ct_wins"`
}

// NewDataCollector creates a new data collector.
func NewDataCollector() *DataCollector {
	return &DataCollector{
		data: &CollectedData{
			StateOutcomes: make(map[string]*StateOutcomeData),
			DuelOutcomes:  make(map[string]*DuelOutcomeData),
			MapData:       make(map[string]*MapData),
		},
	}
}

// stateKey generates a readable key for a game state.
// Format: "5v4_none" or "3v2_planted"
func stateKey(tAlive, ctAlive int, bombPlanted bool) string {
	bombStatus := "none"
	if bombPlanted {
		bombStatus = "planted"
	}
	return fmt.Sprintf("%dv%d_%s", tAlive, ctAlive, bombStatus)
}

// duelKey generates a readable key for a duel.
// Format: "rifle_vs_pistol" or "awp_vs_smg"
func duelKey(attackerCat, defenderCat EconomyCategory) string {
	return fmt.Sprintf("%s_vs_%s", attackerCat.String(), defenderCat.String())
}

// RecordRoundStart records the initial state of a round and resets pending states.
func (dc *DataCollector) RecordRoundStart(tAlive, ctAlive int, bombPlanted bool, mapName string) {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	dc.pendingStates = nil // Reset for new round
}

// RecordStateSnapshot captures the current game state for later attribution.
// Call this before each significant event (kill, bomb plant, etc).
func (dc *DataCollector) RecordStateSnapshot(tAlive, ctAlive int, bombPlanted bool) {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	key := stateKey(tAlive, ctAlive, bombPlanted)
	dc.pendingStates = append(dc.pendingStates, key)
}

// RecordRoundEnd records the outcome of a round.
// Attributes all pending state snapshots to the round winner.
func (dc *DataCollector) RecordRoundEnd(
	tAlive, ctAlive int,
	bombPlanted bool,
	winner common.Team,
	mapName string,
) {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	dc.data.TotalRounds++

	// Attribute all pending state snapshots to the winner
	for _, key := range dc.pendingStates {
		if dc.data.StateOutcomes[key] == nil {
			dc.data.StateOutcomes[key] = &StateOutcomeData{}
		}
		if winner == common.TeamTerrorists {
			dc.data.StateOutcomes[key].TWins++
		} else {
			dc.data.StateOutcomes[key].CTWins++
		}
	}
	dc.pendingStates = nil // Clear for next round

	// Record map data
	if dc.data.MapData[mapName] == nil {
		dc.data.MapData[mapName] = &MapData{}
	}
	if winner == common.TeamTerrorists {
		dc.data.MapData[mapName].TWins++
	} else {
		dc.data.MapData[mapName].CTWins++
	}
}

// RecordKill records the outcome of a kill/duel.
// Records bidirectional data: attacker won in A_V key, defender lost in V_A key.
// This allows computing win rates for any equipment matchup.
func (dc *DataCollector) RecordKill(
	attackerEquip, victimEquip float64,
) {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	dc.data.TotalKills++

	attackerCat := CategorizeEquipment(attackerEquip)
	victimCat := CategorizeEquipment(victimEquip)

	// Record attacker win: in matchup A vs V, A won
	forwardKey := duelKey(attackerCat, victimCat)
	if dc.data.DuelOutcomes[forwardKey] == nil {
		dc.data.DuelOutcomes[forwardKey] = &DuelOutcomeData{}
	}
	dc.data.DuelOutcomes[forwardKey].AttackerWins++

	// Record defender loss: in matchup V vs A, V lost (A defended successfully).
	// Skip for mirror matchups (e.g. awp_vs_awp) where forward and reverse keys
	// are identical — recording both would double-count every kill.
	if attackerCat != victimCat {
		reverseKey := duelKey(victimCat, attackerCat)
		if dc.data.DuelOutcomes[reverseKey] == nil {
			dc.data.DuelOutcomes[reverseKey] = &DuelOutcomeData{}
		}
		dc.data.DuelOutcomes[reverseKey].DefenderWins++
	}
}

// Merge combines data from another collector.
func (dc *DataCollector) Merge(other *DataCollector) {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	other.mu.Lock()
	defer other.mu.Unlock()

	dc.data.TotalRounds += other.data.TotalRounds
	dc.data.TotalKills += other.data.TotalKills

	for key, outcome := range other.data.StateOutcomes {
		if dc.data.StateOutcomes[key] == nil {
			dc.data.StateOutcomes[key] = &StateOutcomeData{}
		}
		dc.data.StateOutcomes[key].TWins += outcome.TWins
		dc.data.StateOutcomes[key].CTWins += outcome.CTWins
	}

	for key, outcome := range other.data.DuelOutcomes {
		if dc.data.DuelOutcomes[key] == nil {
			dc.data.DuelOutcomes[key] = &DuelOutcomeData{}
		}
		dc.data.DuelOutcomes[key].AttackerWins += outcome.AttackerWins
		dc.data.DuelOutcomes[key].DefenderWins += outcome.DefenderWins
	}

	for mapName, mapData := range other.data.MapData {
		if dc.data.MapData[mapName] == nil {
			dc.data.MapData[mapName] = &MapData{}
		}
		dc.data.MapData[mapName].TWins += mapData.TWins
		dc.data.MapData[mapName].CTWins += mapData.CTWins
	}
}

// SaveToFile saves collected data to a JSON file.
func (dc *DataCollector) SaveToFile(filepath string) error {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	data, err := json.MarshalIndent(dc.data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath, data, 0644)
}

// LoadFromFile loads collected data from a JSON file.
func (dc *DataCollector) LoadFromFile(filepath string) error {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	data, err := os.ReadFile(filepath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No existing data, start fresh
		}
		return err
	}

	return json.Unmarshal(data, dc.data)
}

// GetData returns the collected data (for building tables).
func (dc *DataCollector) GetData() *CollectedData {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	return dc.data
}

// BuildTablesFromData creates probability tables from collected data.
// Keys are already in readable format (e.g., "5v4_none", "rifle_vs_smg").
func (dc *DataCollector) BuildTablesFromData() *ProbabilityTables {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	tables := DefaultTables()

	// Update base probabilities from state outcomes
	// Keys are already in "TvCT_status" format (e.g., "5v4_none")
	for key, outcome := range dc.data.StateOutcomes {
		total := outcome.TWins + outcome.CTWins
		if total < 10 {
			continue // Need minimum sample size
		}
		tables.BaseWinProb[key] = float64(outcome.TWins) / float64(total)
	}

	// Update duel win rates
	// Keys are already in "attacker_vs_defender" format (e.g., "rifle_vs_smg")
	for key, outcome := range dc.data.DuelOutcomes {
		total := outcome.AttackerWins + outcome.DefenderWins
		if total < 10 {
			continue
		}
		// Mirror matchups (e.g. awp_vs_awp) have DefenderWins=0 because the
		// reverse key is identical to the forward key. Same-category duels are
		// inherently 50/50.
		if outcome.DefenderWins == 0 && outcome.AttackerWins > 0 {
			tables.DuelWinRates[key] = 0.5
		} else {
			tables.DuelWinRates[key] = float64(outcome.AttackerWins) / float64(total)
		}
	}

	// Update map adjustments
	for mapName, mapData := range dc.data.MapData {
		total := mapData.TWins + mapData.CTWins
		if total < 20 {
			continue
		}
		tables.MapAdjustments[mapName] = float64(mapData.TWins) / float64(total)
	}

	return tables
}

// GetStats returns summary statistics.
func (dc *DataCollector) GetStats() (rounds, kills int) {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	return dc.data.TotalRounds, dc.data.TotalKills
}
