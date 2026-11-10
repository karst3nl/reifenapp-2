package database

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"sync"
	"testing"

	"github.com/karst3nl/reifenapp-2/internal/models"
)

func reserveExample(t *testing.T, db *sql.DB) models.StorageInput {
	t.Helper()
	for _, query := range []string{
		`INSERT INTO customers(id,name) VALUES(10,'Reserviert Fiktiv')`,
		`INSERT INTO vehicles(id,customer_id,plate,car) VALUES(10,10,'MZ-RE 123','VW Golf')`,
		`INSERT INTO tire_sets(id,vehicle_id) VALUES(10,10)`,
		`INSERT INTO storage_claims(id,place_id,tire_set_id,kind,status,updated_at) VALUES(10,'C1.2',10,'reservation','active','2026-11-10T00:00:00Z')`,
	} {
		if _, err := db.Exec(query); err != nil {
			t.Fatal(err)
		}
	}
	input := storageExample()
	input.Name = "Reserviert Fiktiv"
	input.Plate = "MZ-RE 123"
	return input
}
func TestPlaceCheck(t *testing.T) {
	db := testDB(t)
	input := reserveExample(t, db)
	ctx := context.Background()
	for _, tc := range []struct {
		number              int
		name, plate, status string
		allowed             bool
	}{{3, "", "", "free", true}, {2, "", "", "reserved", false}, {2, "Fremder Fiktiv", input.Plate, "reserved", false}, {2, input.Name, "MZ-RE 124", "reserved", false}, {2, "  RESERVIERT   Fiktiv ", "mz re123", "reserved", true}} {
		c, err := CheckPlace(ctx, db, "C1", tc.number, tc.name, tc.plate)
		if err != nil || c.Status != tc.status || c.Allowed != tc.allowed {
			t.Fatalf("%+v %v", c, err)
		}
	}
	if _, err := CheckPlace(ctx, db, "C2", 200, "", ""); !errors.Is(err, ErrInvalidPlace) {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO storage_claims(place_id,kind,status,updated_at) VALUES('C1.4','reservation','active','now')`); err != nil {
		t.Fatal(err)
	}
	c, err := CheckPlace(ctx, db, "C1", 4, input.Name, input.Plate)
	if err != nil || c.Allowed {
		t.Fatalf("unassigned reservation: %+v %v", c, err)
	}
}
func TestMatchingReservationConversion(t *testing.T) {
	db := testDB(t)
	input := reserveExample(t, db)
	ctx := context.Background()
	mismatch := input
	mismatch.Name = "Fremder Fiktiv"
	if _, err := CreateStorage(ctx, db, mismatch); !errors.Is(err, ErrPlaceUnavailable) {
		t.Fatal(err)
	}
	mismatch = input
	mismatch.Plate = "MZ-RE 124"
	if _, err := CreateStorage(ctx, db, mismatch); !errors.Is(err, ErrPlaceUnavailable) {
		t.Fatal(err)
	}
	id, err := CreateStorage(ctx, db, input)
	if err != nil {
		t.Fatal(err)
	}
	var status string
	if err = db.QueryRow("SELECT status FROM storage_claims WHERE id=10").Scan(&status); err != nil || status != "converted" {
		t.Fatalf("%s %v", status, err)
	}
	var tire int
	if err = db.QueryRow("SELECT tire_set_id FROM storage_claims WHERE id=?", id).Scan(&tire); err != nil || tire != 10 {
		t.Fatalf("reuse %d %v", tire, err)
	}
	var count int
	db.QueryRow("SELECT count(*) FROM customers").Scan(&count)
	if count != 1 {
		t.Fatal("duplicated customer")
	}
	checked, err := CheckPlace(ctx, db, "C1", 2, input.Name, input.Plate)
	if err != nil || checked.Status != "occupied" || checked.Allowed {
		t.Fatalf("%+v %v", checked, err)
	}
	rows, err := Search(ctx, db, input.Plate)
	if err != nil || len(rows) != 1 || rows[0].Place != "C1.2" {
		t.Fatalf("%v %v", rows, err)
	}
}
func TestConversionRollsBackOnFailure(t *testing.T) {
	db := testDB(t)
	input := reserveExample(t, db)
	if _, err := db.Exec(`CREATE TRIGGER reject_detail BEFORE INSERT ON storage_details BEGIN SELECT RAISE(ABORT,'test failure'); END`); err != nil {
		t.Fatal(err)
	}
	if _, err := CreateStorage(context.Background(), db, input); err == nil {
		t.Fatal("expected failure")
	}
	var status string
	db.QueryRow("SELECT status FROM storage_claims WHERE id=10").Scan(&status)
	if status != "active" {
		t.Fatal(status)
	}
	var count int
	db.QueryRow("SELECT count(*) FROM storage_claims").Scan(&count)
	if count != 1 {
		t.Fatal("partial occupancy")
	}
}
func TestConcurrentReservationConversion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.sqlite")
	a, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	input := reserveExample(t, a)
	b, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
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
		t.Fatalf("success %d conflict %d", success, conflict)
	}
}
