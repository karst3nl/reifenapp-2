package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/karst3nl/reifenapp-2/migrations"
)

func TestHistoryLifecycleSnapshotsAndPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.sqlite")
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	reserved := reservationInput()
	reserved.Name = "Historie Fiktiv"
	reserved.Plate = "MZ-HI 123"
	reservationID, err := CreateReservation(ctx, db, reserved)
	if err != nil {
		t.Fatal(err)
	}
	storage := storageExample()
	storage.Name = reserved.Name
	storage.Plate = reserved.Plate
	storage.Brand = "Audi"
	storage.Model = "A3"
	claim, err := CreateWheelChange(ctx, db, storage)
	if err != nil {
		t.Fatal(err)
	}
	if err = ReleaseStorage(ctx, db, storage.Area, storage.Number, claim); err != nil {
		t.Fatal(err)
	}
	rows, err := Search(ctx, db, reserved.Plate)
	if err != nil || len(rows) != 3 {
		t.Fatalf("%v %v", rows, err)
	}
	for i, operation := range []string{"release", "wheel_change", "reservation"} {
		if rows[i].Operation != operation || rows[i].Active || rows[i].Source != "recorded" {
			t.Fatal(rows[i])
		}
	}
	if rows[2].Car != "VW Golf" || rows[1].Car != "Audi A3" {
		t.Fatal("snapshot changed", rows)
	}
	var payload string
	if err = db.QueryRow("SELECT payload FROM history_events WHERE claim_id=? AND operation='wheel_change'", claim).Scan(&payload); err != nil {
		t.Fatal(err)
	}
	var snapshot map[string]any
	if err = json.Unmarshal([]byte(payload), &snapshot); err != nil || snapshot["notes"] != storage.Notes || snapshot["front_depth"] != "6.5" {
		t.Fatal(payload, err)
	}
	var count int
	if err = db.QueryRow("SELECT count(*) FROM history_events WHERE claim_id=?", reservationID).Scan(&count); err != nil || count != 1 {
		t.Fatal(count, err)
	}
	for _, query := range []string{"UPDATE history_events SET name='changed'", "DELETE FROM history_events"} {
		if _, err = db.Exec(query); err == nil {
			t.Fatal("mutable history")
		}
	}
	checked, err := CheckPlace(ctx, db, storage.Area, storage.Number, "", "")
	if err != nil || !checked.Allowed {
		t.Fatal(checked, err)
	}
	storage.Name = "Andere Belegung Fiktiv"
	storage.Plate = "MZ-AB 124"
	if _, err = CreateStorage(ctx, db, storage); err != nil {
		t.Fatal(err)
	}
	rows, err = Search(ctx, db, "C1.2")
	if err != nil || len(rows) != 4 || !rows[0].Active {
		t.Fatal(rows, err)
	}
	db.Close()
	db, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, query := range []string{reserved.Name, reserved.Plate, "Auslagerung"} {
		rows, err = Search(ctx, db, query)
		if err != nil || len(rows) == 0 || rows[0].Operation != "release" {
			t.Fatal(rows, err)
		}
	}
}
func TestHistoryFailureRollsBackEveryOperation(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	storage := storageExample()
	claim, err := CreateStorage(ctx, db, storage)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`CREATE TRIGGER reject_history BEFORE INSERT ON history_events BEGIN SELECT RAISE(ABORT,'test failure'); END`); err != nil {
		t.Fatal(err)
	}
	if err = ReleaseStorage(ctx, db, storage.Area, storage.Number, claim); err == nil {
		t.Fatal("release without history")
	}
	target, err := LookupRelease(ctx, db, storage.Area, storage.Number)
	if err != nil || target.ClaimID != claim {
		t.Fatal(target, err)
	}
	storage.Number = 3
	if _, err = CreateStorage(ctx, db, storage); err == nil {
		t.Fatal("storage without history")
	}
	if _, err = CreateWheelChange(ctx, db, storage); err == nil {
		t.Fatal("wheel change without history")
	}
	reservation := reservationInput()
	reservation.Number = 3
	if _, err = CreateReservation(ctx, db, reservation); err == nil {
		t.Fatal("reservation without history")
	}
	for table, want := range map[string]int{"customers": 1, "storage_claims": 1, "storage_releases": 0, "history_events": 1} {
		var n int
		if err = db.QueryRow("SELECT count(*) FROM " + table).Scan(&n); err != nil || n != want {
			t.Fatal(table, n, err)
		}
	}
}
func TestHistoryBackfillFromVersionFour(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v4.sqlite")
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec("CREATE TABLE schema_migrations(version INTEGER PRIMARY KEY)"); err != nil {
		t.Fatal(err)
	}
	for i, name := range []string{"001_initial.sql", "002_storage_details.sql", "003_storage_operation.sql", "004_storage_releases.sql"} {
		script, err := migrations.Files.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = db.Exec(string(script)); err != nil {
			t.Fatal(err)
		}
		if _, err = db.Exec("INSERT INTO schema_migrations VALUES(?)", i+1); err != nil {
			t.Fatal(err)
		}
	}
	if err = SeedDevelopment(db); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`UPDATE storage_claims SET status='released' WHERE id=1;INSERT INTO storage_releases(claim_id,released_at) VALUES(1,'2026-11-10T14:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	db.Close()
	if opened, err := Open(path); err == nil {
		opened.Close()
		t.Fatal("implicit upgrade")
	}
	db, err = OpenWithMigration(path, true)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	rows, err := Search(context.Background(), db, "TESTMM123")
	if err != nil || len(rows) != 2 || rows[0].Operation != "release" || rows[0].Source != "legacy" {
		t.Fatal(rows, err)
	}
	checked, err := CheckPlace(context.Background(), db, "C1", 25, "", "")
	if err != nil || !checked.Allowed {
		t.Fatal(checked, err)
	}
	files, _ := filepath.Glob(path + ".backup-*")
	if len(files) != 1 {
		t.Fatal(files)
	}
	backup, err := sql.Open("sqlite3", files[0]+"?mode=ro")
	if err != nil {
		t.Fatal(err)
	}
	defer backup.Close()
	var version int
	if err = backup.QueryRow("SELECT MAX(version) FROM schema_migrations").Scan(&version); err != nil || version != 4 {
		t.Fatal(version, err)
	}
	db.Close()
	db, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var n int
	if err = db.QueryRow("SELECT count(*) FROM history_events").Scan(&n); err != nil || n != 3 {
		t.Fatal("duplicate backfill", n, err)
	}
}
