package handlers

import (
	"encoding/json"
	"github.com/karst3nl/reifenapp-2/internal/database"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestCapacityHTTP(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "capacity.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/capacity", Capacity(db))
	get := func() *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest("GET", "/api/capacity", nil))
		return w
	}
	w := get()
	var result database.Capacity
	if err = json.Unmarshal(w.Body.Bytes(), &result); err != nil || w.Code != 200 || result.Free != 640 || result.Reserved != 0 || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal(w.Code, w.Body, err)
	}
	db.Close()
	if w = get(); w.Code != 500 {
		t.Fatal(w.Code, w.Body)
	}
}
