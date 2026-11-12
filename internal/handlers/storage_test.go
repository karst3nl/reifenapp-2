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

func TestStorageHTTP(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "api.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/storage", Storage(db))
	mux.HandleFunc("GET /api/search", Search(db))
	mux.HandleFunc("GET /api/storage/options", StorageOptions(db))
	call := func(body, media, origin string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "/api/storage", strings.NewReader(body))
		r.Header.Set("Content-Type", media)
		r.Header.Set("Origin", origin)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if !strings.HasPrefix(w.Header().Get("Content-Type"), "application/json") {
			t.Fatal("not JSON")
		}
		return w
	}
	for _, tc := range []struct {
		body, media, origin string
		code                int
	}{{"{}", "application/json", "", 422}, {"{", "application/json", "", 400}, {"{} {}", "application/json", "", 400}, {`{"unexpected":1}`, "application/json", "", 400}, {"{}", "text/plain", "", 415}, {"{}", "application/json", "http://evil.invalid", 403}, {strings.Repeat("x", 33000), "application/json", "", 400}} {
		if w := call(tc.body, tc.media, tc.origin); w.Code != tc.code {
			t.Fatalf("%d expected %d: %s", w.Code, tc.code, w.Body)
		}
	}
	input := models.StorageInput{Area: "C1", Number: 2, Name: "Fiktiver API Test", Plate: "MZ-AP 1", Brand: "VW", Model: "Golf", TireType: "Winter", RimType: "Stahl", Front: models.TireSize{Width: "205", Ratio: "55", Inches: "16", Load: "91", Speed: "H"}, Rear: models.TireSize{Width: "205", Ratio: "55", Inches: "16", Load: "91", Speed: "H"}, FrontDepth: "6.5", RearDepth: "6.0"}
	body, _ := json.Marshal(input)
	if w := call(string(body), "application/json", ""); w.Code != 201 {
		t.Fatal(w.Body)
	}
	if w := call(string(body), "application/json", ""); w.Code != 409 {
		t.Fatal(w.Body)
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/api/search?q=MZAP1", nil))
	if w.Code != 200 || !bytes.Contains(w.Body.Bytes(), []byte("C1.2")) {
		t.Fatal(w.Body)
	}
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/api/storage/options", nil))
	if w.Code != 200 || !bytes.Contains(w.Body.Bytes(), []byte("C2")) {
		t.Fatal(w.Body)
	}
	db.Close()
	if w := call(string(body), "application/json", ""); w.Code != 500 || strings.Contains(w.Body.String(), "sql") {
		t.Fatal(w.Body)
	}
}
