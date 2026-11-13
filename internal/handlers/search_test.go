package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/karst3nl/reifenapp-2/internal/database"
	"github.com/karst3nl/reifenapp-2/internal/models"
)

func TestSearchHTTP(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "api.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = database.SeedDevelopment(db); err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/search", Search(db))
	for _, tc := range []struct {
		method, q string
		status, n int
	}{{"GET", "C1.25", 200, 1}, {"GET", "no-match", 200, 0}, {"GET", "", 200, 0}, {"GET", strings.Repeat("x", 201), 400, 0}, {"POST", "x", 405, 0}} {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest(tc.method, "/api/search?q="+tc.q, nil))
		if w.Code != tc.status {
			t.Fatalf("status %d expected %d", w.Code, tc.status)
		}
		if tc.status == 200 {
			var rows []models.SearchResult
			if err = json.Unmarshal(w.Body.Bytes(), &rows); err != nil || rows == nil || len(rows) != tc.n {
				t.Fatalf("response %s: %v", w.Body, err)
			}
		}
	}
	db.Close()
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/api/search?q=test", nil))
	if w.Code != 500 || strings.Contains(w.Body.String(), "sqlite") {
		t.Fatal(w.Body.String())
	}
}

func TestSearchFindsReleaseHistory(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "history.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = database.SeedDevelopment(db); err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest("POST", "/api/releases", strings.NewReader(`{"area":"C1","number":25,"claim_id":1,"confirmed":true}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	Releases(db)(w, r)
	if w.Code != 200 {
		t.Fatal(w.Body)
	}
	for _, query := range []string{"TESTMM123", "C1.25", "Auslagerung"} {
		w = httptest.NewRecorder()
		Search(db)(w, httptest.NewRequest("GET", "/api/search?q="+query, nil))
		var rows []models.SearchResult
		if err = json.Unmarshal(w.Body.Bytes(), &rows); err != nil || len(rows) == 0 || rows[0].Operation != "release" || rows[0].Active || rows[0].EventID <= 0 {
			t.Fatal(w.Body, err)
		}
	}
}

func TestSearchHTTPIncludesDetails(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "details.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	input := models.StorageInput{Area: "C1", Number: 7, Name: "Details Fiktiv", Plate: "MZ-DF 123", Brand: "VW", Model: "Golf", TireType: "Sommer", RimType: "Aluminium", Front: models.TireSize{Width: "205", Ratio: "55", Inches: "16", Load: "91", Speed: "H"}, Rear: models.TireSize{Width: "225", Ratio: "45", Inches: "17", Load: "94", Speed: "V"}, FrontDepth: "3.0", RearDepth: "4.5", Notes: "Fiktiver Prüfhinweis", Mechanic: "Fiktiver Monteur", Lock: models.Accessory{Present: true, Location: "Am Radsatz"}}
	if _, err = database.CreateStorage(context.Background(), db, input); err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	Search(db)(w, httptest.NewRequest("GET", "/api/search?q=C1.7", nil))
	var rows []models.SearchResult
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
	if err = json.Unmarshal(w.Body.Bytes(), &rows); err != nil || len(rows) != 1 {
		t.Fatal(err, w.Body)
	}
	d := rows[0].Details
	if d.Notes != input.Notes || d.FrontDepth != "3.0" || d.RearDepth != "4.5" || d.Rear != input.Rear || d.Accessories["lock"] != input.Lock || d.Mechanic != input.Mechanic {
		t.Fatal("API omitted details", d)
	}
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("customer details must not be cached")
	}
}
