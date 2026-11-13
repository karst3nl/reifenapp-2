package database

import (
	"context"
	"testing"
)

func TestTreadWarningOverviewLifecycle(t *testing.T) {
	ctx := context.Background()
	db := testDB(t)
	low := storageExample()
	low.FrontDepth = "4.0"
	low.RearDepth = "3.5"
	claim, err := CreateStorage(ctx, db, low)
	if err != nil {
		t.Fatal(err)
	}
	healthy := storageExample()
	healthy.Number = 3
	healthy.Notes = "Nur ein fiktiver Hinweis"
	if _, err = CreateStorage(ctx, db, healthy); err != nil {
		t.Fatal(err)
	}
	reserved := reservationInput()
	reserved.Number = 4
	if _, err = CreateReservation(ctx, db, reserved); err != nil {
		t.Fatal(err)
	}
	rows, err := TreadWarnings(ctx, db, true)
	if err != nil || len(rows) != 1 || len(rows[0].TreadWarnings) != 2 || rows[0].Place != low.Place() {
		t.Fatal(rows, err)
	}
	found, err := Search(ctx, db, low.Place())
	if err != nil || len(found) != 1 || len(found[0].TreadWarnings) != 2 {
		t.Fatal("search differs", found, err)
	}
	if err = ReleaseStorage(ctx, db, low.Area, low.Number, claim); err != nil {
		t.Fatal(err)
	}
	rows, err = TreadWarnings(ctx, db, true)
	if err != nil || len(rows) != 0 {
		t.Fatal("released still counted", rows, err)
	}
	rows, err = TreadWarnings(ctx, db, false)
	if err != nil || len(rows) != 2 || rows[0].Operation != "release" {
		t.Fatal("history omitted", rows, err)
	}
	healthy.Number = low.Number
	if _, err = CreateWheelChange(ctx, db, healthy); err != nil {
		t.Fatal(err)
	}
	rows, err = TreadWarnings(ctx, db, true)
	if err != nil || len(rows) != 0 {
		t.Fatal("history leaks into healthy replacement", rows, err)
	}
	low.Number = 5
	if _, err = CreateWheelChange(ctx, db, low); err != nil {
		t.Fatal(err)
	}
	rows, err = TreadWarnings(ctx, db, true)
	if err != nil || len(rows) != 1 || rows[0].Operation != "wheel_change" {
		t.Fatal(rows, err)
	}
}
