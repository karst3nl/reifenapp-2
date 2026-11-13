package models

import (
	"testing"
	"time"
)

func TestCurrentSeasonBerlinBoundaries(t *testing.T) {
	for _, tc := range []struct{ stamp, kind, label, stored string }{
		{"2026-03-31T21:59:59Z", "winter", "Winter 2025/26", "Sommer"},
		{"2026-03-31T22:00:00Z", "summer", "Sommer 2026", "Winter"},
		{"2026-09-30T21:59:59Z", "summer", "Sommer 2026", "Winter"},
		{"2026-09-30T22:00:00Z", "winter", "Winter 2026/27", "Sommer"},
		{"2027-01-01T00:00:00Z", "winter", "Winter 2026/27", "Sommer"},
	} {
		now, err := time.Parse(time.RFC3339, tc.stamp)
		if err != nil {
			t.Fatal(err)
		}
		season := CurrentSeason(now)
		if season.Kind != tc.kind || season.Label != tc.label || season.StoredType != tc.stored || season.Start.After(now) || !season.End.After(now) {
			t.Fatal(tc, season)
		}
	}
}
