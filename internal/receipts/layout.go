package receipts

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-pdf/fpdf"
	"github.com/karst3nl/reifenapp-2/internal/models"
)

//go:embed fonts/DejaVuSans-Bold.ttf
var boldFont []byte

type cell struct {
	label, value, align string
	fraction            float64
}
type block struct {
	cells  []cell
	inline bool
	gap    float64
}

func layoutBlocks(r models.Receipt, rows []row) []block {
	values := map[string]string{}
	for _, row := range rows {
		values[row.label] = row.value
	}
	dateLabels := map[string]string{"storage": "Datum der Einlagerung:", "wheel_change": "Datum des Radwechsels:", "reservation": "Datum der Reservierung:", "release": "Datum der Auslagerung:"}
	blocks := []block{}
	for _, meta := range []struct{ label, key string }{{dateLabels[r.Operation], "Zeitpunkt"}, {"Name:", "Kunde"}, {"Kennzeichen:", "Kennzeichen"}, {"Lagerplatz:", "Lagerplatz"}, {"Fahrzeug:", "Fahrzeug"}} {
		blocks = append(blocks, block{inline: true, cells: []cell{{meta.label, values[meta.key], "L", 1}}})
	}
	if r.Operation != "reservation" {
		var details struct {
			TireType string `json:"tire_type"`
			RimType  string `json:"rim_type"`
		}
		json.Unmarshal(r.Payload, &details)
		suffix := "eingelagert"
		if r.Operation == "release" {
			suffix = "ausgelagert"
		}
		blocks = append(blocks, block{gap: 9.5, cells: []cell{{"Reifenart " + suffix, present(details.TireType), "L", 1.0 / 3}, {"Felgenart " + suffix, present(details.RimType), "C", 1.0 / 3}, {"Felgenschloss", values["Felgenschloss"], "R", 1.0 / 3}}},
			block{cells: []cell{{"Radkappen", values["Radkappen"], "L", 1.0 / 3}, {"Profiltiefe Vorderachse", depthOnly(r.Payload, "front_depth"), "C", 1.0 / 3}, {"Profiltiefe Hinterachse", depthOnly(r.Payload, "rear_depth"), "R", 1.0 / 3}}},
			block{cells: []cell{{"Reifengröße Vorderachse", strings.Split(values["Vorderachse"], " · Profil:")[0], "L", 0.5}, {"Reifengröße Hinterachse", strings.Split(values["Hinterachse"], " · Profil:")[0], "R", 0.5}}},
			block{cells: []cell{{"Radschrauben", values["Radschrauben"], "L", 1.0 / 3}, {"Hinweise", values["Bemerkungen"], "C", 2.0 / 3}}},
			block{inline: true, cells: []cell{{"Monteur:", values["Monteur"], "L", 1}}})
	}
	for _, key := range []string{"Ergebnis", "Altbestand"} {
		if value, ok := values[key]; ok {
			blocks = append(blocks, block{inline: true, cells: []cell{{key + ":", value, "L", 1}}})
		}
	}
	return blocks
}
func depthOnly(payload []byte, key string) string {
	var values map[string]json.RawMessage
	json.Unmarshal(payload, &values)
	var value string
	json.Unmarshal(values[key], &value)
	if value == "" {
		return "Nicht erfasst"
	}
	return strings.ReplaceAll(value, ".", ",") + " mm"
}
func lineCount(pdf *fpdf.Fpdf, value string, width float64) int {
	n := len(pdf.SplitText(value, width))
	if n < 1 {
		return 1
	}
	return n
}
func blockHeight(pdf *fpdf.Fpdf, b block, width, size float64) float64 {
	lh := size * 0.352778 * 1.2
	if b.inline {
		pdf.SetFont("Receipt", "", size)
		c := b.cells[0]
		n := lineCount(pdf, c.label, width/2-2)
		if v := lineCount(pdf, c.value, width/2-2); v > n {
			n = v
		}
		return float64(n)*lh + 3 + b.gap
	}
	labels, values := 0, 0
	for _, c := range b.cells {
		w := width*c.fraction - 3
		pdf.SetFont("Receipt", "B", size*0.89)
		if n := lineCount(pdf, c.label, w); n > labels {
			labels = n
		}
		pdf.SetFont("Receipt", "", size*0.89)
		if n := lineCount(pdf, c.value, w); n > values {
			values = n
		}
	}
	return float64(labels+values)*lh*0.89 + 3 + b.gap
}

func renderBlock(pdf *fpdf.Fpdf, b block, x, y, width, size float64) float64 {
	height := blockHeight(pdf, b, width, size)
	y += b.gap
	lh := size * 0.352778 * 1.2
	offset := 0.0
	commonLabelHeight := 0.0
	if !b.inline {
		pdf.SetFont("Receipt", "B", size*0.89)
		for _, c := range b.cells {
			h := float64(lineCount(pdf, c.label, width*c.fraction-3)) * lh * 0.89
			if h > commonLabelHeight {
				commonLabelHeight = h
			}
		}
	}
	for _, c := range b.cells {
		if b.inline {
			pdf.SetFont("Receipt", "", size)
			pdf.SetXY(x, y)
			pdf.MultiCell(width/2-2, lh, c.label, "", "L", false)
			pdf.SetXY(x+width/2+1, y)
			pdf.MultiCell(width/2-1, lh, c.value, "", "L", false)
		} else {
			w := width*c.fraction - 3
			align := c.align
			pdf.SetFont("Receipt", "B", size*0.89)
			labelHeight := commonLabelHeight
			pdf.SetXY(x+offset, y)
			pdf.MultiCell(w, lh*0.89, c.label, "", align, false)
			pdf.SetFont("Receipt", "", size*0.89)
			pdf.SetXY(x+offset, y+labelHeight)
			pdf.MultiCell(w, lh*0.89, c.value, "", align, false)
			offset += width * c.fraction
		}
	}
	end := y - b.gap + height
	pdf.SetDrawColor(130, 130, 130)
	pdf.Line(x, end-1.5, x+width, end-1.5)
	return end
}
func renderHeader(pdf *fpdf.Fpdf, h Header) float64 {
	if text(h.Company) == "" {
		h.Company = "Reifenapp 2.0"
	}
	y := 15.0
	for i, value := range []string{h.Company, h.Street, h.City, h.Phone} {
		value = text(value)
		if value == "" {
			continue
		}
		size, style := 13.5, ""
		if i == 0 {
			size = 18.75
			style = "B"
		}
		if i == 3 {
			style = "B"
			value = "Telefon: " + value
		}
		pdf.SetFont("Receipt", style, size)
		lh := size * 0.352778 * 1.18
		pdf.SetXY(25, y)
		pdf.MultiCell(160, lh, value, "", "L", false)
		y += float64(lineCount(pdf, value, 160))*lh + 1
	}
	return y
}

// Generate preserves the automatic one-page PDF workflow with a neutral header.
func Generate(r models.Receipt) ([]byte, error) { return GenerateWithHeader(r, Header{}) }

// GenerateWithHeader follows the supplied legacy print layout for normal input.
// Unusually long input switches to a compact table rather than clipping data.
func GenerateWithHeader(r models.Receipt, h Header) ([]byte, error) {
	if err := h.validate(); err != nil {
		return nil, err
	}
	titles := map[string]string{"storage": "Neueinlagerung", "wheel_change": "Radwechsel", "reservation": "Reservierung", "release": "Auslagerung"}
	title, ok := titles[r.Operation]
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
	pdf.AddUTF8FontFromBytes("Receipt", "B", boldFont)
	pdf.SetTitle("Reifenapp – "+title, true)
	pdf.SetAuthor("Reifenapp 2.0", false)
	pdf.AddPage()
	pdf.SetTextColor(0, 0, 0)
	headerEnd := renderHeader(pdf, h)
	pdf.SetFont("Receipt", "", 9)
	pdf.SetXY(25, headerEnd+3)
	pdf.CellFormat(160, 5, fmt.Sprintf("%s · Vorgangsbeleg #%d", title, r.EventID), "", 0, "L", false, 0, "")
	start := headerEnd + 21.2
	blocks := layoutBlocks(r, rows)
	size := 13.5
	fit := false
	for size >= 9 {
		total := start
		for _, b := range blocks {
			total += blockHeight(pdf, b, 160, size)
		}
		if total <= 275 {
			fit = true
			break
		}
		size -= 0.5
	}
	if fit {
		y := start
		for _, b := range blocks {
			y = renderBlock(pdf, b, 25, y, 160, size)
		}
	} else {
		// Keep every field, including all notes/accessory locations, on one A4 page.
		y := headerEnd + 11
		size = 10.5
		heights := make([]float64, len(rows))
		lh := 0.0
		for {
			pdf.SetFont("Receipt", "", size)
			lh = size * 0.352778 * 1.18
			total := y
			for i, row := range rows {
				n := lineCount(pdf, row.value, 146)
				if label := lineCount(pdf, row.label, 32); label > n {
					n = label
				}
				heights[i] = float64(n)*lh + 1.5
				total += heights[i]
			}
			if total <= 275 {
				break
			}
			size -= 0.5
			if size < 6 {
				return nil, fmt.Errorf("receipt exceeds single-page layout")
			}
		}
		for i, row := range rows {
			pdf.SetFont("Receipt", "", size)
			pdf.SetXY(14, y)
			pdf.MultiCell(32, lh, row.label, "", "L", false)
			pdf.SetXY(50, y)
			pdf.MultiCell(146, lh, row.value, "", "L", false)
			y += heights[i]
			pdf.SetDrawColor(150, 150, 150)
			pdf.Line(14, y-0.75, 196, y-0.75)
		}
	}
	pdf.SetFont("Receipt", "", 8)
	pdf.SetTextColor(90, 90, 90)
	pdf.SetXY(25, 283)
	pdf.CellFormat(160, 4, fmt.Sprintf("Historieneintrag #%d · Belegung #%d · Seite 1 von 1", r.EventID, r.ClaimID), "", 0, "L", false, 0, "")
	if pdf.PageCount() != 1 {
		return nil, fmt.Errorf("unexpected page count")
	}
	var out bytes.Buffer
	if err = pdf.Output(&out); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
