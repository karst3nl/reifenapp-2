package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/karst3nl/reifenapp-2/internal/database"
)

func TestHealthDatabaseAndVersion(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "health.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	handler := Health(db, "fictional-commit")
	w := httptest.NewRecorder()
	handler(w, httptest.NewRequest(http.MethodGet, "/api/health", nil))
	var response map[string]string
	if err = json.Unmarshal(w.Body.Bytes(), &response); err != nil || w.Code != 200 || response["status"] != "ok" || response["version"] != "fictional-commit" || len(response) != 2 || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal(w.Code, w.Body, err)
	}
	db.Close()
	w = httptest.NewRecorder()
	handler(w, httptest.NewRequest(http.MethodGet, "/api/health", nil))
	if w.Code != 503 {
		t.Fatal(w.Code, w.Body)
	}
}
