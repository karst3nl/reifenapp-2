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

func reservationInput() models.ReservationInput {
	return models.ReservationInput{Area: "C1", Number: 2, Name: "Reservierung Fiktiv", Plate: "MZ-RF 123", Brand: "VW", Model: "Golf"}
}
func TestReservationLifecycle(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.sqlite")
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	input := reservationInput()
	id, err := CreateReservation(context.Background(), db, input)
	if err != nil {
		t.Fatal(err)
	}
	db.Close()
	db, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var kind, status string
	if err = db.QueryRow("SELECT kind,status FROM storage_claims WHERE id=?", id).Scan(&kind, &status); err != nil {
		t.Fatal(err)
	}
	if kind != "reservation" || status != "active" {
		t.Fatal(kind, status)
	}
	var details int
	if err = db.QueryRow("SELECT count(*) FROM storage_details WHERE claim_id=?", id).Scan(&details); err != nil || details != 0 {
		t.Fatal("unexpected reservation details", details, err)
	}
	for _, q := range []string{input.Name, input.Plate} {
		rows, err := Search(context.Background(), db, q)
		if err != nil || len(rows) != 1 || rows[0].Kind != "reservation" {
			t.Fatalf("%v %v", rows, err)
		}
	}
	if _, err = CreateReservation(context.Background(), db, input); !errors.Is(err, ErrPlaceUnavailable) {
		t.Fatal("duplicate", err)
	}
	storage := storageExample()
	storage.Name = input.Name
	storage.Plate = input.Plate
	if _, err = CreateStorage(context.Background(), db, storage); err != nil {
		t.Fatal(err)
	}
	db.QueryRow("SELECT status FROM storage_claims WHERE id=?", id).Scan(&status)
	if status != "converted" {
		t.Fatal(status)
	}
	rows, err := Search(context.Background(), db, input.Name)
	if err != nil || len(rows) != 2 || rows[0].Kind != "occupancy" || rows[1].Kind != "reservation" || rows[1].Active {
		t.Fatalf("%v %v", rows, err)
	}
	var n int
	db.QueryRow("SELECT count(*) FROM customers").Scan(&n)
	if n != 1 {
		t.Fatal("duplicate customer")
	}
	if _, err = CreateReservation(context.Background(), db, input); !errors.Is(err, ErrPlaceUnavailable) {
		t.Fatal("occupied", err)
	}
}
func TestReservationRollbackAndValidation(t *testing.T) {
	db := testDB(t)
	input := reservationInput()
	input.Name = ""
	if _, err := CreateReservation(context.Background(), db, input); !errors.Is(err, ErrInvalidStorage) {
		t.Fatal(err)
	}
	input = reservationInput()
	input.Area = "C2"
	input.Number = 2
	if _, err := CreateReservation(context.Background(), db, input); !errors.Is(err, ErrInvalidPlace) {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TRIGGER reject_reservation BEFORE INSERT ON storage_claims BEGIN SELECT RAISE(ABORT,'test failure'); END`); err != nil {
		t.Fatal(err)
	}
	if _, err := CreateReservation(context.Background(), db, reservationInput()); err == nil {
		t.Fatal("expected failure")
	}
	for _, table := range []string{"customers", "vehicles", "tire_sets", "storage_claims", "storage_details"} {
		var n int
		if err := db.QueryRow("SELECT count(*) FROM " + table).Scan(&n); err != nil || n != 0 {
			t.Fatalf("%s %d %v", table, n, err)
		}
	}
}
func TestConcurrentReservations(t *testing.T) {
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
	var wg sync.WaitGroup
	result := make(chan error, 2)
	for _, db := range []*sql.DB{a, b} {
		wg.Add(1)
		go func(db *sql.DB) {
			defer wg.Done()
			_, err := CreateReservation(context.Background(), db, reservationInput())
			result <- err
		}(db)
	}
	wg.Wait()
	close(result)
	success, conflict := 0, 0
	for err := range result {
		if err == nil {
			success++
		} else if errors.Is(err, ErrPlaceUnavailable) {
			conflict++
		} else {
			t.Fatal(err)
		}
	}
	if success != 1 || conflict != 1 {
		t.Fatalf("success=%d conflict=%d", success, conflict)
	}
}
