package database

import (
	"context"
	"database/sql"
	"github.com/karst3nl/reifenapp-2/migrations"
	"path/filepath"
	"testing"
)

func TestAreaMigrationPreservesVersionSeven(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "v7.sqlite")
	old, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = old.Exec("CREATE TABLE schema_migrations(version INTEGER PRIMARY KEY)"); err != nil {
		t.Fatal(err)
	}
	names := []string{"001_initial.sql", "002_storage_details.sql", "003_storage_operation.sql", "004_storage_releases.sql", "005_history_events.sql", "006_pdf_layouts.sql", "007_admin_account.sql"}
	for i, name := range names {
		script, err := migrations.Files.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = old.Exec(string(script)); err != nil {
			t.Fatal(name, err)
		}
		if _, err = old.Exec("INSERT INTO schema_migrations VALUES(?)", i+1); err != nil {
			t.Fatal(err)
		}
	}
	if err = SeedDevelopment(old); err != nil {
		t.Fatal(err)
	}
	if err = EnsureAdminAccount(ctx, old); err != nil {
		t.Fatal(err)
	}
	account, err := LoadAdminAccount(ctx, old)
	if err != nil {
		t.Fatal(err)
	}
	old.Close()
	if db, err := Open(path); err == nil {
		db.Close()
		t.Fatal("implicit upgrade")
	}
	db, err := OpenWithMigration(path, true)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	current, err := LoadAdminAccount(ctx, db)
	if err != nil || current.Hash != account.Hash || current.Revision != account.Revision {
		t.Fatal("account changed", err)
	}
	results, err := Search(ctx, db, "fiktiv")
	if err != nil || len(results) != 2 {
		t.Fatal("history changed", results, err)
	}
	checked, err := CheckPlace(ctx, db, "C1", 25, "", "")
	if err != nil || checked.Status != "occupied" {
		t.Fatal("occupancy changed", checked, err)
	}
	areas, err := StorageAreas(ctx, db)
	if err != nil || len(areas) != 6 {
		t.Fatal(areas, err)
	}
	var foreignKeys int
	if err = db.QueryRow("PRAGMA foreign_keys").Scan(&foreignKeys); err != nil || foreignKeys != 1 {
		t.Fatal("foreign keys", foreignKeys, err)
	}
	rows, err := db.Query("PRAGMA foreign_key_check")
	if err != nil {
		t.Fatal(err)
	}
	if rows.Next() {
		t.Fatal("broken references")
	}
	rows.Close()
	backups, err := filepath.Glob(path + ".backup-*")
	if err != nil || len(backups) != 1 {
		t.Fatal(backups, err)
	}
	backup, err := sql.Open("sqlite3", backups[0])
	if err != nil {
		t.Fatal(err)
	}
	defer backup.Close()
	var version int
	if err = backup.QueryRow("SELECT max(version) FROM schema_migrations").Scan(&version); err != nil || version != 7 {
		t.Fatal("backup schema", version, err)
	}
	if err = SaveStorageArea(ctx, db, StorageArea{Code: "NEU", First: 1, Last: 3}); err != nil {
		t.Fatal(err)
	}
}
