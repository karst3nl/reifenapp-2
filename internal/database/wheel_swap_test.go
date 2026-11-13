package database

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/karst3nl/reifenapp-2/internal/models"
)

func TestWheelSwapKeepsPlaceAndReusesSeasonalSets(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	input := storageExample()
	first, err := CreateStorage(ctx, db, input)
	if err != nil {
		t.Fatal(err)
	}
	var winterID int64
	if err = db.QueryRow("SELECT tire_set_id FROM storage_claims WHERE id=?", first).Scan(&winterID); err != nil {
		t.Fatal(err)
	}
	input.TireType = "Sommer"
	input.Brand = "Audi"
	input.Model = "A3"
	input.FrontDepth = "3.0"
	input.Notes = "Sommerdaten (fiktiv)"
	second, err := CreateWheelChange(ctx, db, input)
	if err != nil {
		t.Fatal(err)
	}
	var summerID int64
	if err = db.QueryRow("SELECT tire_set_id FROM storage_claims WHERE id=?", second).Scan(&summerID); err != nil || summerID == winterID {
		t.Fatal(summerID, err)
	}
	input.TireType = "Winter"
	input.FrontDepth = "7.0"
	input.Notes = "Aktualisierte Winterdaten (fiktiv)"
	third, err := CreateWheelChange(ctx, db, input)
	if err != nil {
		t.Fatal(err)
	}
	var returned int64
	if err = db.QueryRow("SELECT tire_set_id FROM storage_claims WHERE id=?", third).Scan(&returned); err != nil || returned != winterID {
		t.Fatal("winter identity lost", returned, err)
	}
	input.TireType = "Sommer"
	fourth, err := CreateWheelChange(ctx, db, input)
	if err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow("SELECT tire_set_id FROM storage_claims WHERE id=?", fourth).Scan(&returned); err != nil || returned != summerID {
		t.Fatal("summer identity lost", returned, err)
	}
	for table, want := range map[string]int{"customers": 1, "vehicles": 1, "tire_sets": 2, "storage_claims": 4, "history_events": 4, "storage_releases": 0} {
		var count int
		if err = db.QueryRow("SELECT count(*) FROM " + table).Scan(&count); err != nil || count != want {
			t.Fatal(table, count, err)
		}
	}
	var active int
	if err = db.QueryRow("SELECT count(*) FROM storage_claims WHERE place_id=? AND status='active'", input.Place()).Scan(&active); err != nil || active != 1 {
		t.Fatal(active, err)
	}
	rows, err := Search(ctx, db, input.Place())
	if err != nil || len(rows) != 4 || !rows[0].Active || rows[0].Operation != "wheel_change" {
		t.Fatal(rows, err)
	}
	if _, err := Receipt(ctx, db, first, "release"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatal("wheel swap created release receipt", err)
	}
	for _, row := range rows {
		if row.Operation == "release" {
			t.Fatal("wheel swap created Auslagerung history")
		}
	}
	original, err := Receipt(ctx, db, first, "storage")
	if err != nil || original.Car != "VW Golf" {
		t.Fatal(original, err)
	}
	if rows[len(rows)-1].Details.FrontDepth != "6.5" || rows[len(rows)-1].Details.TireType != "Winter" {
		t.Fatal("original tire details changed")
	}
	if err = ReleaseStorage(ctx, db, input.Area, input.Number, fourth); err != nil {
		t.Fatal(err)
	}
	var releases int
	if err = db.QueryRow("SELECT count(*) FROM storage_releases").Scan(&releases); err != nil || releases != 1 {
		t.Fatal("explicit release log", releases, err)
	}
	finalRows, err := Search(ctx, db, input.Place())
	if err != nil || len(finalRows) != 5 || finalRows[0].Operation != "release" {
		t.Fatal("explicit release history", finalRows, err)
	}
}

func TestWheelSwapRejectsForeignIdentityAndDuplicateType(t *testing.T) {
	ctx := context.Background()
	db := testDB(t)
	input := storageExample()
	first, err := CreateStorage(ctx, db, input)
	if err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*models.StorageInput){func(s *models.StorageInput) { s.Name = "Anderer Kunde (fiktiv)" }, func(s *models.StorageInput) { s.Plate = "MZ-XY 123" }} {
		other := input
		other.TireType = "Sommer"
		mutate(&other)
		if _, err = CreateWheelChange(ctx, db, other); !errors.Is(err, ErrWheelChangeOwner) {
			t.Fatal(err)
		}
	}
	if _, err = CreateWheelChange(ctx, db, input); !errors.Is(err, ErrWheelChangeSameType) {
		t.Fatal(err)
	}
	if _, err = CreateStorage(ctx, db, input); !errors.Is(err, ErrPlaceUnavailable) {
		t.Fatal("normal storage bypasses protection", err)
	}
	target, err := LookupRelease(ctx, db, input.Area, input.Number)
	if err != nil || target.ClaimID != first {
		t.Fatal("old occupancy changed", target, err)
	}
	// Existing identity normalization also applies to occupied places.
	input.Name = " fIkTiV Neueinlagerung "
	input.Plate = "mz-ne 987"
	input.TireType = "Sommer"
	if _, err = CreateWheelChange(ctx, db, input); err != nil {
		t.Fatal("normalized identity", err)
	}
	reservation := reservationInput()
	reservation.Number = 8
	if _, err = CreateReservation(ctx, db, reservation); err != nil {
		t.Fatal(err)
	}
	input.Number = 8
	input.Name = "Fremde Reservierung (fiktiv)"
	if _, err = CreateWheelChange(ctx, db, input); !errors.Is(err, ErrPlaceUnavailable) {
		t.Fatal("foreign reservation consumed", err)
	}
}

func TestWheelSwapFailureRollsBackReplacement(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	input := storageExample()
	first, err := CreateStorage(ctx, db, input)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec("CREATE TRIGGER fail_swap BEFORE INSERT ON history_events WHEN NEW.operation='wheel_change' BEGIN SELECT RAISE(ABORT,'fictitious failure'); END"); err != nil {
		t.Fatal(err)
	}
	input.TireType = "Sommer"
	if _, err = CreateWheelChange(ctx, db, input); err == nil {
		t.Fatal("expected failure")
	}
	for table, want := range map[string]int{"tire_sets": 1, "storage_claims": 1, "storage_releases": 0, "history_events": 1} {
		var count int
		if err = db.QueryRow("SELECT count(*) FROM " + table).Scan(&count); err != nil || count != want {
			t.Fatal(table, count, err)
		}
	}
	target, err := LookupRelease(ctx, db, input.Area, input.Number)
	if err != nil || target.ClaimID != first || target.Status != "occupied" {
		t.Fatal(target, err)
	}
}

func TestConcurrentWheelSwapsPreserveOneOccupancy(t *testing.T) {
	path := filepath.Join(t.TempDir(), "swap.sqlite")
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
	if _, err = CreateStorage(ctx, a, input); err != nil {
		t.Fatal(err)
	}
	input.TireType = "Sommer"
	start := make(chan struct{})
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for _, create := range []func() error{func() error { _, e := CreateWheelChange(ctx, a, input); return e }, func() error { _, e := CreateWheelChange(ctx, b, input); return e }} {
		wg.Add(1)
		go func(fn func() error) { defer wg.Done(); <-start; results <- fn() }(create)
	}
	close(start)
	wg.Wait()
	close(results)
	success, conflict := 0, 0
	for err := range results {
		if err == nil {
			success++
		} else if errors.Is(err, ErrWheelChangeSameType) {
			conflict++
		} else {
			t.Fatal(err)
		}
	}
	if success != 1 || conflict != 1 {
		t.Fatal(success, conflict)
	}
	var count int
	if err = a.QueryRow("SELECT count(*) FROM storage_claims WHERE status='active'").Scan(&count); err != nil || count != 1 {
		t.Fatal(count, err)
	}
	if err = a.QueryRow("SELECT count(*) FROM history_events").Scan(&count); err != nil || count != 2 {
		t.Fatal(count, err)
	}
}

func TestWheelSwapUpdatesSeasonCounterWithoutGrowingStock(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	input := storageExample()
	season := models.CurrentSeason(time.Now())
	input.TireType = season.MountedType
	if _, err := CreateStorage(ctx, db, input); err != nil {
		t.Fatal(err)
	}
	before, err := CurrentSeasonProgress(ctx, db, time.Now())
	if err != nil || before.Open != 1 || before.Total != 1 {
		t.Fatal(before, err)
	}
	input.TireType = season.StoredType
	if _, err = CreateWheelChange(ctx, db, input); err != nil {
		t.Fatal(err)
	}
	after, err := CurrentSeasonProgress(ctx, db, time.Now())
	if err != nil || after.Completed != 1 || after.Open != 0 || after.Total != 1 || after.Active != 1 {
		t.Fatal(after, err)
	}
}
