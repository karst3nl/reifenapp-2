package receipts

import (
	_ "embed"
	"encoding/json"
	"strings"
	"time"
	_ "time/tzdata"

	"github.com/karst3nl/reifenapp-2/internal/models"
)

//go:embed fonts/DejaVuSans.ttf
var font []byte

type row struct{ label, value string }

func text(value string) string { return strings.Join(strings.Fields(value), " ") }
func present(value string) string {
	value = text(value)
	if value == "" {
		return "Nicht erfasst"
	}
	return value
}
func rowsFor(r models.Receipt) ([]row, error) {
	var details struct {
		Brand       string                      `json:"brand"`
		Model       string                      `json:"model"`
		TireType    string                      `json:"tire_type"`
		RimType     string                      `json:"rim_type"`
		Front       models.TireSize             `json:"front"`
		Rear        models.TireSize             `json:"rear"`
		FrontDepth  string                      `json:"front_depth"`
		RearDepth   string                      `json:"rear_depth"`
		Accessories map[string]models.Accessory `json:"accessories"`
		Mechanic    string                      `json:"mechanic"`
		Notes       string                      `json:"notes"`
	}
	if err := json.Unmarshal(r.Payload, &details); err != nil {
		return nil, err
	}
	stamp := r.Date
	if parsed, err := time.Parse(time.RFC3339Nano, r.Date); err == nil {
		location, err := time.LoadLocation("Europe/Berlin")
		if err != nil {
			return nil, err
		}
		stamp = parsed.In(location).Format("02.01.2006 15:04:05 MST")
	}
	rows := []row{{"Zeitpunkt", present(stamp)}, {"Lagerplatz", present(r.Place)}, {"Kunde", present(r.Name)}, {"Kennzeichen", present(r.Plate)}, {"Fahrzeug", present(r.Car)}}
	if r.Operation != "reservation" {
		size := func(s models.TireSize, depth string) string {
			profile := present(depth)
			if depth != "" {
				profile = strings.ReplaceAll(profile, ".", ",") + " mm"
			}
			if s.Width == "" && s.Ratio == "" && s.Inches == "" && s.Load == "" && s.Speed == "" {
				return "Reifengröße nicht erfasst · Profil: " + profile
			}
			return present(s.Width) + "/" + present(s.Ratio) + " R" + present(s.Inches) + " · Traglast: " + present(s.Load) + " · Index: " + present(s.Speed) + " · Profil: " + profile
		}
		rows = append(rows, row{"Reifen / Felgen", present(details.TireType) + " / " + present(details.RimType)}, row{"Vorderachse", size(details.Front, details.FrontDepth)}, row{"Hinterachse", size(details.Rear, details.RearDepth)})
		for _, a := range []struct{ key, label string }{{"hubcaps", "Radkappen"}, {"bolts", "Radschrauben"}, {"lock", "Felgenschloss"}} {
			value := "Nicht erfasst"
			if accessory, ok := details.Accessories[a.key]; ok {
				if accessory.Present {
					value = "Vorhanden · Aufbewahrung: " + present(accessory.Location)
				} else {
					value = "Nicht vorhanden"
				}
			}
			rows = append(rows, row{a.label, value})
		}
		rows = append(rows, row{"Monteur", present(details.Mechanic)}, row{"Bemerkungen", present(details.Notes)})
	}
	if r.Operation == "release" {
		rows = append(rows, row{"Ergebnis", "Radsatz ausgelagert; die Belegung wurde beendet."})
	}
	if r.Source == "legacy" {
		rows = append(rows, row{"Altbestand", "Beleg aus den noch vorhandenen gespeicherten Angaben rekonstruiert."})
	}
	return rows, nil
}
