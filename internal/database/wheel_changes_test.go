package database

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"github.com/karst3nl/reifenapp-2/migrations"
)

func TestWheelChangeOperationPersists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.sqlite")
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	input := storageExample()
	id, err := CreateWheelChange(context.Background(), db, input)
	if err != nil {
		t.Fatal(err)
	}
	var operation string
	if err = db.QueryRow("SELECT operation FROM storage_details WHERE claim_id=?", id).Scan(&operation); err != nil || operation != "wheel_change" {
		t.Fatalf("%s %v", operation, err)
	}
	if _, err = CreateWheelChange(context.Background(), db, input); !errors.Is(err, ErrPlaceUnavailable) {
		t.Fatal(err)
	}
	db.Close()
	db, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	rows, err := Search(context.Background(), db, input.Plate)
	if err != nil || len(rows) != 1 || rows[0].Operation != "wheel_change" || rows[0].Kind != "occupancy" {
		t.Fatalf("%v %v", rows, err)
	}
	input.Number = 3
	if _, err = CreateStorage(context.Background(), db, input); err != nil {
		t.Fatal(err)
	}
	var normal string
	if err = db.QueryRow("SELECT operation FROM storage_details ORDER BY claim_id DESC LIMIT 1").Scan(&normal); err != nil || normal != "storage" {
		t.Fatal(normal, err)
	}
	input.Number = 4
	input.FrontDepth = ""
	if _, err = CreateWheelChange(context.Background(), db, input); !errors.Is(err, ErrInvalidStorage) {
		t.Fatal(err)
	}
}
func TestWheelChangeConvertsMatchingReservation(t *testing.T) {
	db := testDB(t)
	input := reserveExample(t, db)
	id, err := CreateWheelChange(context.Background(), db, input)
	if err != nil {
		t.Fatal(err)
	}
	var operation, status string
	if err = db.QueryRow("SELECT operation FROM storage_details WHERE claim_id=?", id).Scan(&operation); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow("SELECT status FROM storage_claims WHERE id=10").Scan(&status); err != nil {
		t.Fatal(err)
	}
	if operation != "wheel_change" || status != "converted" {
		t.Fatal(operation, status)
	}
}
func TestVersionTwoUpgradePreservesStorage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v2.sqlite")
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec("CREATE TABLE schema_migrations(version INTEGER PRIMARY KEY)"); err != nil {
		t.Fatal(err)
	}
	for i, name := range []string{"001_initial.sql", "002_storage_details.sql"} {
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
	if _, err = db.Exec(`INSERT INTO storage_details VALUES(1,'Winter','Stahl','{}','{}','6.5','6.0','{}','Fiktiver Monteur','Testnotiz')`); err != nil {
		t.Fatal(err)
	}
	db.Close()
	if opened, err := Open(path); err == nil {
		opened.Close()
		t.Fatal("upgrade needs explicit flag")
	}
	db, err = OpenWithMigration(path, true)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var operation, notes string
	if err = db.QueryRow("SELECT operation,notes FROM storage_details WHERE claim_id=1").Scan(&operation, &notes); err != nil || operation != "storage" || notes != "Testnotiz" {
		t.Fatal(operation, notes, err)
	}
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
	if err = backup.QueryRow("SELECT MAX(version) FROM schema_migrations").Scan(&version); err != nil || version != 2 {
		t.Fatal(version, err)
	}
	if _, err = db.Exec("UPDATE storage_details SET operation='unknown'"); err == nil {
		t.Fatal("operation constraint missing")
	}
}
