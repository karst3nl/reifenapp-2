package database

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/karst3nl/reifenapp-2/internal/receipts"
)

func backupEntries(t *testing.T, path string) map[string][]byte {
	t.Helper()
	archive, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()
	files := map[string][]byte{}
	for _, entry := range archive.File {
		reader, err := entry.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(reader)
		reader.Close()
		if err != nil {
			t.Fatal(err)
		}
		if entry.Mode().Perm() != 0600 {
			t.Fatal("unsafe archive permissions", entry.Name, entry.Mode())
		}
		files[filepath.Base(entry.Name)] = data
	}
	return files
}
func verifyBackupExports(t *testing.T, backup *Backup) map[string][]byte {
	t.Helper()
	files := backupEntries(t, backup.Path)
	for _, name := range []string{"datenbank.sqlite", "uebersicht.html", "daten.json", "schema.sql", "manifest.json", "BITTE-LESEN.txt"} {
		if len(files[name]) == 0 {
			t.Fatal("missing file", name)
		}
	}
	var manifest BackupManifest
	if err := json.Unmarshal(files["manifest.json"], &manifest); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(files["datenbank.sqlite"])
	if fmt.Sprintf("%x", sum) != manifest.DatabaseSHA256 {
		t.Fatal("wrong database checksum")
	}
	path := filepath.Join(t.TempDir(), "restored.sqlite")
	if err := os.WriteFile(path, files["datenbank.sqlite"], 0600); err != nil {
		t.Fatal(err)
	}
	db, err := Open(path)
	if err != nil {
		t.Fatal("cannot open complete backup", err)
	}
	defer db.Close()
	var integrity string
	if err = db.QueryRow("PRAGMA integrity_check").Scan(&integrity); err != nil || integrity != "ok" {
		t.Fatal(integrity, err)
	}
	var data struct{ Tables map[string][]map[string]any }
	if err = json.Unmarshal(files["daten.json"], &data); err != nil {
		t.Fatal("invalid JSON export", err)
	}
	for table, count := range manifest.Tables {
		var restored int
		if err = db.QueryRow("SELECT count(*) FROM " + quotedIdentifier(table)).Scan(&restored); err != nil || restored != count || len(data.Tables[table]) != count {
			t.Fatal("inconsistent snapshot", table, restored, count, len(data.Tables[table]), err)
		}
	}
	return files
}
func TestBackupIncludesWALHistorySettingsAndReadableExports(t *testing.T) {
	ctx := context.Background()
	db := testDB(t)
	if _, err := db.Exec("PRAGMA wal_autocheckpoint=0"); err != nil {
		t.Fatal(err)
	}
	input := storageExample()
	input.Name = "Backup Müller (fiktiv)"
	input.Notes = "Fiktiv <script>window.bad=true</script>\nMit Umlauten äöü."
	first, err := CreateStorage(ctx, db, input)
	if err != nil {
		t.Fatal(err)
	}
	input.TireType = "Sommer"
	if _, err = CreateWheelChange(ctx, db, input); err != nil {
		t.Fatal(err)
	}
	r := reservationInput()
	r.Number = 10
	if _, err = CreateReservation(ctx, db, r); err != nil {
		t.Fatal(err)
	}
	if err = SaveStorageArea(ctx, db, StorageArea{Code: "HALLE2", First: 1, Last: 3}); err != nil {
		t.Fatal(err)
	}
	if err = EnsureAdminAccount(ctx, db); err != nil {
		t.Fatal(err)
	}
	account, err := LoadAdminAccount(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	layout := PDFLayout{Operation: "storage", Template: receipts.DefaultTemplate("storage"), Enabled: true}
	var logo bytes.Buffer
	if err = png.Encode(&logo, image.NewNRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	logoData := "data:image/png;base64," + base64.StdEncoding.EncodeToString(logo.Bytes())
	layout.Template.Elements = append(layout.Template.Elements, receipts.Element{ID: "fictitious-logo", Type: "image", Image: logoData, X: 140, Y: 2, Width: 40, Height: 12, Color: "#000000", Background: "#ffffff"})
	if _, err = SavePDFLayout(ctx, db, layout); err != nil {
		t.Fatal(err)
	}
	var source string
	rows, err := db.Query("PRAGMA database_list")
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var seq int
		var name string
		var path string
		rows.Scan(&seq, &name, &path)
		if name == "main" {
			source = path
		}
	}
	rows.Close()
	if stat, err := os.Stat(source + "-wal"); err != nil || stat.Size() == 0 {
		t.Fatal("fixture has no WAL")
	}
	header := []byte("{\"company\":\"Firma Fiktiv\"}")
	backup, err := CreateBackup(ctx, db, header)
	if err != nil {
		t.Fatal(err)
	}
	directory := backup.directory
	files := verifyBackupExports(t, backup)
	if !bytes.Equal(files["receipt-header.json"], header) {
		t.Fatal("missing workshop header")
	}
	if strings.Contains(string(files["uebersicht.html"]), "<script>window.bad") || !strings.Contains(string(files["uebersicht.html"]), "&lt;script&gt;window.bad") {
		t.Fatal("unsafe or missing HTML text")
	}
	if bytes.Contains(files["daten.json"], []byte(account.Hash)) || bytes.Contains(files["uebersicht.html"], []byte(account.Hash)) {
		t.Fatal("password hash exposed in readable exports")
	}
	path := filepath.Join(t.TempDir(), "complete.sqlite")
	os.WriteFile(path, files["datenbank.sqlite"], 0600)
	restored, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Close()
	restoredAccount, err := LoadAdminAccount(ctx, restored)
	if err != nil || restoredAccount.Hash != account.Hash {
		t.Fatal("complete SQLite copy lost admin account", err)
	}
	if _, err = Receipt(ctx, restored, first, "storage"); err != nil {
		t.Fatal("history lost", err)
	}
	pdf, err := LoadPDFLayout(ctx, restored, "storage")
	if err != nil || !pdf.Enabled || pdf.Template.Elements[len(pdf.Template.Elements)-1].Image != logoData {
		t.Fatal("layout lost", err)
	}
	if err = backup.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(directory); !os.IsNotExist(err) {
		t.Fatal("temporary backup remains", err)
	}
	if sourceRows, err := Search(ctx, db, input.Plate); err != nil || len(sourceRows) != 2 {
		t.Fatal("live data changed", sourceRows, err)
	}
	// Exported schema must be usable on an empty SQLite database.
	schemaDB, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "schema.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer schemaDB.Close()
	if _, err = schemaDB.Exec(string(files["schema.sql"])); err != nil {
		t.Fatal("schema export invalid", err)
	}
}
func TestBackupConcurrentWritesRemainConsistent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "live.sqlite")
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	other, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	if err = SeedDevelopment(db); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	start := make(chan struct{})
	go func() {
		<-start
		for n := 30; n < 50; n++ {
			input := storageExample()
			input.Number = n
			if _, err := CreateStorage(context.Background(), other, input); err != nil {
				done <- err
				return
			}
		}
		done <- nil
	}()
	close(start)
	backup, err := CreateBackup(context.Background(), db, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer backup.Close()
	if err = <-done; err != nil {
		t.Fatal(err)
	}
	verifyBackupExports(t, backup)
}
func TestBackupCancellationAndFailureLeaveNoArtifacts(t *testing.T) {
	db := testDB(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if backup, err := CreateBackup(ctx, db, nil); err == nil || backup != nil {
		t.Fatal("canceled request succeeded")
	}
	// A bad optional header fails after snapshot generation, then removes staging.
	base := t.TempDir()
	t.Setenv("TMPDIR", base)
	if backup, err := CreateBackup(context.Background(), db, []byte("invalid JSON")); err == nil || backup != nil {
		t.Fatal("invalid header succeeded")
	}
	entries, err := os.ReadDir(base)
	if err != nil || len(entries) != 0 {
		t.Fatal("temporary customer data retained", entries, err)
	}
	db.Close()
	if _, err = CreateBackup(context.Background(), db, nil); err == nil {
		t.Fatal("closed database succeeded")
	}
}
func TestBackupReadOnlyCopyHasNoSidecars(t *testing.T) {
	db := testDB(t)
	backup, err := CreateBackup(context.Background(), db, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer backup.Close()
	u := url.URL{Scheme: "file", Path: filepath.Join(backup.directory, "datenbank.sqlite")}
	restored, err := sql.Open("sqlite3", u.String()+"?mode=ro&immutable=1")
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Close()
	if err = restored.Ping(); err != nil {
		t.Fatal(err)
	}
	for _, suffix := range []string{"-wal", "-shm"} {
		if _, err = os.Stat(u.Path + suffix); !os.IsNotExist(err) {
			t.Fatal("backup depends on sidecar", suffix, err)
		}
	}
}
