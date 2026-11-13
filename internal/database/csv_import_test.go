package database

import (
	"bytes"
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/karst3nl/reifenapp-2/internal/importcsv"
)

func TestCSVArchivePreservationAndDeduplication(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	raw := "DATUM,TÄTIGKEIT,NAME,KENNZEICHEN,LAGERPLATZ,AUTO,REIFENART,PROFILTIEFEVORNE,HINWEISE\n01.08.2024 12:00:00,Neueinlagerung,Archivkunde Fiktiv,MZ-AF 123,C1.222,Testauto,Allwetter,1.6mm,Originalhinweis\n02.08.2024 12:00:00,Auslagerung,Archivkunde Fiktiv,MZ-AF 123,C1.222\n"
	doc, err := importcsv.Parse([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	preview, err := PreviewCSV(ctx, db, doc, "fiktiv.csv")
	if err != nil || preview.New != 2 {
		t.Fatalf("preview %+v %v", preview, err)
	}
	var count int
	db.QueryRow("SELECT count(*) FROM history_events").Scan(&count)
	if count != 0 {
		t.Fatal("preview wrote history")
	}
	result, err := ImportCSV(ctx, db, doc, "fiktiv.csv")
	if err != nil || result.Imported != 2 || result.Backup == "" {
		t.Fatalf("import %+v %v", result, err)
	}
	var dbpath string
	db.QueryRow("SELECT file FROM pragma_database_list WHERE name='main'").Scan(&dbpath)
	if _, err = os.Stat(filepath.Join(filepath.Dir(dbpath), result.Backup)); err != nil {
		t.Fatal(err)
	}
	name, original, err := OriginalCSV(ctx, db, result.ID)
	if err != nil || name != "fiktiv.csv" || string(original) != raw {
		t.Fatal("raw original lost")
	}
	rows, err := Search(ctx, db, "Originalhinweis")
	if err != nil || len(rows) != 1 || rows[0].Source != "csv" || rows[0].Active || rows[0].ReceiptURL != "" || rows[0].Details.Import == nil {
		t.Fatalf("archive search: %+v %v", rows, err)
	}
	page, err := BrowseHistory(ctx, db, HistoryFilter{Source: "csv", Status: "archived", Page: 1, PageSize: 20})
	if err != nil || page.Total != 2 {
		t.Fatalf("history %+v %v", page, err)
	}
	repeat, err := ImportCSV(ctx, db, doc, "renamed.csv")
	if err != nil || !repeat.AlreadyImported {
		t.Fatal("same-file dedup failed", err)
	}
	overlap, err := importcsv.Parse([]byte(strings.Replace(raw, "Originalhinweis", "Originalhinweis", 1) + "03.08.2024,Radwechsel,Archivkunde Fiktiv,MZ-AF 123,C1.222\n"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := ImportCSV(ctx, db, overlap, "overlap.csv")
	if err != nil || second.Imported != 1 || second.Duplicates != 2 {
		t.Fatalf("overlap %+v %v", second, err)
	}
	for _, table := range []string{"storage_claims", "customers", "vehicles", "tire_sets"} {
		if err = db.QueryRow("SELECT count(*) FROM " + table).Scan(&count); err != nil || count != 0 {
			t.Fatalf("live inventory changed %s: %d %v", table, count, err)
		}
	}
	warnings, err := TreadWarnings(ctx, db, true)
	if err != nil || len(warnings) != 0 {
		t.Fatal("archived warning became current", err)
	}
	backup, err := CreateBackup(ctx, db, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer backup.Close()
	files := verifyBackupExports(t, backup)
	if !bytes.Contains(files["daten.json"], []byte("csv_import_batches")) {
		t.Fatal("missing archive backup")
	}
	restoredPath := filepath.Join(t.TempDir(), "restored.sqlite")
	if err = os.WriteFile(restoredPath, files["datenbank.sqlite"], 0600); err != nil {
		t.Fatal(err)
	}
	restored, err := sql.Open("sqlite3", restoredPath)
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Close()
	_, restoredRaw, err := OriginalCSV(ctx, restored, result.ID)
	if err != nil || !bytes.Equal(restoredRaw, []byte(raw)) {
		t.Fatal("backup lost original CSV", err)
	}
	if _, err = db.Exec("DELETE FROM history_events WHERE source='csv'"); err == nil {
		t.Fatal("history mutable")
	}
}

func TestCSVImportRollback(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	doc, err := importcsv.Parse([]byte("DATUM,TÄTIGKEIT,NAME,KENNZEICHEN,LAGERPLATZ\n01.08.2024,Auslagerung,Fiktiver Test,MZ-FT 123,C1.222\n"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec("CREATE TRIGGER fail_csv_row BEFORE INSERT ON csv_import_rows BEGIN SELECT RAISE(ABORT,'simulated failure'); END"); err != nil {
		t.Fatal(err)
	}
	if _, err = ImportCSV(ctx, db, doc, "fiktiv.csv"); err == nil {
		t.Fatal("expected failure")
	}
	for _, table := range []string{"csv_import_batches", "csv_import_rows", "history_events"} {
		var count int
		if err = db.QueryRow("SELECT count(*) FROM " + table).Scan(&count); err != nil || count != 0 {
			t.Fatalf("partial write %s %d %v", table, count, err)
		}
	}
}
