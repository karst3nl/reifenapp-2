package database

import (
	"context"
	"testing"
)

func TestCapacityReservationStorageReleaseAndCatalogue(t *testing.T) {
	ctx := context.Background()
	db := testDB(t)
	check := func(total, reserved, occupied, free int) {
		t.Helper()
		got, err := StorageCapacity(ctx, db)
		want := Capacity{Total: total, Reserved: reserved, Occupied: occupied, Free: free}
		if err != nil || got != want {
			t.Fatalf("got %+v want %+v: %v", got, want, err)
		}
	}
	check(640, 0, 0, 640)
	input := reservationInput()
	if _, err := CreateReservation(ctx, db, input); err != nil {
		t.Fatal(err)
	}
	check(640, 1, 0, 639)
	storage := storageExample()
	storage.Name = input.Name
	storage.Plate = input.Plate
	id, err := CreateStorage(ctx, db, storage)
	if err != nil {
		t.Fatal(err)
	}
	check(640, 0, 1, 639)
	if err = ReleaseStorage(ctx, db, storage.Area, storage.Number, id); err != nil {
		t.Fatal(err)
	}
	check(640, 0, 0, 640)
	// Shrinking a catalogue keeps disabled historical places but excludes them from capacity.
	if err = SaveStorageArea(ctx, db, StorageArea{Code: "OR", First: 1, Last: 50, Revision: 1}); err != nil {
		t.Fatal(err)
	}
	check(630, 0, 0, 630)
	if err = SaveStorageArea(ctx, db, StorageArea{Code: "TEST", First: 1, Last: 3}); err != nil {
		t.Fatal(err)
	}
	check(633, 0, 0, 633)
	if _, err = db.Exec("UPDATE storage_places SET enabled=0"); err != nil {
		t.Fatal(err)
	}
	check(0, 0, 0, 0)
}
