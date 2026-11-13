package handlers

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/karst3nl/reifenapp-2/internal/database"
	"github.com/karst3nl/reifenapp-2/internal/models"
	"github.com/karst3nl/reifenapp-2/internal/receipts"
)

func TestLabelAdminAndPreviewAPI(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(filepath.Join(t.TempDir(), "labels.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	a, err := NewAdmin(db, receipts.Header{})
	if err != nil {
		t.Fatal(err)
	}
	a.printer.Run = func(_ context.Context, name string, _ []byte, args ...string) ([]byte, error) {
		if name == "lp" {
			t.Fatal("preview must never physically print")
		}
		if strings.Contains(strings.Join(args, " "), "-v") {
			return []byte("device for Argox_USB: usb://Argox/Test\n"), nil
		}
		return []byte("printer Argox_USB is idle. enabled since today"), nil
	}
	mux := http.NewServeMux()
	a.Register(mux)
	mux.HandleFunc("GET /api/label-jobs/{id}", LabelJobStatus(db))
	call := func(method, path string, body any, auth, csrf bool) *httptest.ResponseRecorder {
		var raw []byte
		if body != nil {
			raw, _ = json.Marshal(body)
		}
		r := httptest.NewRequest(method, path, bytes.NewReader(raw))
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
	for _, path := range []string{"config", "printers", "jobs"} {
		if w := call("GET", "/api/admin/labels/"+path, nil, false, false); w.Code != 401 {
			t.Fatal(path, w.Code)
		}
	}
	account, err := database.LoadAdminAccount(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	a.sessions[sha256.Sum256([]byte("fictional-session"))] = adminSession{csrf: "fictional-csrf", expires: time.Now().Add(time.Hour), revision: account.Revision}
	if w := call("GET", "/api/admin/labels/config", nil, true, false); w.Code != 403 {
		t.Fatal("setup bypass", w.Code)
	}
	if _, err = db.Exec("UPDATE admin_account SET must_change=0"); err != nil {
		t.Fatal(err)
	}
	s := models.DefaultLabelSettings()
	s.Queue = "Argox_USB"
	s.Enabled = true
	if w := call("POST", "/api/admin/labels/preview", s, true, false); w.Code != 403 {
		t.Fatal("csrf bypass", w.Code)
	}
	if w := call("POST", "/api/admin/labels/preview", s, true, true); w.Code != 200 || w.Header().Get("Content-Type") != "application/pdf" || !bytes.HasPrefix(w.Body.Bytes(), []byte("%PDF-")) {
		t.Fatal(w.Code, w.Body)
	}
	c := database.LabelConfig{Revision: 1, Settings: s}
	if w := call("POST", "/api/admin/labels/config", c, true, true); w.Code != 200 {
		t.Fatal(w.Code, w.Body)
	}
	if w := call("POST", "/api/admin/labels/config", c, true, true); w.Code != 409 {
		t.Fatal("stale revision", w.Code)
	}
	if w := call("POST", "/api/admin/labels/test", map[string]bool{"confirmed": false}, true, true); w.Code != 422 {
		t.Fatal(w.Code)
	}
	if w := call("POST", "/api/admin/labels/test", map[string]bool{"confirmed": true}, true, true); w.Code != 202 {
		t.Fatal(w.Code, w.Body)
	}
	jobs, err := database.LabelJobs(ctx, db)
	if err != nil || len(jobs) != 1 || jobs[0].EventID != 0 || jobs[0].Settings.Copies != 1 {
		t.Fatal(jobs, err)
	}
	if w := call("GET", "/api/label-jobs/1", nil, false, false); w.Code != 200 || strings.Contains(w.Body.String(), "Mustermann") {
		t.Fatal("status leaks receipt", w.Body)
	}
	if w := call("POST", "/api/admin/labels/jobs/1", map[string]any{"action": "cancel", "confirmed": false}, true, true); w.Code != 422 {
		t.Fatal(w.Code)
	}
	if w := call("POST", "/api/admin/labels/jobs/1", map[string]any{"action": "cancel", "confirmed": true}, true, true); w.Code != 200 {
		t.Fatal(w.Code, w.Body)
	}
}
