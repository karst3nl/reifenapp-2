package database

import (
	"context"
	"testing"
)

func TestLabelOutboxFollowsCommittedOperations(t *testing.T) {
	ctx := context.Background()
	db := testDB(t)
	input := storageExample()
	input.Number = 20
	if _, err := CreateStorage(ctx, db, input); err != nil {
		t.Fatal(err)
	}
	jobs, err := LabelJobs(ctx, db)
	if err != nil || len(jobs) != 0 {
		t.Fatal("disabled printer enqueued", err)
	}
	c, err := LabelConfiguration(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	c.Settings.Enabled = true
	c.Settings.Queue = "Argox_USB"
	c.Settings.Copies = 4
	if err = SaveLabelConfiguration(ctx, db, c); err != nil {
		t.Fatal(err)
	}
	input.Number = 21
	claim, err := CreateStorage(ctx, db, input)
	if err != nil {
		t.Fatal(err)
	}
	id, err := LabelJobForClaim(ctx, db, claim, "storage")
	if err != nil || id < 1 {
		t.Fatal(id, err)
	}
	j, err := LabelJobStatus(ctx, db, id)
	if err != nil || j.Status != "pending" || j.Settings.Copies != 4 || j.Receipt.Name != input.Name || j.EventID == 0 {
		t.Fatal(j, err)
	}
	if _, err = CreateStorage(ctx, db, input); err == nil {
		t.Fatal("duplicate storage accepted")
	}
	jobs, err = LabelJobs(ctx, db)
	if err != nil || len(jobs) != 1 {
		t.Fatal("failed booking enqueued", err)
	}
	input.TireType = "Sommer"
	if _, err = CreateWheelChange(ctx, db, input); err != nil {
		t.Fatal(err)
	}
	jobs, err = LabelJobs(ctx, db)
	if err != nil || len(jobs) != 2 || jobs[0].Receipt.Operation != "wheel_change" {
		t.Fatal(jobs, err)
	}
	// Immutable snapshots survive later template edits.
	c, err = LabelConfiguration(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	c.Settings.Copies = 2
	if err = SaveLabelConfiguration(ctx, db, c); err != nil {
		t.Fatal(err)
	}
	j, err = LabelJobStatus(ctx, db, id)
	if err != nil || j.Settings.Copies != 4 {
		t.Fatal("snapshot changed", err)
	}
	claimed, err := ClaimLabelJob(ctx, db)
	if err != nil || claimed.ID != id || claimed.Attempts != 1 {
		t.Fatal(claimed, err)
	}
	if err = RecoverLabelJobs(ctx, db); err != nil {
		t.Fatal(err)
	}
	j, err = LabelJobStatus(ctx, db, id)
	if err != nil || j.Status != "uncertain" {
		t.Fatal(j, err)
	}
	if err = UpdateLabelJob(ctx, db, id, true); err != nil {
		t.Fatal(err)
	}
	j, err = LabelJobStatus(ctx, db, id)
	if err != nil || j.Settings.Copies != 2 || j.Status != "pending" {
		t.Fatal("retry must use saved config", j, err)
	}
	if err = UpdateLabelJob(ctx, db, id, false); err != nil {
		t.Fatal(err)
	}
	c, err = LabelConfiguration(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	c.Settings.Storage = false
	c.Settings.WheelChange = false
	if err = SaveLabelConfiguration(ctx, db, c); err != nil {
		t.Fatal(err)
	}
	input.Number = 22
	if _, err = CreateStorage(ctx, db, input); err != nil {
		t.Fatal(err)
	}
	jobs, err = LabelJobs(ctx, db)
	if err != nil || len(jobs) != 2 {
		t.Fatal("disabled operation enqueued", err)
	}
	if err = SaveLabelConfiguration(ctx, db, c); err != ErrLabelRevision {
		t.Fatal("revision bypass", err)
	}
	// Reservation never enters the label outbox.
	reservation := reservationInput()
	reservation.Number = 23
	if _, err = CreateReservation(ctx, db, reservation); err != nil {
		t.Fatal(err)
	}
	jobs, err = LabelJobs(ctx, db)
	if err != nil || len(jobs) != 2 {
		t.Fatal("reservation printed", err)
	}
	backup, err := CreateBackup(ctx, db, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer backup.Close()
	verifyBackupExports(t, backup)
}
func TestLabelQueueFailureRollsBackStorage(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	c, err := LabelConfiguration(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	c.Settings.Enabled = true
	c.Settings.Queue = "Argox_USB"
	if err = SaveLabelConfiguration(ctx, db, c); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec("CREATE TRIGGER fail_label BEFORE INSERT ON label_print_jobs BEGIN SELECT RAISE(ABORT,'simulated queue failure'); END"); err != nil {
		t.Fatal(err)
	}
	input := storageExample()
	if _, err = CreateStorage(ctx, db, input); err == nil {
		t.Fatal("expected outbox failure")
	}
	for _, table := range []string{"history_events", "storage_claims", "label_print_jobs"} {
		var n int
		if err = db.QueryRow("SELECT count(*) FROM " + table).Scan(&n); err != nil || n != 0 {
			t.Fatal(table, n, err)
		}
	}
}
