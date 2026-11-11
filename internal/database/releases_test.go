package database

import (
	"context"
	"database/sql"
	"errors"
	"github.com/karst3nl/reifenapp-2/migrations"
	"path/filepath"
	"sync"
	"testing"
)

func TestReleaseFreesPlacePreservesData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.sqlite")
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	input := storageExample()
	id, err := CreateStorage(context.Background(), db, input)
	if err != nil {
		t.Fatal(err)
	}
	var original string
	db.QueryRow("SELECT updated_at FROM storage_claims WHERE id=?", id).Scan(&original)
	target, err := LookupRelease(context.Background(), db, input.Area, input.Number)
	if err != nil || target.ClaimID != id || target.Name != input.Name || target.Plate != input.Plate {
		t.Fatalf("%+v %v", target, err)
	}
	if err = ReleaseStorage(context.Background(), db, input.Area, input.Number, id); err != nil {
		t.Fatal(err)
	}
	var status, stamp, released string
	if err = db.QueryRow(`SELECT s.status,s.updated_at,r.released_at FROM storage_claims s JOIN storage_releases r ON r.claim_id=s.id WHERE s.id=?`, id).Scan(&status, &stamp, &released); err != nil || status != "released" || stamp != original || released == "" {
		t.Fatalf("%s %s %s %v", status, stamp, released, err)
	}
	checked, err := CheckPlace(context.Background(), db, input.Area, input.Number, input.Name, input.Plate)
	if err != nil || !checked.Allowed || checked.Status != "free" {
		t.Fatalf("%+v %v", checked, err)
	}
	rows, err := Search(context.Background(), db, input.Name)
	if err != nil || len(rows) != 2 || rows[0].Operation != "release" || rows[0].Active || rows[1].Active {
		t.Fatalf("%v %v", rows, err)
	}
	for _, table := range []string{"customers", "vehicles", "tire_sets", "storage_details"} {
		var n int
		db.QueryRow("SELECT count(*) FROM " + table).Scan(&n)
		if n != 1 {
			t.Fatal("lost data", table, n)
		}
	}
	db.Close()
	db, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	target, err = LookupRelease(context.Background(), db, input.Area, input.Number)
	if err != nil || target.Status != "free" {
		t.Fatal(target, err)
	}
	newer, err := CreateStorage(context.Background(), db, input)
	if err != nil {
		t.Fatal(err)
	}
	if err = ReleaseStorage(context.Background(), db, input.Area, input.Number, id); !errors.Is(err, ErrReleaseChanged) {
		t.Fatal("stale confirmation accepted", err)
	}
	target, err = LookupRelease(context.Background(), db, input.Area, input.Number)
	if err != nil || target.ClaimID != newer {
		t.Fatal("newer occupancy released", target, err)
	}
}
func TestReleaseRejectsReservationAndRollsBack(t *testing.T) {
	db := testDB(t)
	input := reserveExample(t, db)
	target, err := LookupRelease(context.Background(), db, input.Area, input.Number)
	if err != nil || target.Status != "reserved" || target.ClaimID != 0 {
		t.Fatal(target, err)
	}
	if err = ReleaseStorage(context.Background(), db, input.Area, input.Number, 10); !errors.Is(err, ErrReleaseChanged) {
		t.Fatal(err)
	}
	if _, err = LookupRelease(context.Background(), db, "C2", 2); !errors.Is(err, ErrInvalidPlace) {
		t.Fatal(err)
	}
	id, err := CreateWheelChange(context.Background(), db, input)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`CREATE TRIGGER reject_release BEFORE INSERT ON storage_releases BEGIN SELECT RAISE(ABORT,'test failure'); END`); err != nil {
		t.Fatal(err)
	}
	if err = ReleaseStorage(context.Background(), db, input.Area, input.Number, id); err == nil {
		t.Fatal("expected failure")
	}
	target, err = LookupRelease(context.Background(), db, input.Area, input.Number)
	if err != nil || target.ClaimID != id {
		t.Fatal("partial release", target, err)
	}
}
func TestConcurrentRelease(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.sqlite")
	a, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	b, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	input := storageExample()
	id, err := CreateStorage(context.Background(), a, input)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, db := range []*sql.DB{a, b} {
		wg.Add(1)
		go func(db *sql.DB) {
			defer wg.Done()
			results <- ReleaseStorage(context.Background(), db, input.Area, input.Number, id)
		}(db)
	}
	wg.Wait()
	close(results)
	success, conflict := 0, 0
	for err := range results {
		if err == nil {
			success++
		} else if errors.Is(err, ErrReleaseChanged) {
			conflict++
		} else {
			t.Fatal(err)
		}
	}
	if success != 1 || conflict != 1 {
		t.Fatal(success, conflict)
	}
}

func TestReleaseUpgradeFromVersionThree(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v3.sqlite")
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec("CREATE TABLE schema_migrations(version INTEGER PRIMARY KEY)"); err != nil {
		t.Fatal(err)
	}
	for i, name := range []string{"001_initial.sql", "002_storage_details.sql", "003_storage_operation.sql"} {
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
	db.Close()
	if opened, err := Open(path); err == nil {
		opened.Close()
		t.Fatal("upgrade without flag")
	}
	db, err = OpenWithMigration(path, true)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	files, err := filepath.Glob(path + ".backup-*")
	if err != nil || len(files) != 1 {
		t.Fatal(files, err)
	}
	backup, err := sql.Open("sqlite3", files[0]+"?mode=ro")
	if err != nil {
		t.Fatal(err)
	}
	defer backup.Close()
	var version int
	if err = backup.QueryRow("SELECT MAX(version) FROM schema_migrations").Scan(&version); err != nil || version != 3 {
		t.Fatal(version, err)
	}
	var n int
	if err = db.QueryRow("SELECT count(*) FROM storage_claims WHERE status='active'").Scan(&n); err != nil || n != 2 {
		t.Fatal(n, err)
	}
	if err = ReleaseStorage(context.Background(), db, "C1", 25, 1); err != nil {
		t.Fatal(err)
	}
}
