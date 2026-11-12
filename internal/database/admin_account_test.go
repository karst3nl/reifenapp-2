package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/karst3nl/reifenapp-2/internal/receipts"
	"github.com/karst3nl/reifenapp-2/migrations"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestAdminAccountIsHashedPersistentAndNotReset(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	if err := EnsureAdminAccount(ctx, db); err != nil {
		t.Fatal(err)
	}
	a, err := LoadAdminAccount(ctx, db)
	if err != nil || !a.MustChange || a.Revision != 1 || a.Hash == DefaultAdminPassword || bcrypt.CompareHashAndPassword([]byte(a.Hash), []byte(DefaultAdminPassword)) != nil {
		t.Fatal("initial hash", err)
	}
	for _, password := range []string{"short", DefaultAdminPassword, strings.Repeat("a", 73), strings.Repeat(" ", 15)} {
		if _, err = ChangeAdminPassword(ctx, db, 1, "", password); !errors.Is(err, ErrAdminPolicy) {
			t.Fatal("policy", err)
		}
	}
	next, err := ChangeAdminPassword(ctx, db, 1, "", "NeuesFiktivesPasswort123!")
	if err != nil || next.MustChange || next.Revision != 2 {
		t.Fatal(next.MustChange, next.Revision, err)
	}
	if err = EnsureAdminAccount(ctx, db); err != nil {
		t.Fatal(err)
	}
	stored, err := LoadAdminAccount(ctx, db)
	if err != nil || stored.Hash != next.Hash {
		t.Fatal("account reset", err)
	}
	if bcrypt.CompareHashAndPassword([]byte(stored.Hash), []byte(DefaultAdminPassword)) == nil {
		t.Fatal("default hash retained")
	}
	if _, err = ChangeAdminPassword(ctx, db, 1, "", "AnderesFiktivesPasswort123!"); !errors.Is(err, ErrAdminRevision) {
		t.Fatal(err)
	}
	if _, err = ChangeAdminPassword(ctx, db, 2, "wrong", "AnderesFiktivesPasswort123!"); !errors.Is(err, ErrAdminCredentials) {
		t.Fatal(err)
	}
	if _, err = ChangeAdminPassword(ctx, db, 2, "NeuesFiktivesPasswort123!", "AnderesFiktivesPasswort123!"); err != nil {
		t.Fatal(err)
	}
}

func TestAdminMigrationPreservesTemplatesAndHistory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v6.sqlite")
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec("CREATE TABLE schema_migrations(version INTEGER PRIMARY KEY)"); err != nil {
		t.Fatal(err)
	}
	for i, name := range []string{"001_initial.sql", "002_storage_details.sql", "003_storage_operation.sql", "004_storage_releases.sql", "005_history_events.sql", "006_pdf_layouts.sql"} {
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

	data, _ := json.Marshal(receipts.DefaultTemplate("storage"))
	if _, err = db.Exec("INSERT INTO pdf_layouts(operation,layout,enabled,revision,updated_at) VALUES('storage',?,0,3,'2026-11-12T12:00:00Z')", string(data)); err != nil {
		t.Fatal(err)
	}
	db.Close()
	if opened, err := Open(path); err == nil {
		opened.Close()
		t.Fatal("implicit migration")
	}
	db, err = OpenWithMigration(path, true)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = EnsureAdminAccount(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	for table, want := range map[string]int{"customers": 2, "history_events": 2, "pdf_layouts": 1, "admin_account": 1} {
		var n int
		if err = db.QueryRow("SELECT count(*) FROM " + table).Scan(&n); err != nil || n != want {
			t.Fatal(table, n, err)
		}
	}
	var revision int
	if err = db.QueryRow("SELECT revision FROM pdf_layouts WHERE operation='storage'").Scan(&revision); err != nil || revision != 3 {
		t.Fatal(revision, err)
	}
	backups, _ := filepath.Glob(path + ".backup-*")
	if len(backups) != 1 {
		t.Fatal(backups)
	}
	backup, err := sql.Open("sqlite3", backups[0]+"?mode=ro")
	if err != nil {
		t.Fatal(err)
	}
	defer backup.Close()
	var version int
	if err = backup.QueryRow("SELECT MAX(version) FROM schema_migrations").Scan(&version); err != nil || version != 6 {
		t.Fatal(version, err)
	}
}
