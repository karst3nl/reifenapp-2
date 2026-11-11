package database

import (
	"context"
	"database/sql"
	"path/filepath"
	"sync"
	"testing"
)

func testDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), "test.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}
func TestDatabase(t *testing.T) {
	db := testDB(t)
	var mode string
	if err := db.QueryRow("PRAGMA journal_mode").Scan(&mode); err != nil || mode != "wal" {
		t.Fatalf("WAL: %s %v", mode, err)
	}
	var count int
	db.QueryRow("SELECT count(*) FROM storage_places").Scan(&count)
	if count != 640 {
		t.Fatal(count)
	}
	for _, s := range []string{
		`INSERT INTO storage_places VALUES('C2.2','C2',2)`,
		`INSERT INTO storage_claims(place_id,kind,status,updated_at) VALUES('BAD','reservation','active','now')`,
	} {
		if _, err := db.Exec(s); err == nil {
			t.Fatalf("expected rejection: %s", s)
		}
	}
}
func TestSearch(t *testing.T) {
	db := testDB(t)
	second := storageExample()
	second.Area = "C2"
	second.Number = 215
	second.Name = "Erika Beispiel (fiktiv)"
	second.Plate = "MZ-EB 456"
	if _, err := CreateStorage(context.Background(), db, second); err != nil {
		t.Fatal(err)
	}
	first := storageExample()
	first.Number = 25
	first.Name = "Müller (fiktiv)"
	first.Plate = "MZ-MM 123"
	if _, err := CreateStorage(context.Background(), db, first); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		q string
		n int
	}{{"", 0}, {"MÜLLER", 1}, {"mzmm123", 1}, {"C1.25", 1}, {"Beispiel", 1}, {"unbekannt", 0}, {"' OR 1=1 --", 0}} {
		rows, err := Search(context.Background(), db, tc.q)
		if err != nil || len(rows) != tc.n {
			t.Fatalf("%q: %v %v", tc.q, rows, err)
		}
	}
	rows, err := Search(context.Background(), db, "fiktiv")
	if err != nil || len(rows) != 2 || rows[0].Place != "C1.25" {
		t.Fatalf("order: %v %v", rows, err)
	}
}
func TestActiveClaims(t *testing.T) {
	db := testDB(t)
	if err := SeedDevelopment(db); err != nil {
		t.Fatal(err)
	}
	reserve := func(place string) error {
		_, err := db.Exec(`INSERT INTO storage_claims(place_id,kind,status,updated_at) VALUES(?,'reservation','active','now')`, place)
		return err
	}
	if err := reserve("C1.25"); err == nil {
		t.Fatal("reserved occupied place")
	}
	if err := reserve("C1.3"); err != nil {
		t.Fatal(err)
	}
	if err := reserve("C1.3"); err == nil {
		t.Fatal("double reservation")
	}
	if _, err := db.Exec(`UPDATE storage_claims SET place_id='C1.3' WHERE id=1`); err == nil {
		t.Fatal("occupancy on reservation")
	}
	if _, err := db.Exec(`INSERT INTO storage_claims(place_id,tire_set_id,kind,status,updated_at) VALUES('C1.25',2,'occupancy','active','now')`); err == nil {
		t.Fatal("double occupancy")
	}
	var wg sync.WaitGroup
	success := make(chan bool, 2)
	for range 2 {
		wg.Add(1)
		go func() { defer wg.Done(); success <- reserve("C1.4") == nil }()
	}
	wg.Wait()
	close(success)
	n := 0
	for ok := range success {
		if ok {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("concurrent claims: %d", n)
	}
}
func TestReopenPreservesData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.sqlite")
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = SeedDevelopment(db); err != nil {
		t.Fatal(err)
	}
	db.Close()
	db, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	r, err := Search(context.Background(), db, "fiktiv")
	if err != nil || len(r) != 2 {
		t.Fatalf("%v %v", r, err)
	}
}
