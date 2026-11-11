package receipts

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	_ "time/tzdata"

	"github.com/go-pdf/fpdf"
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

// Generate lays out all captured data on exactly one portrait DIN A4 page.
// Text is wrapped and the body font adjusted before rendering; nothing is clipped.
func Generate(r models.Receipt) ([]byte, error) {
	labels := map[string]string{"storage": "Neueinlagerung", "wheel_change": "Radwechsel", "reservation": "Reservierung", "release": "Auslagerung"}
	title, ok := labels[r.Operation]
	if !ok {
		return nil, fmt.Errorf("unknown operation")
	}
	rows, err := rowsFor(r)
	if err != nil {
		return nil, err
	}
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(14, 12, 14)
	pdf.SetAutoPageBreak(false, 12)
	pdf.AddUTF8FontFromBytes("Receipt", "", font)
	pdf.SetTitle("Reifenapp – "+title, true)
	pdf.SetAuthor("Reifenapp 2.0", false)
	// fixed page size: 210 x 297 mm; body area y=42 through y=274.
	fontSize, lineHeight := 10.5, 0.0
	heights := make([]float64, len(rows))
	for {
		pdf.SetFont("Receipt", "", fontSize)
		lineHeight = fontSize * 0.352778 * 1.22
		total := 0.0
		for i, r := range rows {
			lines := len(pdf.SplitText(r.value, 146))
			if lines < 1 {
				lines = 1
			}
			labels := len(pdf.SplitText(r.label, 32))
			if labels > lines {
				lines = labels
			}
			heights[i] = float64(lines)*lineHeight + 2
			total += heights[i]
		}
		if total <= 232 {
			break
		}
		fontSize -= 0.5
		if fontSize < 7 {
			return nil, fmt.Errorf("receipt exceeds one-page layout: %.1f mm at %.1f pt", total, fontSize+0.5)
		}
	}
	if err = pdf.Error(); err != nil {
		return nil, err
	}
	pdf.AddPage()
	pdf.SetTextColor(18, 35, 45)
	pdf.SetFont("Receipt", "", 20)
	pdf.SetXY(14, 13)
	pdf.CellFormat(182, 10, title, "", 1, "L", false, 0, "")
	pdf.SetFont("Receipt", "", 10)
	pdf.SetTextColor(70, 85, 95)
	pdf.CellFormat(182, 7, fmt.Sprintf("Reifenapp 2.0 · Vorgangsbeleg #%d", r.EventID), "", 1, "L", false, 0, "")
	pdf.SetDrawColor(70, 115, 100)
	pdf.Line(14, 35, 196, 35)
	y := 42.0
	for i, r := range rows {
		pdf.SetFont("Receipt", "", fontSize)
		pdf.SetTextColor(65, 80, 90)
		pdf.SetXY(14, y)
		pdf.MultiCell(32, lineHeight, r.label, "", "L", false)
		pdf.SetTextColor(15, 25, 30)
		pdf.SetXY(50, y)
		pdf.MultiCell(146, lineHeight, r.value, "", "L", false)
		y += heights[i]
		pdf.SetDrawColor(225, 230, 233)
		pdf.Line(14, y-1.5, 196, y-1.5)
	}
	pdf.SetFont("Receipt", "", 8)
	pdf.SetTextColor(90, 100, 110)
	pdf.SetXY(14, 282)
	pdf.CellFormat(182, 5, fmt.Sprintf("Historieneintrag #%d · Belegung #%d · Seite 1 von 1", r.EventID, r.ClaimID), "", 0, "L", false, 0, "")
	if pdf.PageCount() != 1 {
		return nil, fmt.Errorf("unexpected receipt page count")
	}
	var out bytes.Buffer
	if err = pdf.Output(&out); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
