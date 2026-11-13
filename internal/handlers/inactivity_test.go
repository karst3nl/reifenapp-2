package handlers

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"github.com/karst3nl/reifenapp-2/internal/database"
	"github.com/karst3nl/reifenapp-2/internal/receipts"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"
)

func TestInactivityAPIAndProtectedSettings(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "inactivity.sqlite"))
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
	mux.HandleFunc("GET /api/inactivity", inactivity(db, func() time.Time { return time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC) }))
	call := func(method, path, body string, auth, csrf bool) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, bytes.NewBufferString(body))
		r.Header.Set("Content-Type", "application/json")
		if auth {
			r.AddCookie(&http.Cookie{Name: "reifenapp_admin", Value: "fictional-session"})
		}
		if csrf {
			r.Header.Set("X-CSRF-Token", "fictional-csrf")
		}
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		return w
	}
	if w := call("GET", "/api/admin/inactivity", "", false, false); w.Code != 401 {
		t.Fatal(w.Code)
	}
	account, err := database.LoadAdminAccount(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	a.sessions[sha256.Sum256([]byte("fictional-session"))] = adminSession{csrf: "fictional-csrf", expires: time.Now().Add(time.Hour), revision: account.Revision}
	if w := call("POST", "/api/admin/inactivity", "{\"months\":6}", true, true); w.Code != 403 {
		t.Fatal("setup bypass", w.Code)
	}
	if _, err = db.Exec("UPDATE admin_account SET must_change=0"); err != nil {
		t.Fatal(err)
	}
	if w := call("POST", "/api/admin/inactivity", "{\"months\":6}", true, false); w.Code != 403 {
		t.Fatal("csrf bypass", w.Code)
	}
	for _, body := range []string{"{\"months\":0}", "{\"months\":121}"} {
		if w := call("POST", "/api/admin/inactivity", body, true, true); w.Code != 422 {
			t.Fatal(w.Code)
		}
	}
	if w := call("POST", "/api/admin/inactivity", "{\"months\":6}", true, true); w.Code != 200 {
		t.Fatal(w.Code, w.Body)
	}
	w := call("GET", "/api/inactivity", "", false, false)
	var report database.InactivityReport
	if err = json.Unmarshal(w.Body.Bytes(), &report); err != nil || w.Code != 200 || report.Months != 6 || report.Count != 0 || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal(w.Code, w.Body, err)
	}
	if w = call("POST", "/api/inactivity", "{}", false, false); w.Code != 405 {
		t.Fatal(w.Code)
	}
	db.Close()
	if w = call("GET", "/api/inactivity", "", false, false); w.Code != 500 {
		t.Fatal(w.Code)
	}
}
