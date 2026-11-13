package handlers

import (
	"context"
	"encoding/json"
	"github.com/karst3nl/reifenapp-2/internal/database"
	"github.com/karst3nl/reifenapp-2/internal/models"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestTreadWarningsHTTP(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "warnings.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	handler := TreadWarnings(db)
	w := httptest.NewRecorder()
	handler(w, httptest.NewRequest("GET", "/api/warnings", nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"items":[]`) {
		t.Fatal(w.Code, w.Body)
	}
	input := models.StorageInput{Area: "C1", Number: 7, Name: "Warnung Fiktiv", Plate: "MZ-WF 123", Brand: "VW", Model: "Golf", TireType: "Sommer", RimType: "Stahl", Front: models.TireSize{Width: "195", Ratio: "65", Inches: "15", Load: "91", Speed: "T"}, Rear: models.TireSize{Width: "195", Ratio: "65", Inches: "15", Load: "91", Speed: "T"}, FrontDepth: "3.0", RearDepth: "0.0"}
	if _, err = database.CreateStorage(context.Background(), db, input); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/api/warnings", "/api/warnings?scope=active", "/api/warnings?scope=all"} {
		w = httptest.NewRecorder()
		handler(w, httptest.NewRequest("GET", path, nil))
		var response struct {
			Count int                   `json:"count"`
			Axles int                   `json:"axle_count"`
			Items []models.SearchResult `json:"items"`
		}
		if err = json.Unmarshal(w.Body.Bytes(), &response); err != nil || w.Code != 200 || response.Count != 1 || response.Axles != 2 || len(response.Items) != 1 || response.Items[0].Details.FrontDepth != "3.0" {
			t.Fatal(w.Body, err)
		}
		if w.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("missing no-store")
		}
	}
	w = httptest.NewRecorder()
	handler(w, httptest.NewRequest("GET", "/api/warnings?scope=bad", nil))
	if w.Code != 400 {
		t.Fatal(w.Code)
	}
	db.Close()
	w = httptest.NewRecorder()
	handler(w, httptest.NewRequest("GET", "/api/warnings", nil))
	if w.Code != 500 || strings.Contains(w.Body.String(), "sqlite") {
		t.Fatal(w.Code, w.Body)
	}
}
