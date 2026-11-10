package handlers

import (
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
