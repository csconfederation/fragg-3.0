package parser

import (
	"testing"
)

func TestGetFlashAssistsFiltersStaleAndDedupes(t *testing.T) {
	dt := NewDamageTracker()
	tickRate := 64.0

	// Fresh 2s flash from attacker 1
	dt.RecordFlash(1, 100, 2.0, 1000)
	// Stale 2s flash from attacker 2 (happened 10s ago)
	dt.RecordFlash(2, 100, 2.0, 1000-int(10*tickRate))
	// Second flash from attacker 1 with longer duration — should win dedupe
	dt.RecordFlash(1, 100, 3.0, 1050)

	assists := dt.GetFlashAssists(100, 1100, tickRate)
	if len(assists) != 1 {
		t.Fatalf("expected 1 assist after stale filter + dedupe, got %d", len(assists))
	}
	if assists[0].PlayerID != 1 {
		t.Fatalf("expected attacker 1, got %d", assists[0].PlayerID)
	}
	if assists[0].Duration != 3.0 {
		t.Fatalf("expected duration 3.0, got %f", assists[0].Duration)
	}
}

func TestGetFlashAssistsKeepsActiveFlash(t *testing.T) {
	dt := NewDamageTracker()
	dt.RecordFlash(7, 50, 1.5, 640)
	// 1.5s * 64 = 96 ticks; at tick 700 still active
	assists := dt.GetFlashAssists(50, 700, 64)
	if len(assists) != 1 {
		t.Fatalf("expected active flash, got %d", len(assists))
	}
}
