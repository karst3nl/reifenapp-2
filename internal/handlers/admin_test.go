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

func TestAdminSetupAuthenticationAndEditorAPI(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "admin.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	a, err := NewAdmin(db, receipts.Header{})
	if err != nil {
		t.Fatal(err)
	}
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
	session := func(w *httptest.ResponseRecorder) bool {
		t.Helper()
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body)
		}
		cookie = w.Result().Cookies()[0]
		var response struct {
			CSRF       string `json:"csrf"`
			MustChange bool   `json:"must_change"`
		}
		if err = json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		csrf = response.CSRF
		return response.MustChange
	}
	if w := call("GET", "/api/admin/layout?operation=storage", nil, false); w.Code != 401 {
		t.Fatal(w.Code)
	}
	body, _ := json.Marshal(map[string]string{"username": "other", "password": database.DefaultAdminPassword})
	if w := call("POST", "/api/admin/login", body, false); w.Code != 401 {
		t.Fatal(w.Code)
	}
	body, _ = json.Marshal(map[string]string{"username": "admin", "password": database.DefaultAdminPassword})
	w := call("POST", "/api/admin/login", body, false)
	if !session(w) {
		t.Fatal("first login must change password")
	}
	if !cookie.HttpOnly || cookie.SameSite != http.SameSiteStrictMode {
		t.Fatal(cookie)
	}
	oldCookie := cookie
	for _, path := range []string{"/api/admin/layout?operation=storage", "/api/admin/fields", "/api/admin/font"} {
		if w = call("GET", path, nil, false); w.Code != 403 {
			t.Fatal("setup bypass", path, w.Code)
		}
	}
	change := []byte(`{"new_password":"NeuesFiktivesPasswort123!","confirm_password":"NeuesFiktivesPasswort123!"}`)
	if w = call("POST", "/api/admin/password", change, false); w.Code != 403 {
		t.Fatal("missing CSRF", w.Code)
	}
	if w = call("POST", "/api/admin/password", []byte(`{"new_password":"different","confirm_password":"mismatch"}`), true); w.Code != 422 {
		t.Fatal(w.Code)
	}
	if session(call("POST", "/api/admin/password", change, true)) {
		t.Fatal("change flag not cleared")
	}
	freshCookie := cookie
	cookie = oldCookie
	if w = call("GET", "/api/admin/fields", nil, false); w.Code != 401 {
		t.Fatal("old session survived")
	}
	cookie = freshCookie
	w = call("GET", "/api/admin/layout?operation=storage", nil, false)
	if w.Code != 200 {
		t.Fatal(w.Body)
	}
	var response struct {
		Layout database.PDFLayout `json:"layout"`
	}
	json.Unmarshal(w.Body.Bytes(), &response)
	response.Layout.Enabled = true
	body, _ = json.Marshal(response.Layout)
	if w = call("POST", "/api/admin/layout", body, false); w.Code != 403 {
		t.Fatal(w.Code)
	}
	if w = call("POST", "/api/admin/preview", body, true); w.Code != 200 || !bytes.HasPrefix(w.Body.Bytes(), []byte("%PDF-")) {
		t.Fatal(w.Code, w.Body)
	}
	if w = call("POST", "/api/admin/layout", body, true); w.Code != 200 {
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
	if w = call("POST", "/api/admin/password", []byte(`{"current_password":"wrong","new_password":"NochEinFiktivesPasswort123!","confirm_password":"NochEinFiktivesPasswort123!"}`), true); w.Code != 422 {
		t.Fatal("current password not checked", w.Code)
	}
	if w = call("POST", "/api/admin/logout", []byte(`{}`), true); w.Code != 200 {
		t.Fatal(w.Body)
	}
	cookie = nil
	body, _ = json.Marshal(map[string]string{"username": "admin", "password": database.DefaultAdminPassword})
	if w = call("POST", "/api/admin/login", body, false); w.Code != 401 {
		t.Fatal("default still usable")
	}
	body, _ = json.Marshal(map[string]string{"username": "admin", "password": "NeuesFiktivesPasswort123!"})
	if session(call("POST", "/api/admin/login", body, false)) {
		t.Fatal("setup flag returned")
	}
}
func TestAdminRateLimitExpirationAndRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "admin.sqlite")
	db, err := database.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	a, err := NewAdmin(db, receipts.Header{})
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	a.Register(mux)
	for i := 0; i < 7; i++ {
		r := httptest.NewRequest("POST", "/api/admin/login", strings.NewReader(`{"username":"admin","password":"wrong"}`))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
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
	body, _ := json.Marshal(map[string]string{"username": "admin", "password": database.DefaultAdminPassword})
	r := httptest.NewRequest("POST", "/api/admin/login", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatal(w.Body)
	}
	cookie := w.Result().Cookies()[0]
	a.now = func() time.Time { return time.Now().Add(time.Hour) }
	r = httptest.NewRequest("GET", "/api/admin/session", nil)
	r.AddCookie(cookie)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != 401 {
		t.Fatal(w.Code)
	}
	if _, err = database.ChangeAdminPassword(r.Context(), db, 1, "", "PersistentesFiktivPasswort123!"); err != nil {
		t.Fatal(err)
	}
	db.Close()
	db, err = database.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	restarted, err := NewAdmin(db, receipts.Header{})
	if err != nil {
		t.Fatal(err)
	}
	mux = http.NewServeMux()
	restarted.Register(mux)
	for password, want := range map[string]int{database.DefaultAdminPassword: 401, "PersistentesFiktivPasswort123!": 200} {
		body, _ = json.Marshal(map[string]string{"username": "admin", "password": password})
		r = httptest.NewRequest("POST", "/api/admin/login", bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code != want {
			t.Fatal("restart", w.Code, want)
		}
	}
}
