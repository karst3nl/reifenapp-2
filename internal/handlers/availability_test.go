package handlers

import (
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/karst3nl/reifenapp-2/internal/database"
)

func TestCheckStorageHTTP(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "test.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = database.SeedDevelopment(db); err != nil {
		t.Fatal(err)
	}
	call := func(body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "/api/storage/check", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		CheckStorage(db)(w, r)
		return w
	}
	for _, tc := range []struct {
		body, status string
		code         int
	}{{`{"area":"C1","number":25}`, "occupied", 200}, {`{"area":"C1","number":2}`, "free", 200}, {`{"area":"C2","number":2}`, "", 422}, {`{"area":"C1","number":2,"unexpected":true}`, "", 400}, {`{`, "", 400}} {
		w := call(tc.body)
		if w.Code != tc.code {
			t.Fatalf("%d %s", w.Code, w.Body)
		}
		if tc.code == 200 {
			var c database.PlaceCheck
			if err = json.Unmarshal(w.Body.Bytes(), &c); err != nil || c.Status != tc.status {
				t.Fatal(w.Body)
			}
			if strings.Contains(w.Body.String(), "Mustermann") {
				t.Fatal("owner disclosed")
			}
		}
		if w.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("cache")
		}
	}
	db.Close()
	if w := call(`{"area":"C1","number":2}`); w.Code != 500 {
		t.Fatal(w.Body)
	}
}
