package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/karst3nl/reifenapp-2/internal/database"
)

func TestHistoryHTTP(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "history.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/history", History(db))
	call := func(path string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		return w
	}
	w := call("/api/history")
	if w.Code != 200 || !strings.Contains(w.Body.String(), "\"items\":[]") {
		t.Fatal(w.Code, w.Body)
	}
	if err = database.SeedDevelopment(db); err != nil {
		t.Fatal(err)
	}
	w = call("/api/history")
	var page database.HistoryPage
	if err = json.Unmarshal(w.Body.Bytes(), &page); err != nil || page.Total != 2 || page.PageSize != 20 || len(page.Items) != 2 || page.Items[0].ReceiptURL == "" {
		t.Fatal(w.Body, err)
	}
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("missing no-store")
	}
	for _, query := range []string{"page=0", "page=-1", "page=x", "page=1000001", "page_size=21", "page_size=0", "operation=bad", "status=bad", "source=bad", "warnings=bad", "snapshot=-1", "snapshot=x", "from=2026-02-30", "until=bad", "from=2026-11-14T00:00:00Z&until=2026-11-13T00:00:00Z", "q=" + strings.Repeat("x", 201)} {
		w = call("/api/history?" + query)
		if w.Code != 400 {
			t.Fatal(query, w.Code, w.Body)
		}
	}
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("POST", "/api/history", nil))
	if w.Code != 405 {
		t.Fatal(w.Code)
	}
	db.Close()
	w = call("/api/history")
	if w.Code != 500 || strings.Contains(w.Body.String(), "sqlite") {
		t.Fatal(w.Code, w.Body)
	}
}
