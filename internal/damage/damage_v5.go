//go:build damage_v5

package damage

import "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/events"

// Taken is v5.2.0's HealthDamageTaken (game_events.go playerHurt):
// min(dmg_health, 100), or the victim's entity health on a fatal hit.
func Taken(e events.PlayerHurt) int {
	if e.Health == 0 && e.Player != nil {
		return e.Player.Health()
	}
	return min(e.HealthDamage, 100)
}
