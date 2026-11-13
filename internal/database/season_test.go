package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

func seasonClaim(t *testing.T, db *sql.DB, number int, kind, operation, stamp, source, status string) int64 {
	t.Helper()
	insert := func(query string, args ...any) int64 {
		r, err := db.Exec(query, args...)
		if err != nil {
			t.Fatal(err)
		}
		id, err := r.LastInsertId()
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	customer := insert("INSERT INTO customers(name) VALUES('Saisontest (fiktiv)')")
	vehicle := insert("INSERT INTO vehicles(customer_id,plate,car) VALUES(?,?,'Fiktives Fahrzeug')", customer, fmt.Sprint("MZ-ST ", number))
	tire := insert("INSERT INTO tire_sets(vehicle_id) VALUES(?)", vehicle)
	claimKind := "occupancy"
	if operation == "reservation" {
		claimKind = "reservation"
	}
	place := fmt.Sprintf("C1.%d", number)
	claim := insert("INSERT INTO storage_claims(place_id,tire_set_id,kind,status,updated_at) VALUES(?,?,?,?,?)", place, tire, claimKind, status, stamp)
	values := map[string]string{"tire_type": kind}
	if kind == "missing" {
		values = map[string]string{}
	}
	payload, _ := json.Marshal(values)
	insert("INSERT INTO history_events(claim_id,operation,occurred_at,place,name,plate,car,payload,source) VALUES(?,?,?,?,?,?,?,?,?)", claim, operation, stamp, place, "Saisontest (fiktiv)", fmt.Sprint("MZ-ST ", number), "Fiktives Fahrzeug", string(payload), source)
	return claim
}
func TestWinterSeasonProgressClassifiesCurrentStock(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	now, _ := time.Parse(time.RFC3339, "2026-11-20T12:00:00Z")
	done := seasonClaim(t, db, 1, "Sommer", "wheel_change", "2026-09-30T22:00:00Z", "recorded", "active")
	seasonClaim(t, db, 2, "Winter", "storage", "2026-04-01T12:00:00Z", "recorded", "active")
	seasonClaim(t, db, 3, "Sommer", "wheel_change", "2026-09-30T21:59:59Z", "recorded", "active")
	seasonClaim(t, db, 4, "Sommer", "storage", "2026-11-01T12:00:00Z", "recorded", "active")
	seasonClaim(t, db, 5, "Ganzjahr", "wheel_change", "2026-11-01T12:00:00Z", "recorded", "active")
	seasonClaim(t, db, 6, "Sommer", "wheel_change", "2026-11-21T12:00:00Z", "recorded", "active")
	seasonClaim(t, db, 7, "Sommer", "wheel_change", "2026-11-01T12:00:00Z", "legacy", "active")
	unknown := seasonClaim(t, db, 8, "missing", "storage", "2026-11-01T12:00:00Z", "legacy", "active")
	if _, err := db.Exec("INSERT INTO storage_details(claim_id,tire_type,rim_type,front_size,rear_size,front_depth,rear_depth,accessories,mechanic,notes) VALUES(?,'Winter','Stahl','{}','{}','6.0','6.0','{}','','')", unknown); err != nil {
		t.Fatal(err)
	}
	seasonClaim(t, db, 9, "Sommer", "wheel_change", "2026-11-01T12:00:00Z", "recorded", "released")
	seasonClaim(t, db, 10, "Winter", "reservation", "2026-11-01T12:00:00Z", "recorded", "active")
	seasonClaim(t, db, 11, "Winter", "wheel_change", "2026-11-01T12:00:00Z", "recorded", "active")
	seasonClaim(t, db, 12, "Sommer", "wheel_change", "2026-11-01T12:00:00Z", "recorded", "active")
	progress, err := CurrentSeasonProgress(ctx, db, now)
	if err != nil || progress.Completed != 2 || progress.Open != 2 || progress.Total != 4 || progress.Unknown != 5 || progress.AllSeason != 1 || progress.Active != 10 {
		t.Fatalf("%+v %v", progress, err)
	}
	if progress.Season.Label != "Winter 2026/27" {
		t.Fatal(progress.Season)
	}
	// Completion uses immutable history, not subsequently edited mutable details.
	var count int
	if err = db.QueryRow("SELECT count(*) FROM history_events").Scan(&count); err != nil || count != 12 {
		t.Fatal("read changed history", count, err)
	}
	if _, err = db.Exec("UPDATE storage_claims SET status='released' WHERE id=?", done); err != nil {
		t.Fatal(err)
	}
	progress, err = CurrentSeasonProgress(ctx, db, now)
	if err != nil || progress.Completed != 1 || progress.Total != 3 || progress.Active != 9 {
		t.Fatal(progress, err)
	}
}
func TestSummerSeasonProgressAndEmptyStock(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	now, _ := time.Parse(time.RFC3339, "2026-05-01T12:00:00Z")
	empty, err := CurrentSeasonProgress(ctx, db, now)
	if err != nil || empty.Total != 0 || empty.Active != 0 {
		t.Fatal(empty, err)
	}
	seasonClaim(t, db, 1, "Winter", "wheel_change", "2026-03-31T22:00:00Z", "recorded", "active")
	seasonClaim(t, db, 2, "Sommer", "storage", "2026-02-01T12:00:00Z", "recorded", "active")
	seasonClaim(t, db, 3, "Winter", "wheel_change", "2026-03-31T21:59:59Z", "recorded", "active")
	seasonClaim(t, db, 4, "Winter", "wheel_change", "bad-date", "recorded", "active")
	progress, err := CurrentSeasonProgress(ctx, db, now)
	if err != nil || progress.Completed != 1 || progress.Open != 1 || progress.Total != 2 || progress.Unknown != 2 || progress.Season.Kind != "summer" {
		t.Fatal(progress, err)
	}
}
