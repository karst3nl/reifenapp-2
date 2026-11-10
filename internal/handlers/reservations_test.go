package handlers

import (
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/karst3nl/reifenapp-2/internal/database"
)

func TestReservationsHTTP(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "test.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	call := func(body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "/api/reservations", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		Reservations(db)(w, r)
		return w
	}
	valid := `{"area":"C1","number":2,"name":"API Reservierung Fiktiv","plate":"MZ-AR 123","brand":"VW","model":"Golf","tire_type":"Winter","rim_type":"Stahl"}`
	for _, tc := range []struct {
		body   string
		status int
	}{{"{", 400}, {"{}", 422}, {strings.TrimSuffix(valid, "}") + `,"front":{}}`, 400}, {valid, 201}, {valid, 409}} {
		w := call(tc.body)
		if w.Code != tc.status {
			t.Fatalf("%d expected %d: %s", w.Code, tc.status, w.Body)
		}
		if !strings.HasPrefix(w.Header().Get("Content-Type"), "application/json") {
			t.Fatal("not JSON")
		}
	}
	body := `{"area":"C1","number":2,"name":"API Reservierung Fiktiv","plate":"MZ-AR 123","purpose":"reservation"}`
	r := httptest.NewRequest("POST", "/api/storage/check", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	CheckStorage(db)(w, r)
	var checked database.PlaceCheck
	if err = json.Unmarshal(w.Body.Bytes(), &checked); err != nil || checked.Allowed || checked.Status != "reserved" {
		t.Fatal(w.Body, err)
	}
	r = httptest.NewRequest("POST", "/api/storage/check", strings.NewReader(strings.Replace(body, "reservation", "storage", 1)))
	r.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	CheckStorage(db)(w, r)
	if err = json.Unmarshal(w.Body.Bytes(), &checked); err != nil || !checked.Allowed {
		t.Fatal(w.Body, err)
	}
	db.Close()
	if w = call(valid); w.Code != 500 {
		t.Fatal(w.Code)
	}
}
