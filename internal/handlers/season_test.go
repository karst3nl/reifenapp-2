package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/karst3nl/reifenapp-2/internal/database"
)

func TestSeasonProgressHTTP(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "season.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	now, _ := time.Parse(time.RFC3339, "2027-01-10T12:00:00Z")
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/season", seasonProgress(db, func() time.Time { return now }))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/api/season", nil))
	var progress database.SeasonProgress
	if err = json.Unmarshal(w.Body.Bytes(), &progress); err != nil || w.Code != 200 || progress.Season.Label != "Winter 2026/27" || progress.Total != 0 {
		t.Fatal(w.Body, err)
	}
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("missing no-store")
	}
	if err = database.SeedDevelopment(db); err != nil {
		t.Fatal(err)
	}
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/api/season", nil))
	if err = json.Unmarshal(w.Body.Bytes(), &progress); err != nil || progress.Unknown != 2 || progress.Total != 0 {
		t.Fatal(w.Body, err)
	}
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("POST", "/api/season", nil))
	if w.Code != 405 {
		t.Fatal(w.Code)
	}
	db.Close()
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/api/season", nil))
	if w.Code != 500 || strings.Contains(w.Body.String(), "sqlite") {
		t.Fatal(w.Code, w.Body)
	}
}
