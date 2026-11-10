package database

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"sync"
	"testing"

	"github.com/karst3nl/reifenapp-2/internal/models"
	"github.com/karst3nl/reifenapp-2/migrations"
)

func storageExample() models.StorageInput {
	return models.StorageInput{Area: "C1", Number: 2, Name: "Fiktiv Neueinlagerung", Plate: "MZ-NE 987", Brand: "VW", Model: "Golf", TireType: "Winter", RimType: "Stahl", Front: models.TireSize{Width: "205", Ratio: "55", Inches: "16", Load: "91", Speed: "H"}, Rear: models.TireSize{Width: "205", Ratio: "55", Inches: "16", Load: "91", Speed: "H"}, FrontDepth: "6.5", RearDepth: "7.0", Hubcaps: models.Accessory{Present: true, Location: "Am Radsatz"}, Mechanic: "Testmonteur", Notes: "Ausschließlich fiktiv"}
}
func TestCreateStoragePersistsAndSearches(t *testing.T) {
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
	var depth, accessories, brand, stamp string
	if err = db.QueryRow(`SELECT d.front_depth,d.accessories,v.brand,c.updated_at FROM storage_details d JOIN storage_claims c ON c.id=d.claim_id JOIN tire_sets t ON t.id=c.tire_set_id JOIN vehicles v ON v.id=t.vehicle_id WHERE d.claim_id=?`, id).Scan(&depth, &accessories, &brand, &stamp); err != nil {
		t.Fatal(err)
	}
	if depth != "6.5" || brand != "VW" || accessories == "" || stamp == "" {
		t.Fatal("missing details")
	}
	db.Close()
	db, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, q := range []string{input.Name, input.Plate} {
		rows, err := Search(context.Background(), db, q)
		if err != nil || len(rows) != 1 || rows[0].Place != "C1.2" {
			t.Fatalf("search %v %v", rows, err)
		}
	}
	_, err = CreateStorage(context.Background(), db, input)
	if !errors.Is(err, ErrPlaceUnavailable) {
		t.Fatal(err)
	}
	for _, table := range []string{"customers", "vehicles", "tire_sets", "storage_claims", "storage_details"} {
		var n int
		if err = db.QueryRow("SELECT count(*) FROM " + table).Scan(&n); err != nil || n != 1 {
			t.Fatalf("rollback %s %d %v", table, n, err)
		}
	}
}
func TestReservedAndConcurrentStorage(t *testing.T) {
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
	_, err = a.Exec(`INSERT INTO storage_claims(place_id,kind,status,updated_at) VALUES('C1.2','reservation','active','now')`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = CreateStorage(context.Background(), a, storageExample()); !errors.Is(err, ErrPlaceUnavailable) {
		t.Fatal(err)
	}
	input := storageExample()
	input.Number = 3
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, db := range []*sql.DB{a, b} {
		wg.Add(1)
		go func(db *sql.DB) {
			defer wg.Done()
			_, err := CreateStorage(context.Background(), db, input)
			results <- err
		}(db)
	}
	wg.Wait()
	close(results)
	success, conflict := 0, 0
	for err := range results {
		if err == nil {
			success++
		} else if errors.Is(err, ErrPlaceUnavailable) {
			conflict++
		} else {
			t.Fatal(err)
		}
	}
	if success != 1 || conflict != 1 {
		t.Fatalf("success=%d conflicts=%d", success, conflict)
	}
}
func TestMigrationBackupPreservesV1(t *testing.T) {
	path := filepath.Join(t.TempDir(), "existing.sqlite")
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	script, err := migrations.Files.ReadFile("001_initial.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec("CREATE TABLE schema_migrations(version INTEGER PRIMARY KEY);" + string(script) + "INSERT INTO schema_migrations VALUES(1)"); err != nil {
		t.Fatal(err)
	}
	if err = SeedDevelopment(db); err != nil {
		t.Fatal(err)
	}
	db.Close()
	if db, err = Open(path); err == nil {
		db.Close()
		t.Fatal("upgrade without explicit flag")
	}
	db, err = OpenWithMigration(path, true)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var version int
	db.QueryRow("SELECT MAX(version) FROM schema_migrations").Scan(&version)
	if version != 2 {
		t.Fatal(version)
	}
	rows, err := Search(context.Background(), db, "fiktiv")
	if err != nil || len(rows) != 2 {
		t.Fatalf("existing data: %v %v", rows, err)
	}
	backups, err := filepath.Glob(path + ".backup-*")
	if err != nil || len(backups) != 1 {
		t.Fatalf("backup: %v %v", backups, err)
	}
	backup, err := sql.Open("sqlite3", backups[0]+"?mode=ro")
	if err != nil {
		t.Fatal(err)
	}
	defer backup.Close()
	backup.QueryRow("SELECT MAX(version) FROM schema_migrations").Scan(&version)
	if version != 1 {
		t.Fatal("backup not v1")
	}
	var n int
	backup.QueryRow("SELECT count(*) FROM customers").Scan(&n)
	if n != 2 {
		t.Fatal("backup lost customers")
	}
}

func TestMigrationFailureRollsBack(t *testing.T) {
	path := filepath.Join(t.TempDir(), "broken-upgrade.sqlite")
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	script, err := migrations.Files.ReadFile("001_initial.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec("CREATE TABLE schema_migrations(version INTEGER PRIMARY KEY);" + string(script) + "INSERT INTO schema_migrations VALUES(1); CREATE TABLE storage_details(dummy TEXT)"); err != nil {
		t.Fatal(err)
	}
	db.Close()
	if opened, err := OpenWithMigration(path, true); err == nil {
		opened.Close()
		t.Fatal("expected failed migration")
	}
	db, err = sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var version int
	if err = db.QueryRow("SELECT MAX(version) FROM schema_migrations").Scan(&version); err != nil || version != 1 {
		t.Fatalf("version %d %v", version, err)
	}
	if _, err = db.Exec("SELECT brand FROM vehicles"); err == nil {
		t.Fatal("ALTER TABLE was not rolled back")
	}
	backups, err := filepath.Glob(path + ".backup-*")
	if err != nil || len(backups) != 1 {
		t.Fatal("missing recovery backup")
	}
}
