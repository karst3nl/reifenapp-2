package database

import (
	"context"
	"testing"
	"time"
)

func TestBrowseHistoryFiltersAndSnapshots(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	r := reservationInput()
	r.Name = "Chronik Fiktiv Müller"
	r.Plate = "MZ-CF 123"
	if _, err := CreateReservation(ctx, db, r); err != nil {
		t.Fatal(err)
	}
	s := storageExample()
	s.Name = r.Name
	s.Plate = r.Plate
	s.Brand = "Audi"
	s.Model = "A3"
	s.FrontDepth = "3.0"
	s.RearDepth = "4.0"
	id, err := CreateWheelChange(ctx, db, s)
	if err != nil {
		t.Fatal(err)
	}
	if err = ReleaseStorage(ctx, db, s.Area, s.Number, id); err != nil {
		t.Fatal(err)
	}
	s.Number = 3
	s.Name = "Anderer Fiktiv"
	s.FrontDepth = "6.5"
	s.RearDepth = "7.0"
	s.Notes = "Seitenwandprüfung fiktiv"
	if _, err = CreateStorage(ctx, db, s); err != nil {
		t.Fatal(err)
	}
	r.Number = 4
	if _, err = CreateReservation(ctx, db, r); err != nil {
		t.Fatal(err)
	}
	base := HistoryFilter{Page: 1, PageSize: 20}
	all, err := BrowseHistory(ctx, db, base)
	if err != nil || all.Total != 5 || len(all.Items) != 5 {
		t.Fatal(all, err)
	}
	if all.Operations["reservation"] != 2 || all.Operations["wheel_change"] != 1 || all.Operations["release"] != 1 || all.Operations["storage"] != 1 {
		t.Fatal(all.Operations)
	}
	for _, tc := range []struct {
		q, op, status, source string
		warnings              bool
		total                 int
	}{
		{"", "release", "", "", false, 1}, {"", "", "active", "", false, 2}, {"", "", "closed", "", false, 3},
		{"MZCF123", "", "", "", false, 5}, {"müller", "", "", "", false, 4}, {"C1.2", "", "", "", false, 3},
		{"Seitenwandprüfung", "", "", "", false, 1}, {"", "", "", "recorded", false, 5}, {"", "", "", "legacy", false, 0},
		{"", "", "", "", true, 2}, {"", "", "active", "", true, 0}, {"xyz", "", "", "", false, 0},
	} {
		f := base
		f.Query = tc.q
		f.Operation = tc.op
		f.Status = tc.status
		f.Source = tc.source
		f.WarningsOnly = tc.warnings
		got, err := BrowseHistory(ctx, db, f)
		if err != nil || got.Total != tc.total {
			t.Fatalf("%+v: total=%d err=%v", tc, got.Total, err)
		}
	}
	f := base
	f.Query = "C1.2"
	chain, err := BrowseHistory(ctx, db, f)
	if err != nil {
		t.Fatal(err)
	}
	if chain.Items[0].Operation != "release" || chain.Items[1].Car != "Audi A3" || chain.Items[2].Car != "VW Golf" || chain.Items[1].Details.FrontDepth != "3.0" {
		t.Fatal(chain)
	}
	stamp, err := time.Parse(time.RFC3339Nano, all.Items[0].Date)
	if err != nil {
		t.Fatal(err)
	}
	f = base
	f.From = stamp
	f.Until = stamp.Add(time.Nanosecond)
	got, err := BrowseHistory(ctx, db, f)
	if err != nil || got.Total != 1 {
		t.Fatal("date boundary", got, err)
	}
	f = base
	f.Until = stamp
	got, err = BrowseHistory(ctx, db, f)
	if err != nil || got.Total != 4 {
		t.Fatal("exclusive until", got.Total, err)
	}
}
func TestBrowseHistoryStablePaginationAndLegacy(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	s := storageExample()
	for n := 1; n <= 25; n++ {
		s.Number = n
		if _, err := CreateStorage(ctx, db, s); err != nil {
			t.Fatal(err)
		}
	}
	f := HistoryFilter{Page: 1, PageSize: 20}
	first, err := BrowseHistory(ctx, db, f)
	if err != nil || first.Total != 25 || first.Pages != 2 || len(first.Items) != 20 {
		t.Fatal(first, err)
	}
	s.Number = 26
	if _, err := CreateStorage(ctx, db, s); err != nil {
		t.Fatal(err)
	}
	f.Page = 2
	f.Snapshot = first.Snapshot
	second, err := BrowseHistory(ctx, db, f)
	if err != nil || second.Total != 25 || len(second.Items) != 5 {
		t.Fatal(second, err)
	}
	seen := map[int64]bool{}
	for _, item := range first.Items {
		seen[item.EventID] = true
	}
	for _, item := range second.Items {
		if seen[item.EventID] {
			t.Fatal("duplicate across pages")
		}
		seen[item.EventID] = true
	}
	f.Page = 1
	f.Snapshot = 0
	fresh, err := BrowseHistory(ctx, db, f)
	if err != nil || fresh.Total != 26 || fresh.Items[0].EventID <= first.Snapshot {
		t.Fatal(fresh, err)
	}
	f.Page = 999
	empty, err := BrowseHistory(ctx, db, f)
	if err != nil || len(empty.Items) != 0 || empty.Total != 26 {
		t.Fatal(empty, err)
	}
	if _, err = BrowseHistory(ctx, db, HistoryFilter{}); err == nil {
		t.Fatal("invalid page accepted")
	}
	if _, err = db.Exec("INSERT INTO history_events(claim_id,operation,occurred_at,place,name,plate,car,payload,source) SELECT id,'release','2020-01-01T12:00:00Z',place_id,'Altbestand Fiktiv','','','{}','legacy' FROM storage_claims LIMIT 1"); err != nil {
		t.Fatal(err)
	}
	f = HistoryFilter{Page: 1, PageSize: 20, Source: "legacy"}
	legacy, err := BrowseHistory(ctx, db, f)
	if err != nil || legacy.Total != 1 || legacy.Items[0].Details.FrontDepth != "" || legacy.Items[0].Source != "legacy" {
		t.Fatal(legacy, err)
	}
}
