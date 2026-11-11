package receipts

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/karst3nl/reifenapp-2/internal/models"
)

func receiptExample(operation string) models.Receipt {
	payload, _ := json.Marshal(map[string]any{"tire_type": "Winter", "rim_type": "Aluminium", "front": models.TireSize{Width: "205", Ratio: "55", Inches: "16", Load: "91", Speed: "H"}, "rear": models.TireSize{Width: "225", Ratio: "45", Inches: "17", Load: "94", Speed: "V"}, "front_depth": "6.5", "rear_depth": "7.0", "accessories": map[string]models.Accessory{"hubcaps": {Present: true, Location: "Am Radsatz"}, "bolts": {Present: false}, "lock": {Present: true, Location: "Kofferraum"}}, "mechanic": "Jörg Fiktiv", "notes": "Prüfung: Größe und Zubehör vollständig. Ende."})
	return models.Receipt{EventID: 42, ClaimID: 9, Operation: operation, Date: "2026-11-11T12:30:00Z", Name: "Müller Škoda Fiktiv", Plate: "MZ-AB 123", Car: "Škoda Octavia", Place: "C1.9", Source: "recorded", Payload: payload}
}
func TestSingleA4Receipts(t *testing.T) {
	for _, op := range []string{"storage", "wheel_change", "reservation", "release"} {
		t.Run(op, func(t *testing.T) {
			t.Parallel()
			data, err := Generate(receiptExample(op))
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.HasPrefix(data, []byte("%PDF-")) || !bytes.Contains(data, []byte("/Count 1")) || !bytes.Contains(data, []byte("/MediaBox [0 0 595.28 841.89]")) || !bytes.Contains(data, []byte("/FontFile2")) {
				t.Fatalf("invalid one-page A4 PDF (%d bytes)", len(data))
			}
		})
	}
}
func TestMaximumTextFitsWithoutTruncation(t *testing.T) {
	r := receiptExample("release")
	r.Name = strings.Repeat("W", 200)
	r.Car = strings.Repeat("W", 120)
	var payload map[string]any
	json.Unmarshal(r.Payload, &payload)
	payload["notes"] = strings.Repeat("W", 1990) + " ENDMARKER"
	payload["mechanic"] = strings.Repeat("W", 100)
	accessories := map[string]models.Accessory{}
	for _, key := range []string{"hubcaps", "bolts", "lock"} {
		accessories[key] = models.Accessory{Present: true, Location: strings.Repeat("W", 200)}
	}
	payload["accessories"] = accessories
	r.Payload, _ = json.Marshal(payload)
	rows, err := rowsFor(r)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, row := range rows {
		if row.label == "Bemerkungen" {
			found = strings.HasSuffix(row.value, "ENDMARKER")
		}
	}
	if !found {
		t.Fatal("notes truncated")
	}
	data, err := Generate(r)
	if err != nil || !bytes.Contains(data, []byte("/Count 1")) {
		t.Fatal(err)
	}
}
func TestReceiptOmittedFieldsAndInvalidPayload(t *testing.T) {
	r := receiptExample("reservation")
	rows, err := rowsFor(r)
	if err != nil || len(rows) != 5 {
		t.Fatal(rows, err)
	}
	r.Operation = "release"
	r.Source = "legacy"
	r.Payload = []byte(`{}`)
	rows, err = rowsFor(r)
	if err != nil {
		t.Fatal(err)
	}
	if rows[len(rows)-1].label != "Altbestand" {
		t.Fatal(rows)
	}
	r.Payload = []byte(`{`)
	if _, err = Generate(r); err == nil {
		t.Fatal("invalid JSON accepted")
	}
	r = receiptExample("unknown")
	if _, err = Generate(r); err == nil {
		t.Fatal("invalid operation accepted")
	}
}
