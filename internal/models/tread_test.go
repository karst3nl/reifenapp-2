package models

import "testing"

func TestTreadWarnings(t *testing.T) {
	for _, tc := range []struct {
		kind, front, rear string
		count             int
	}{{"Sommer", "3.0", "3.5", 1}, {"Winter", "4.0", "4.5", 1}, {"Ganzjahr", "0.0", "4.0", 2}, {"Sommer", "3,0", "", 1}, {"Sommer", "", "", 0}, {"Winter", "NaN", "-1", 0}, {"Winter", "+Inf", "bad", 0}, {"Unbekannt", "1.0", "1.0", 0}, {"Sommer", "10.0", "10.0", 0}} {
		d := SearchDetails{TireType: tc.kind, FrontDepth: tc.front, RearDepth: tc.rear}
		warnings := d.TreadWarnings()
		if len(warnings) != tc.count {
			t.Fatal(tc, warnings)
		}
		if tc.count > 0 && warnings[0].Axle != "VA" {
			t.Fatal(warnings)
		}
	}
	d := SearchDetails{TireType: "Winter", RearDepth: "2.5"}
	w := d.TreadWarnings()
	if len(w) != 1 || w[0].Axle != "HA" || w[0].Depth != 2.5 || w[0].Threshold != 4 {
		t.Fatal(w)
	}
}
