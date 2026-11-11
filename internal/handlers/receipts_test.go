package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/karst3nl/reifenapp-2/internal/database"
	"github.com/karst3nl/reifenapp-2/internal/models"
)

func TestFormReceiptURLsAndPDFs(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "receipts.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/storage", Storage(db))
	mux.HandleFunc("POST /api/wheel-changes", WheelChanges(db))
	mux.HandleFunc("POST /api/reservations", Reservations(db))
	mux.HandleFunc("POST /api/releases", Releases(db))
	mux.HandleFunc("GET /api/receipts/{claim}/{operation}", Receipts(db))
	input := models.StorageInput{Area: "C1", Number: 2, Name: "PDF Fiktiv Müller", Plate: "MZ-PF 123", Brand: "VW", Model: "Golf", TireType: "Winter", RimType: "Stahl", Front: models.TireSize{Width: "205", Ratio: "55", Inches: "16", Load: "91", Speed: "H"}, Rear: models.TireSize{Width: "205", Ratio: "55", Inches: "16", Load: "91", Speed: "H"}, FrontDepth: "6.5", RearDepth: "6.0"}
	urls := []string{}
	post := func(path string, body []byte) map[string]any {
		r := httptest.NewRequest("POST", path, bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code != 200 && w.Code != 201 {
			t.Fatal(w.Code, w.Body)
		}
		var data map[string]any
		if err = json.Unmarshal(w.Body.Bytes(), &data); err != nil {
			t.Fatal(err)
		}
		url, ok := data["pdf_url"].(string)
		if !ok || url == "" {
			t.Fatal("no PDF URL", data)
		}
		urls = append(urls, url)
		return data
	}
	body, _ := json.Marshal(input)
	stored := post("/api/storage", body)
	released, _ := json.Marshal(map[string]any{"area": "C1", "number": 2, "claim_id": stored["id"], "confirmed": true})
	post("/api/releases", released)
	input.Number = 3
	body, _ = json.Marshal(input)
	post("/api/wheel-changes", body)
	reservation := models.ReservationInput{Area: "C1", Number: 4, Name: input.Name, Plate: input.Plate, Brand: "VW", Model: "Golf"}
	body, _ = json.Marshal(reservation)
	post("/api/reservations", body)
	for _, url := range urls {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest("GET", url, nil))
		if w.Code != 200 || w.Header().Get("Content-Type") != "application/pdf" || !strings.HasPrefix(w.Header().Get("Content-Disposition"), "inline;") || !strings.Contains(w.Header().Get("Cache-Control"), "no-store") || !bytes.HasPrefix(w.Body.Bytes(), []byte("%PDF-")) {
			t.Fatalf("%s %d %s", url, w.Code, w.Body)
		}
	}
	for _, tc := range []struct {
		url  string
		code int
	}{{"/api/receipts/999/storage.pdf", 404}, {"/api/receipts/1/unknown.pdf", 400}, {"/api/receipts/0/storage.pdf", 400}, {"/api/receipts/1/storage", 400}} {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest("GET", tc.url, nil))
		if w.Code != tc.code {
			t.Fatal(tc.url, w.Code)
		}
	}
	db.Close()
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", urls[0], nil))
	if w.Code != 500 {
		t.Fatal(w.Code)
	}
}
