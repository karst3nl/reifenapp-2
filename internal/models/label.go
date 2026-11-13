package models

import (
	"errors"
	"math"
	"regexp"
)

type LabelElement struct {
	Field  string  `json:"field"`
	Text   string  `json:"text"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
	Size   float64 `json:"size"`
	Bold   bool    `json:"bold"`
}
type LabelSettings struct {
	Version     int            `json:"version"`
	Enabled     bool           `json:"enabled"`
	Queue       string         `json:"queue"`
	Copies      int            `json:"copies"`
	Storage     bool           `json:"storage"`
	WheelChange bool           `json:"wheel_change"`
	Elements    []LabelElement `json:"elements"`
}

var LabelFields = map[string]string{"text": "Freitext", "place": "Lagerplatz", "name": "Kunde", "plate": "Kennzeichen", "car": "Fahrzeug", "date": "Datum", "operation": "Vorgang", "tire_type": "Reifenart", "rim_type": "Felgenart", "front": "Reifengröße VA", "rear": "Reifengröße HA", "front_depth": "Profiltiefe VA", "rear_depth": "Profiltiefe HA", "notes": "Hinweise"}
var queueName = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_.-]{0,126}$`)

func ValidPrinterQueue(s string) bool { return queueName.MatchString(s) }
func DefaultLabelSettings() LabelSettings {
	return LabelSettings{Version: 1, Copies: 1, Storage: true, WheelChange: true, Elements: []LabelElement{
		{Field: "place", X: 3, Y: 3, Width: 72, Height: 10, Size: 22, Bold: true},
		{Field: "name", X: 3, Y: 14, Width: 72, Height: 8, Size: 12, Bold: true},
		{Field: "plate", X: 3, Y: 23, Width: 72, Height: 7, Size: 14, Bold: true},
		{Field: "tire_type", X: 3, Y: 31, Width: 72, Height: 5, Size: 10},
		{Field: "date", X: 3, Y: 37, Width: 72, Height: 5, Size: 9},
		{Field: "operation", X: 3, Y: 43, Width: 72, Height: 5, Size: 9},
	}}
}
func (s LabelSettings) Validate() error {
	bad := errors.New("Etikett prüfen: 78 × 51 mm, 1–20 Felder innerhalb des Etiketts, Schrift 6–32 pt, 1–8 Exemplare und gültige Druckerwarteschlange erforderlich.")
	if s.Version != 1 || s.Copies < 1 || s.Copies > 8 || len(s.Elements) < 1 || len(s.Elements) > 20 || (s.Queue != "" && !ValidPrinterQueue(s.Queue)) || (s.Enabled && s.Queue == "") {
		return bad
	}
	for _, e := range s.Elements {
		if _, ok := LabelFields[e.Field]; !ok {
			return bad
		}
		for _, n := range []float64{e.X, e.Y, e.Width, e.Height, e.Size} {
			if math.IsNaN(n) || math.IsInf(n, 0) {
				return bad
			}
		}
		if e.X < 0 || e.Y < 0 || e.Width < 2 || e.Height < 2 || e.X+e.Width > 78.001 || e.Y+e.Height > 51.001 || e.Size < 6 || e.Size > 32 || len([]rune(e.Text)) > 200 {
			return bad
		}
	}
	return nil
}
