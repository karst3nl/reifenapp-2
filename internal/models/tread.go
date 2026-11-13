package models

import (
	"math"
	"strconv"
	"strings"
)

type TreadWarning struct {
	Axle      string  `json:"axle"`
	Depth     float64 `json:"depth"`
	Threshold float64 `json:"threshold"`
}

// TreadWarnings is shared by search cards and the warning overview.
func (d SearchDetails) TreadWarnings() []TreadWarning {
	warnings := []TreadWarning{}
	threshold, known := map[string]float64{"Sommer": 3, "Winter": 4, "Ganzjahr": 4}[d.TireType]
	if !known {
		return warnings
	}
	for _, axle := range []struct{ key, value string }{{"VA", d.FrontDepth}, {"HA", d.RearDepth}} {
		depth, err := strconv.ParseFloat(strings.ReplaceAll(strings.TrimSpace(axle.value), ",", "."), 64)
		if err == nil && !math.IsNaN(depth) && !math.IsInf(depth, 0) && depth >= 0 && depth <= threshold {
			warnings = append(warnings, TreadWarning{axle.key, depth, threshold})
		}
	}
	return warnings
}
