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

func TestWheelChangesHTTP(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "test.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/wheel-changes", WheelChanges(db))
	call := func(body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "/api/wheel-changes", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		return w
	}
	if w := call("{}"); w.Code != 422 {
		t.Fatal(w.Body)
	}
	if w := call(`{"operation":"storage"}`); w.Code != 400 {
		t.Fatal(w.Body)
	}
	input := models.StorageInput{Area: "C1", Number: 2, Name: "Radwechsel API Fiktiv", Plate: "MZ-RW 123", Brand: "VW", Model: "Golf", TireType: "Winter", RimType: "Stahl", Front: models.TireSize{Width: "205", Ratio: "55", Inches: "16", Load: "91", Speed: "H"}, Rear: models.TireSize{Width: "205", Ratio: "55", Inches: "16", Load: "91", Speed: "H"}, FrontDepth: "6.5", RearDepth: "6.0"}
	body, _ := json.Marshal(input)
	w := call(string(body))
	if w.Code != 201 {
		t.Fatal(w.Body)
	}
	var result struct {
		Operation string `json:"operation"`
	}
	if err = json.Unmarshal(w.Body.Bytes(), &result); err != nil || result.Operation != "wheel_change" {
		t.Fatal(w.Body, err)
	}
	if w = call(string(body)); w.Code != 409 {
		t.Fatal(w.Body)
	}
	input.TireType = "Sommer"
	body, _ = json.Marshal(input)
	w = call(string(body))
	if w.Code != 201 {
		t.Fatal("occupied own place must swap", w.Code, w.Body)
	}
	var count int
	if err = db.QueryRow("SELECT count(*) FROM storage_claims WHERE status='active' AND place_id='C1.2'").Scan(&count); err != nil || count != 1 {
		t.Fatal(count, err)
	}
	w = call(string(body))
	if err = db.QueryRow("SELECT count(*) FROM history_events WHERE operation='release'").Scan(&count); err != nil || count != 0 {
		t.Fatal("swap generated release event", count, err)
	}
	if err = db.QueryRow("SELECT count(*) FROM storage_releases").Scan(&count); err != nil || count != 0 {
		t.Fatal("swap generated release log", count, err)
	}
	var duplicate struct{ Fields map[string]string }
	if err = json.Unmarshal(w.Body.Bytes(), &duplicate); err != nil || w.Code != 409 || duplicate.Fields["tire_type"] == "" {
		t.Fatal(w.Code, w.Body, err)
	}
	input.Name = "Fremder Kunde (fiktiv)"
	input.TireType = "Winter"
	body, _ = json.Marshal(input)
	w = call(string(body))
	var owner struct{ Fields map[string]string }
	if err = json.Unmarshal(w.Body.Bytes(), &owner); err != nil || w.Code != 409 || owner.Fields["name"] == "" || owner.Fields["plate"] == "" {
		t.Fatal(w.Code, w.Body, err)
	}
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/api/wheel-changes", nil))
	if w.Code != 405 {
		t.Fatal(w.Code)
	}
	db.Close()
	if w = call(string(body)); w.Code != 500 {
		t.Fatal(w.Body)
	}
}
