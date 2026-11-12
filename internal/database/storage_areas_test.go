package database

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

func TestStorageAreaLifecycleProtectsClaimsAndHistory(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	a := StorageArea{Code: "HALLE2", First: 10, Last: 12}
	if err := SaveStorageArea(ctx, db, a); err != nil {
		t.Fatal(err)
	}
	input := storageExample()
	input.Area = a.Code
	input.Number = 10
	claim, err := CreateStorage(ctx, db, input)
	if err != nil {
		t.Fatal(err)
	}
	reservation := reservationInput()
	reservation.Area = a.Code
	reservation.Number = 12
	if _, err = CreateReservation(ctx, db, reservation); err != nil {
		t.Fatal(err)
	}
	a.Revision = 1
	a.First = 11
	if err = SaveStorageArea(ctx, db, a); !errors.Is(err, ErrAreaOccupied) {
		t.Fatal("occupied range removal", err)
	}
	a.First = 10
	a.Last = 11
	if err = SaveStorageArea(ctx, db, a); !errors.Is(err, ErrAreaOccupied) {
		t.Fatal("reserved range removal", err)
	}
	a.Last = 15
	if err = SaveStorageArea(ctx, db, a); err != nil {
		t.Fatal(err)
	}
	if err = SaveStorageArea(ctx, db, a); !errors.Is(err, ErrAreaConflict) {
		t.Fatal("stale editor", err)
	}
	if err = ReleaseStorage(ctx, db, a.Code, 10, claim); err != nil {
		t.Fatal(err)
	}
	a.Revision = 2
	a.First = 11
	if err = SaveStorageArea(ctx, db, a); err != nil {
		t.Fatal(err)
	}
	if _, err = CheckPlace(ctx, db, a.Code, 10, "", ""); !errors.Is(err, ErrInvalidPlace) {
		t.Fatal("retired place selectable", err)
	}
	if _, err = CreateStorage(ctx, db, input); !errors.Is(err, ErrInvalidPlace) {
		t.Fatal("retired place writable", err)
	}
	results, err := Search(ctx, db, "HALLE2.10")
	if err != nil || len(results) != 2 {
		t.Fatal("history lost", results, err)
	}
	if _, err = db.Exec(`INSERT INTO storage_claims(place_id,kind,status,updated_at) VALUES('HALLE2.10','reservation','active','now')`); err == nil {
		t.Fatal("inactive place accepted direct claim")
	}
	a.Revision = 3
	a.First = 10
	if err = SaveStorageArea(ctx, db, a); err != nil {
		t.Fatal(err)
	}
	if checked, err := CheckPlace(ctx, db, a.Code, 10, "", ""); err != nil || !checked.Allowed {
		t.Fatal("reactivation", checked, err)
	}
	areas, err := StorageAreas(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	for _, saved := range areas {
		if saved.Code == a.Code && (saved.Occupied != 0 || saved.Reserved != 1 || saved.Revision != 4) {
			t.Fatal(saved)
		}
	}
	var enabled int
	if err = db.QueryRow("PRAGMA foreign_keys").Scan(&enabled); err != nil || enabled != 1 {
		t.Fatal("foreign keys disabled", enabled, err)
	}
}
func TestStorageAreaInputAndRollback(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	for _, a := range []StorageArea{{Code: "bad", First: 1, Last: 2}, {Code: "A.B", First: 1, Last: 2}, {Code: "NEW", First: 0, Last: 2}, {Code: "NEW", First: 3, Last: 2}, {Code: "NEW", First: 1, Last: 10001}, {Code: "NEW", First: 100000, Last: 100001}} {
		if err := SaveStorageArea(ctx, db, a); !errors.Is(err, ErrAreaInput) {
			t.Fatal(a, err)
		}
	}
	if _, err := db.Exec(`CREATE TRIGGER fail_catalogue BEFORE INSERT ON storage_places WHEN NEW.area='FAIL' BEGIN SELECT RAISE(ABORT,'test'); END`); err != nil {
		t.Fatal(err)
	}
	if err := SaveStorageArea(ctx, db, StorageArea{Code: "FAIL", First: 1, Last: 2}); err == nil {
		t.Fatal("expected failure")
	}
	var count int
	if err := db.QueryRow("SELECT count(*) FROM storage_areas WHERE code='FAIL'").Scan(&count); err != nil || count != 0 {
		t.Fatal("partial save", count, err)
	}
}

func TestAreaShrinkConcurrentWithStorage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "concurrent.sqlite")
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
	ctx := context.Background()
	input := storageExample()
	input.Number = 200
	start := make(chan struct{})
	results := make(chan error, 2)
	go func() {
		<-start
		results <- SaveStorageArea(ctx, a, StorageArea{Code: "C1", First: 1, Last: 199, Revision: 1})
	}()
	go func() { <-start; _, err := CreateStorage(ctx, b, input); results <- err }()
	close(start)
	success, conflicts := 0, 0
	for range 2 {
		err := <-results
		if err == nil {
			success++
		} else if errors.Is(err, ErrAreaOccupied) || errors.Is(err, ErrInvalidPlace) {
			conflicts++
		} else {
			t.Fatal(err)
		}
	}
	if success != 1 || conflicts != 1 {
		t.Fatal(success, conflicts)
	}
	var invalid int
	if err = a.QueryRow(`SELECT count(*) FROM storage_claims c JOIN storage_places p ON p.id=c.place_id WHERE c.status='active' AND p.enabled=0`).Scan(&invalid); err != nil || invalid != 0 {
		t.Fatal("inactive occupied place", invalid, err)
	}
}
