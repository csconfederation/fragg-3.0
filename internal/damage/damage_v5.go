//go:build damage_v5

package damage

import "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/events"

// Taken is v5's HealthDamageTaken: min(dmg_health, 100), or the HP left on a
// fatal hit.
func Taken(e events.PlayerHurt) int {
	if e.Health == 0 {
		return e.HealthDamageTaken
	}
	return min(e.HealthDamage, 100)
}
