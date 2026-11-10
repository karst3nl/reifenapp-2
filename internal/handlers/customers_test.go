package handlers

import (
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/karst3nl/reifenapp-2/internal/database"
)

func TestCustomerSuggestions(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "test.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, name := range []string{"Müller Fiktiv", "Müller Fiktiv", "Mustermann Fiktiv", "Anderer Test"} {
		if _, err = db.Exec("INSERT INTO customers(name) VALUES(?)", name); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		q string
		n int
	}{{"MÜ", 1}, {"Mu", 1}, {"", 0}, {"unbekannt", 0}, {"%", 0}} {
		w := httptest.NewRecorder()
		Customers(db)(w, httptest.NewRequest("GET", "/api/customers?q="+tc.q, nil))
		var names []string
		if err = json.Unmarshal(w.Body.Bytes(), &names); err != nil || len(names) != tc.n || names == nil || w.Code != 200 {
			t.Fatalf("%q %s %v", tc.q, w.Body, err)
		}
		if w.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("cache header")
		}
	}
	for range 25 {
		if _, err = db.Exec("INSERT INTO customers(name) VALUES('Test-' || lower(hex(randomblob(8))))"); err != nil {
			t.Fatal(err)
		}
	}
	w := httptest.NewRecorder()
	Customers(db)(w, httptest.NewRequest("GET", "/api/customers?q=Test-", nil))
	var names []string
	json.Unmarshal(w.Body.Bytes(), &names)
	if len(names) != 20 {
		t.Fatal(len(names))
	}
	w = httptest.NewRecorder()
	Customers(db)(w, httptest.NewRequest("GET", "/api/customers?q="+strings.Repeat("a", 201), nil))
	if w.Code != 400 {
		t.Fatal(w.Code)
	}
	db.Close()
	w = httptest.NewRecorder()
	Customers(db)(w, httptest.NewRequest("GET", "/api/customers?q=M", nil))
	if w.Code != 500 {
		t.Fatal(w.Code)
	}
}
