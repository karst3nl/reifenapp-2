package database

import (
	"context"
	"database/sql"
	"errors"
	"github.com/karst3nl/reifenapp-2/migrations"
	"path/filepath"
	"testing"

	"github.com/karst3nl/reifenapp-2/internal/receipts"
)

func TestPDFLayoutsPersistAndRejectStaleWrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.sqlite")
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	layout, err := LoadPDFLayout(ctx, db, "storage")
	if err != nil || layout.Enabled || layout.Revision != 0 {
		t.Fatal(layout, err)
	}
	layout.Enabled = true
	layout.Template.Elements[0].X = 15
	saved, err := SavePDFLayout(ctx, db, layout)
	if err != nil || saved.Revision != 1 {
		t.Fatal(saved, err)
	}
	if _, err = SavePDFLayout(ctx, db, layout); !errors.Is(err, ErrLayoutConflict) {
		t.Fatal(err)
	}
	saved.Enabled = false
	saved, err = SavePDFLayout(ctx, db, saved)
	if err != nil || saved.Revision != 2 {
		t.Fatal(err)
	}
	db.Close()
	db, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	loaded, err := LoadPDFLayout(ctx, db, "storage")
	if err != nil || loaded.Enabled || loaded.Template.Elements[0].X != 15 {
		t.Fatal(loaded, err)
	}
	other, err := LoadPDFLayout(ctx, db, "reservation")
	if err != nil || other.Revision != 0 {
		t.Fatal(other, err)
	}
	loaded.Template = receipts.Template{}
	if _, err = SavePDFLayout(ctx, db, loaded); err == nil {
		t.Fatal("invalid layout saved")
	}
}

func TestLayoutMigrationFromVersionFivePreservesHistory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v5.sqlite")
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec("CREATE TABLE schema_migrations(version INTEGER PRIMARY KEY)"); err != nil {
		t.Fatal(err)
	}
	for i, name := range []string{"001_initial.sql", "002_storage_details.sql", "003_storage_operation.sql", "004_storage_releases.sql", "005_history_events.sql"} {
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
		t.Fatal("implicit upgrade")
	}
	db, err = OpenWithMigration(path, true)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for table, want := range map[string]int{"history_events": 2, "storage_claims": 2, "pdf_layouts": 0} {
		var count int
		if err = db.QueryRow("SELECT count(*) FROM " + table).Scan(&count); err != nil || count != want {
			t.Fatal(table, count, err)
		}
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
	if err = backup.QueryRow("SELECT MAX(version) FROM schema_migrations").Scan(&version); err != nil || version != 5 {
		t.Fatal(version, err)
	}
}
