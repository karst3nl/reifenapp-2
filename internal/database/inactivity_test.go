package database

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestInactivePlacesCurrentOccupancyAndBoundary(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	now, _ := time.Parse(time.RFC3339, "2026-11-13T12:00:00Z")
	seasonClaim(t, db, 1, "Winter", "storage", "2025-11-13T12:00:00Z", "recorded", "active")
	seasonClaim(t, db, 2, "Winter", "wheel_change", "2025-11-13T12:00:01Z", "recorded", "active")
	seasonClaim(t, db, 3, "Winter", "storage", "2020-01-01T12:00:00Z", "recorded", "released")
	seasonClaim(t, db, 4, "Winter", "reservation", "2020-01-01T12:00:00Z", "recorded", "active")
	seasonClaim(t, db, 5, "Winter", "storage", "2020-01-01T12:00:00Z", "legacy", "active")
	seasonClaim(t, db, 6, "Winter", "storage", "bad-date", "recorded", "active")
	seasonClaim(t, db, 7, "Winter", "storage", "2027-01-01T12:00:00Z", "recorded", "active")
	updated := seasonClaim(t, db, 8, "Winter", "storage", "2020-01-01T12:00:00Z", "recorded", "active")
	if _, err := db.Exec("INSERT INTO history_events(claim_id,operation,occurred_at,place,name,plate,car,payload,source) VALUES(?,'wheel_change','2026-11-01T12:00:00Z','C1.8','Fiktiver Test','MZ-FT 123','Testauto','{}','recorded')", updated); err != nil {
		t.Fatal(err)
	}
	// An unrelated historical CSV event must not reset a live place's clock.
	if _, err := db.Exec("INSERT INTO history_events(claim_id,operation,occurred_at,place,name,plate,car,payload,source,import_key) VALUES(NULL,'wheel_change','2026-11-13T12:00:00Z','C1.1','Fiktiver Test','MZ-FT 123','Testauto','{}','csv','fictional-key')"); err != nil {
		t.Fatal(err)
	}
	report, err := InactivePlaces(ctx, db, now)
	if err != nil || report.Count != 1 || report.Unknown != 3 || report.Months != 12 || report.Items[0].Place != "C1.1" || report.Items[0].Days != 365 {
		t.Fatalf("report %+v %v", report, err)
	}
	if !report.Items[0].Active || report.Items[0].ReceiptURL == "" {
		t.Fatal("missing card data")
	}
	if err = SaveInactivitySettings(ctx, db, InactivitySettings{Months: 1}); err != nil {
		t.Fatal(err)
	}
	report, err = InactivePlaces(ctx, db, now)
	if err != nil || report.Count != 2 {
		t.Fatal(report, err)
	}
	// Releasing a place removes the warning immediately.
	if _, err = db.Exec("UPDATE storage_claims SET status='released' WHERE place_id='C1.1'"); err != nil {
		t.Fatal(err)
	}
	report, err = InactivePlaces(ctx, db, now)
	if err != nil || report.Count != 1 || report.Items[0].Place != "C1.2" {
		t.Fatal(report, err)
	}
	for _, months := range []int{0, -1, 121} {
		if err = SaveInactivitySettings(ctx, db, InactivitySettings{Months: months}); !errors.Is(err, ErrInactivityMonths) {
			t.Fatal("invalid threshold accepted", months, err)
		}
	}
	settings, err := ReadInactivitySettings(ctx, db)
	if err != nil || settings.Months != 1 {
		t.Fatal(settings, err)
	}
}
func TestInactivePlacesEmpty(t *testing.T) {
	report, err := InactivePlaces(context.Background(), testDB(t), time.Now())
	if err != nil || report.Count != 0 || report.Unknown != 0 || report.Items == nil {
		t.Fatal(report, err)
	}
}
