package receipts

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strings"

	"github.com/go-pdf/fpdf"
	"github.com/karst3nl/reifenapp-2/internal/models"
)

type Element struct {
	ID         string  `json:"id"`
	Image      string  `json:"image,omitempty"`
	Type       string  `json:"type"`
	Text       string  `json:"text"`
	Field      string  `json:"field"`
	X          float64 `json:"x"`
	Y          float64 `json:"y"`
	Width      float64 `json:"width"`
	Height     float64 `json:"height"`
	FontSize   float64 `json:"font_size"`
	Bold       bool    `json:"bold"`
	Align      string  `json:"align"`
	Color      string  `json:"color"`
	Background string  `json:"background"`
	Stroke     float64 `json:"stroke"`
}
type Template struct {
	Version  int       `json:"version"`
	Elements []Element `json:"elements"`
}
type Field struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

var Fields = []Field{{"operation", "Vorgangsart"}, {"date", "Zeitpunkt"}, {"place", "Lagerplatz"}, {"name", "Kundenname"}, {"plate", "Kennzeichen"}, {"car", "Fahrzeug"}, {"tire_type", "Reifenart"}, {"rim_type", "Felgenart"}, {"front_size", "Reifengröße VA"}, {"rear_size", "Reifengröße HA"}, {"front_depth", "Profiltiefe VA"}, {"rear_depth", "Profiltiefe HA"}, {"hubcaps", "Radkappen"}, {"bolts", "Radschrauben"}, {"lock", "Felgenschloss"}, {"mechanic", "Monteur"}, {"notes", "Bemerkungen"}, {"event_id", "Vorgangsnummer"}, {"claim_id", "Belegungsnummer"}, {"company", "Firma"}, {"street", "Straße"}, {"city", "PLZ / Ort"}, {"phone", "Telefon"}, {"source", "Altbestandshinweis"}}

func ValidOperation(s string) bool {
	return s == "storage" || s == "wheel_change" || s == "reservation" || s == "release"
}

var colorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

func (t Template) Validate() error {
	if t.Version != 1 || len(t.Elements) < 1 || len(t.Elements) > 100 {
		return fmt.Errorf("Vorlage benötigt 1 bis 100 Elemente und Version 1.")
	}
	fields := map[string]bool{}
	for _, f := range Fields {
		fields[f.Key] = true
	}
	seen := map[string]bool{}
	logoBytes := 0
	for _, e := range t.Elements {
		if e.ID == "" || len(e.ID) > 64 || seen[e.ID] {
			return fmt.Errorf("Elementkennungen müssen eindeutig sein.")
		}
		seen[e.ID] = true
		if e.Type == "image" {
			data, _, err := decodeLogo(e.Image)
			if err != nil {
				return err
			}
			logoBytes += len(data)
			if logoBytes > MaxTemplateLogoBytes {
				return fmt.Errorf("Logos einer Vorlage dürfen zusammen höchstens 4 MiB belegen.")
			}
		} else if e.Image != "" {
			return fmt.Errorf("Bilddaten sind nur für Logo-Elemente erlaubt.")
		}

		if e.Type != "text" && e.Type != "field" && e.Type != "line" && e.Type != "box" && e.Type != "image" {
			return fmt.Errorf("Unbekannter Elementtyp.")
		}
		for _, n := range []float64{e.X, e.Y, e.Width, e.Height, e.FontSize, e.Stroke} {
			if math.IsNaN(n) || math.IsInf(n, 0) {
				return fmt.Errorf("Ungültige Abmessung.")
			}
		}
		if e.X < 0 || e.Y < 0 || e.Width < 1 || e.Height < 1 || e.X+e.Width > 210.001 || e.Y+e.Height > 297.001 {
			return fmt.Errorf("Alle Elemente müssen innerhalb der DIN-A4-Seite liegen.")
		}
		if len([]rune(e.Text)) > 2000 {
			return fmt.Errorf("Elementtext ist zu lang.")
		}
		if e.Type == "field" && !fields[e.Field] {
			return fmt.Errorf("Unbekanntes Datenfeld.")
		}
		if e.Type == "text" || e.Type == "field" {
			if e.FontSize < 6 || e.FontSize > 48 {
				return fmt.Errorf("Schriftgröße muss 6 bis 48 pt betragen.")
			}
			if e.Align != "L" && e.Align != "C" && e.Align != "R" {
				return fmt.Errorf("Ungültige Textausrichtung.")
			}
		}
		if !colorPattern.MatchString(e.Color) || !colorPattern.MatchString(e.Background) || e.Stroke < 0 || e.Stroke > 3 {
			return fmt.Errorf("Ungültige Farbe oder Linienstärke.")
		}
	}
	return nil
}
func FieldValues(r models.Receipt, h Header) (map[string]string, error) {
	rows, err := rowsFor(r)
	if err != nil {
		return nil, err
	}
	v := map[string]string{}
	for _, row := range rows {
		v[row.label] = row.value
	}
	var details struct {
		TireType string `json:"tire_type"`
		RimType  string `json:"rim_type"`
	}
	json.Unmarshal(r.Payload, &details)
	company := h.Company
	if company == "" {
		company = "Reifenapp 2.0"
	}
	result := map[string]string{"operation": map[string]string{"storage": "Neueinlagerung", "wheel_change": "Radwechsel", "reservation": "Reservierung", "release": "Auslagerung"}[r.Operation], "date": v["Zeitpunkt"], "place": r.Place, "name": r.Name, "plate": r.Plate, "car": r.Car, "tire_type": present(details.TireType), "rim_type": present(details.RimType), "front_size": strings.Split(v["Vorderachse"], " · Profil:")[0], "rear_size": strings.Split(v["Hinterachse"], " · Profil:")[0], "front_depth": depthOnly(r.Payload, "front_depth"), "rear_depth": depthOnly(r.Payload, "rear_depth"), "hubcaps": v["Radkappen"], "bolts": v["Radschrauben"], "lock": v["Felgenschloss"], "mechanic": v["Monteur"], "notes": v["Bemerkungen"], "event_id": fmt.Sprint(r.EventID), "claim_id": fmt.Sprint(r.ClaimID), "company": company, "street": h.Street, "city": h.City, "phone": h.Phone, "source": v["Altbestand"]}
	return result, nil
}
func ExampleReceipt(operation string) models.Receipt {
	payload := `{"tire_type":"Winter","rim_type":"Aluminium","front":{"width":"205","ratio":"55","inches":"16","load":"91","speed":"H"},"rear":{"width":"225","ratio":"45","inches":"17","load":"94","speed":"V"},"front_depth":"6.5","rear_depth":"7.0","accessories":{"hubcaps":{"present":true,"location":"Am Radsatz"},"bolts":{"present":true,"location":"Beutel am Radsatz"},"lock":{"present":false,"location":""}},"mechanic":"Testmonteur (fiktiv)","notes":"Ausschließlich fiktive Vorschaudaten."}`
	if operation == "reservation" {
		payload = `{}`
	}
	return models.Receipt{EventID: 42, ClaimID: 9, Operation: operation, Date: "2026-11-12T09:30:00Z", Place: "C1.9", Name: "Max Mustermann (fiktiv)", Plate: "MZ-MM 123", Car: "VW Golf", Source: "recorded", Payload: []byte(payload)}
}
func DefaultTemplate(operation string) Template {
	t := Template{Version: 1}
	n := 0
	add := func(typ, field, value string, x, y, w, h, size float64, bold bool) {
		n++
		t.Elements = append(t.Elements, Element{ID: fmt.Sprintf("element-%d", n), Type: typ, Field: field, Text: value, X: x, Y: y, Width: w, Height: h, FontSize: size, Bold: bold, Align: "L", Color: "#000000", Background: "#ffffff", Stroke: 0})
	}
	for _, e := range []struct {
		field      string
		y, h, size float64
		bold       bool
	}{{"company", 15, 10, 19, true}, {"street", 27, 7, 12, false}, {"city", 35, 7, 12, false}, {"phone", 43, 7, 12, true}, {"operation", 54, 7, 11, true}} {
		add("field", e.field, "", 25, e.y, 160, e.h, e.size, e.bold)
	}
	for i, p := range []struct{ label, field string }{{"Datum:", "date"}, {"Name:", "name"}, {"Kennzeichen:", "plate"}, {"Lagerplatz:", "place"}, {"Fahrzeug:", "car"}} {
		y := 68 + float64(i)*12
		add("text", "", p.label, 25, y, 75, 9, 12, false)
		add("field", p.field, "", 105, y, 80, 9, 12, false)
		add("line", "", "", 25, y+10, 160, 1, 6, false)
		t.Elements[len(t.Elements)-1].Stroke = 0.2
	}
	if operation != "reservation" {
		for _, e := range []struct {
			label, field string
			x, y, w, h   float64
		}{{"Reifenart", "tire_type", 25, 137, 49, 20}, {"Felgenart", "rim_type", 80, 137, 49, 20}, {"Felgenschloss", "lock", 135, 137, 50, 24}, {"Radkappen", "hubcaps", 25, 164, 49, 24}, {"Profiltiefe VA", "front_depth", 80, 164, 49, 20}, {"Profiltiefe HA", "rear_depth", 135, 164, 50, 20}, {"Reifengröße VA", "front_size", 25, 191, 77, 23}, {"Reifengröße HA", "rear_size", 108, 191, 77, 23}, {"Radschrauben", "bolts", 25, 217, 49, 28}, {"Hinweise", "notes", 80, 217, 105, 28}, {"Monteur", "mechanic", 25, 248, 160, 15}} {
			add("text", "", e.label, e.x, e.y, e.w, 6, 10, true)
			add("field", e.field, "", e.x, e.y+7, e.w, e.h-7, 10, false)
		}
	}
	add("field", "event_id", "Historieneintrag #", 25, 280, 160, 5, 8, false)
	return t
}
func rgb(s string) (int, int, int) {
	var r, g, b int
	fmt.Sscanf(s, "#%02x%02x%02x", &r, &g, &b)
	return r, g, b
}

// GenerateTemplate uses the same millimeter coordinates as the editor. Preview
// rejects overflowing fields; production can use the lossless standard fallback.
func GenerateTemplate(r models.Receipt, h Header, t Template) ([]byte, error) {
	if err := t.Validate(); err != nil {
		return nil, err
	}
	values, err := FieldValues(r, h)
	if err != nil {
		return nil, err
	}
	p := fpdf.New("P", "mm", "A4", "")
	p.SetAutoPageBreak(false, 0)
	p.SetMargins(0, 0, 0)
	p.AddUTF8FontFromBytes("Receipt", "", font)
	p.AddUTF8FontFromBytes("Receipt", "B", boldFont)
	p.AddPage()
	for _, e := range t.Elements {
		cr, cg, cb := rgb(e.Color)
		p.SetTextColor(cr, cg, cb)
		p.SetDrawColor(cr, cg, cb)
		p.SetLineWidth(e.Stroke)
		if e.Type == "line" {
			if e.Stroke > 0 {
				p.Line(e.X, e.Y, e.X+e.Width, e.Y)
			}
			continue
		}
		if e.Type == "box" {
			br, bg, bb := rgb(e.Background)
			p.SetFillColor(br, bg, bb)
			style := "F"
			if e.Stroke > 0 {
				style = "DF"
			}
			p.Rect(e.X, e.Y, e.Width, e.Height, style)
			continue
		}
		if e.Type == "image" {
			data, cfg, err := decodeLogo(e.Image)
			if err != nil {
				return nil, err
			}
			name := fmt.Sprintf("logo-%x", sha256.Sum256(data))
			options := fpdf.ImageOptions{ImageType: "PNG", ReadDpi: false}
			p.RegisterImageOptionsReader(name, options, bytes.NewReader(data))
			scale := math.Min(e.Width/float64(cfg.Width), e.Height/float64(cfg.Height))
			width, height := float64(cfg.Width)*scale, float64(cfg.Height)*scale
			p.ImageOptions(name, e.X+(e.Width-width)/2, e.Y+(e.Height-height)/2, width, height, false, options, 0, "")
			continue
		}
		value := e.Text
		if e.Type == "field" {
			value += values[e.Field]
		}
		value = text(value)
		if value == "" {
			continue
		}
		size := e.FontSize
		style := ""
		if e.Bold {
			style = "B"
		}
		for {
			p.SetFont("Receipt", style, size)
			lines := lineCount(p, value, e.Width)
			if float64(lines)*size*0.352778*1.18 <= e.Height+0.01 {
				break
			}
			size -= 0.5
			if size < 6 {
				return nil, fmt.Errorf("Textfeld %s ist zu klein für den Inhalt.", e.ID)
			}
		}
		p.SetXY(e.X, e.Y)
		p.MultiCell(e.Width, size*0.352778*1.18, value, "", e.Align, false)
	}
	if p.PageCount() != 1 {
		return nil, fmt.Errorf("Vorlage erzeugt mehr als eine Seite.")
	}
	var out bytes.Buffer
	err = p.Output(&out)
	return out.Bytes(), err
}
func GenerateWithFallback(r models.Receipt, h Header, t Template) ([]byte, bool, error) {
	data, err := GenerateTemplate(r, h, t)
	if err == nil {
		return data, false, nil
	}
	data, err = generateStandard(r, h, "Eigenes Layout zu klein oder ungültig: vollständiger Standardbeleg verwendet.")
	return data, true, err
}

func FontBytes() []byte     { return font }
func BoldFontBytes() []byte { return boldFont }
