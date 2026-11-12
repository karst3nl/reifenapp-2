package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/karst3nl/reifenapp-2/internal/database"
	"github.com/karst3nl/reifenapp-2/internal/receipts"
)

func TestAdminAuthenticationAndEditorAPI(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "admin.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	a := NewAdmin(db, receipts.Header{}, "FiktivesTestPasswort123!")
	mux := http.NewServeMux()
	a.Register(mux)
	var cookie *http.Cookie
	csrf := ""
	call := func(method, path string, body []byte, withCSRF bool) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		if cookie != nil {
			r.AddCookie(cookie)
		}
		if withCSRF {
			r.Header.Set("X-CSRF-Token", csrf)
		}
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		return w
	}
	if w := call("GET", "/api/admin/layout?operation=storage", nil, false); w.Code != 401 {
		t.Fatal(w.Code)
	}
	w := call("POST", "/api/admin/login", []byte(`{"password":"wrong"}`), false)
	if w.Code != 401 {
		t.Fatal(w.Code)
	}
	w = call("POST", "/api/admin/login", []byte(`{"password":"FiktivesTestPasswort123!"}`), false)
	if w.Code != 200 {
		t.Fatal(w.Body)
	}
	cookie = w.Result().Cookies()[0]
	if !cookie.HttpOnly || cookie.SameSite != http.SameSiteStrictMode {
		t.Fatal(cookie)
	}
	var login struct {
		CSRF string `json:"csrf"`
	}
	json.Unmarshal(w.Body.Bytes(), &login)
	csrf = login.CSRF
	w = call("GET", "/api/admin/layout?operation=storage", nil, false)
	if w.Code != 200 {
		t.Fatal(w.Body)
	}
	var response struct {
		Layout database.PDFLayout `json:"layout"`
	}
	json.Unmarshal(w.Body.Bytes(), &response)
	response.Layout.Enabled = true
	body, _ := json.Marshal(response.Layout)
	if w = call("POST", "/api/admin/layout", body, false); w.Code != 403 {
		t.Fatal(w.Code)
	}
	if w = call("POST", "/api/admin/preview", body, true); w.Code != 200 || w.Header().Get("Content-Type") != "application/pdf" || !bytes.HasPrefix(w.Body.Bytes(), []byte("%PDF-")) {
		t.Fatal(w.Code, w.Body)
	}
	w = call("POST", "/api/admin/layout", body, true)
	if w.Code != 200 {
		t.Fatal(w.Body)
	}
	if w = call("POST", "/api/admin/layout", body, true); w.Code != 409 {
		t.Fatal(w.Body)
	}
	if w = call("GET", "/api/admin/font?bold=1", nil, false); w.Code != 200 || len(w.Body.Bytes()) < 1000 {
		t.Fatal(w.Code)
	}
	if w = call("GET", "/api/admin/values?operation=reservation", nil, false); w.Code != 200 || !strings.Contains(w.Body.String(), "fiktiv") {
		t.Fatal(w.Body)
	}
	if w = call("POST", "/api/admin/logout", []byte(`{}`), true); w.Code != 200 {
		t.Fatal(w.Body)
	}
	if w = call("GET", "/api/admin/fields", nil, false); w.Code != 401 {
		t.Fatal(w.Code)
	}
}
func TestAdminDisabledRateLimitAndExpiration(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "auth.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	disabled := NewAdmin(db, receipts.Header{}, "")
	mux := http.NewServeMux()
	disabled.Register(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/api/admin/session", nil))
	if w.Code != 503 {
		t.Fatal(w.Code)
	}
	a := NewAdmin(db, receipts.Header{}, "FiktivesTestPasswort123!")
	mux = http.NewServeMux()
	a.Register(mux)
	for i := 0; i < 7; i++ {
		r := httptest.NewRequest("POST", "/api/admin/login", strings.NewReader(`{"password":"wrong"}`))
		r.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		expected := 401
		if i == 6 {
			expected = 429
		}
		if w.Code != expected {
			t.Fatal(i, w.Code)
		}
	}
	a.now = func() time.Time { return time.Now().Add(2 * time.Minute) }
	r := httptest.NewRequest("POST", "/api/admin/login", strings.NewReader(`{"password":"FiktivesTestPasswort123!"}`))
	r.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatal(w.Body)
	}
	cookie := w.Result().Cookies()[0]
	a.now = func() time.Time { return time.Now().Add(time.Hour) }
	r = httptest.NewRequest("GET", "/api/admin/fields", nil)
	r.AddCookie(cookie)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != 401 {
		t.Fatal(w.Code)
	}
}
