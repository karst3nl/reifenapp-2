package handlers

import (
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/karst3nl/reifenapp-2/internal/database"
)

func TestReleaseHTTP(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "test.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = database.SeedDevelopment(db); err != nil {
		t.Fatal(err)
	}
	call := func(body string, lookup bool) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "/api/releases", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		if lookup {
			CheckRelease(db)(w, r)
		} else {
			Releases(db)(w, r)
		}
		return w
	}
	for _, tc := range []struct {
		body   string
		status int
	}{{`{"area":"C1","number":25}`, 200}, {`{"area":"C1","number":2}`, 200}, {`{"area":"C2","number":2}`, 422}, {`{`, 400}} {
		w := call(tc.body, true)
		if w.Code != tc.status {
			t.Fatal(w.Code, w.Body)
		}
		if w.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("cache")
		}
	}
	w := call(`{"area":"C1","number":25}`, true)
	var target database.ReleaseTarget
	if err = json.Unmarshal(w.Body.Bytes(), &target); err != nil || target.ClaimID != 1 || target.Name != "Max Mustermann (fiktiv)" {
		t.Fatal(w.Body, err)
	}
	for _, tc := range []struct {
		body string
		code int
	}{{`{"area":"C1","number":25,"claim_id":1}`, 422}, {`{"area":"C1","number":25,"claim_id":2,"confirmed":true}`, 409}, {`{"area":"C2","number":2,"claim_id":1,"confirmed":true}`, 422}, {`{"area":"C1","number":25,"claim_id":1,"confirmed":true}`, 200}, {`{"area":"C1","number":25,"claim_id":1,"confirmed":true}`, 409}} {
		if w = call(tc.body, false); w.Code != tc.code {
			t.Fatal(w.Code, w.Body)
		}
	}
	w = call(`{"area":"C1","number":25}`, true)
	if !strings.Contains(w.Body.String(), `"status":"free"`) {
		t.Fatal(w.Body)
	}
	db.Close()
	if w = call(`{"area":"C1","number":25}`, true); w.Code != 500 {
		t.Fatal(w.Body)
	}
}
