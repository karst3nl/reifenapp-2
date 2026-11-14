package database

import (
	"context"
	"testing"

	"github.com/karst3nl/reifenapp-2/internal/models"
)

func TestSearchDetailsPreserveHistoryAfterReleaseAndReuse(t *testing.T) {
	ctx := context.Background()
	db := testDB(t)
	input := storageExample()
	input.FrontDepth = "3.0"
	input.RearDepth = "4.0"
	input.Notes = "Fiktiver Hinweis: <script>kein HTML</script>\nSeitenwand prüfen."
	input.Mechanic = "Testmonteur (fiktiv)"
	options, err := LoadAccessoryOptions(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	options.Locations["hubcaps"] = append(options.Locations["hubcaps"], "Fiktive Zubehörbox")
	if err = SaveAccessoryOptions(ctx, db, options); err != nil {
		t.Fatal(err)
	}
	input.Hubcaps = models.Accessory{Present: true, Location: "Fiktive Zubehörbox"}
	input.Bolts = models.Accessory{Present: false}
	claim, err := CreateStorage(ctx, db, input)
	if err != nil {
		t.Fatal(err)
	}
	if err = ReleaseStorage(ctx, db, input.Area, input.Number, claim); err != nil {
		t.Fatal(err)
	}
	// The search must read event snapshots, even if mutable detail rows are changed later.
	if _, err = db.Exec("UPDATE storage_details SET notes='Spätere Änderung',front_depth='10.0' WHERE claim_id=?", claim); err != nil {
		t.Fatal(err)
	}
	replacement := storageExample()
	replacement.Name = "Neue Belegung (fiktiv)"
	replacement.Notes = "Neuer Hinweis"
	if _, err = CreateStorage(ctx, db, replacement); err != nil {
		t.Fatal(err)
	}
	rows, err := Search(ctx, db, input.Place())
	if err != nil || len(rows) != 3 {
		t.Fatal(rows, err)
	}
	if rows[0].Details.Notes != replacement.Notes {
		t.Fatal("new event mixed with old data")
	}
	for _, r := range rows[1:] {
		d := r.Details
		if d.Notes != input.Notes || d.FrontDepth != "3.0" || d.RearDepth != "4.0" || d.Front != input.Front || d.Rear != input.Rear || d.TireType != input.TireType || d.RimType != input.RimType || d.Brand != input.Brand || d.Model != input.Model || d.Mechanic != input.Mechanic || d.Accessories["hubcaps"] != input.Hubcaps || d.Accessories["bolts"].Present {
			t.Fatalf("snapshot lost fields: %+v", d)
		}
	}
	reservation := reservationInput()
	reservation.Number = 20
	if _, err = CreateReservation(ctx, db, reservation); err != nil {
		t.Fatal(err)
	}
	rows, err = Search(ctx, db, "C1.20")
	if err != nil || len(rows) != 1 {
		t.Fatal(rows, err)
	}
	d := rows[0].Details
	if d.FrontDepth != "" || d.RearDepth != "" || d.TireType != "" || d.Notes != "" || d.Brand != reservation.Brand {
		t.Fatal("reservation invented tire details", d)
	}
}
