package database

import (
	"context"
	"errors"
	"testing"
)

func TestAccessoryOptionsValidationRevisionAndHistory(t *testing.T) {
	ctx := context.Background()
	db := testDB(t)
	options, err := LoadAccessoryOptions(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	original := storageExample()
	id, err := CreateStorage(ctx, db, original)
	if err != nil {
		t.Fatal(err)
	}
	options.Locations["hubcaps"] = []string{"Neue Testbox (fiktiv)"}
	if err = SaveAccessoryOptions(ctx, db, options); err != nil {
		t.Fatal(err)
	}
	if err = SaveAccessoryOptions(ctx, db, options); !errors.Is(err, ErrAccessoryRevision) {
		t.Fatal("stale options accepted", err)
	}
	historical, err := Receipt(ctx, db, id, "storage")
	if err != nil || string(historical.Payload) == "" {
		t.Fatal(err)
	}
	rows, err := Search(ctx, db, original.Name)
	if err != nil || rows[0].Details.Accessories["hubcaps"].Location != "Am Radsatz" {
		t.Fatal("history rewritten", rows, err)
	}
	next := storageExample()
	next.Number = 3
	_, err = CreateStorage(ctx, db, next)
	var fields *AccessoryFieldsError
	if !errors.As(err, &fields) || fields.Fields["hubcaps.location"] == "" {
		t.Fatal("removed dropdown value accepted", err)
	}
	var count int
	db.QueryRow("SELECT count(*) FROM storage_claims").Scan(&count)
	if count != 1 {
		t.Fatal("invalid input changed database")
	}
	next.Hubcaps.Location = "Neue Testbox (fiktiv)"
	if _, err = CreateStorage(ctx, db, next); err != nil {
		t.Fatal(err)
	}
	for _, values := range [][]string{nil, {" "}, {"Box", "box"}, {"Box\nOrt"}} {
		c, err := LoadAccessoryOptions(ctx, db)
		if err != nil {
			t.Fatal(err)
		}
		c.Locations["bolts"] = values
		if err = SaveAccessoryOptions(ctx, db, c); !errors.Is(err, ErrAccessoryOptions) {
			t.Fatal("invalid options accepted", values, err)
		}
	}
	backup, err := CreateBackup(ctx, db, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer backup.Close()
	verifyBackupExports(t, backup)
}
