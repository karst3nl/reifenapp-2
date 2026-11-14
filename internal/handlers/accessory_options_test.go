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

func TestAccessoryRequiredForBothOperationAPIs(t *testing.T) {
	for _, route := range []string{"/api/storage", "/api/wheel-changes"} {
		t.Run(route, func(t *testing.T) {
			db, err := database.Open(filepath.Join(t.TempDir(), "accessories.sqlite"))
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			handler := Storage(db)
			if route == "/api/wheel-changes" {
				handler = WheelChanges(db)
			}
			body := []byte(`{"area":"C1","number":2,"name":"Fiktiver Zubehörtest","plate":"MZ-ZT 123","brand":"VW","model":"Golf","tire_type":"Winter","rim_type":"Stahl","front":{"width":"205","ratio":"55","inches":"16","load":"91","speed":"H"},"rear":{"width":"205","ratio":"55","inches":"16","load":"91","speed":"H"},"front_depth":"6.5","rear_depth":"7.0"}`)
			var input map[string]any
			if err = json.Unmarshal(body, &input); err != nil {
				t.Fatal(err)
			}
			call := func() *httptest.ResponseRecorder {
				raw, _ := json.Marshal(input)
				r := httptest.NewRequest("POST", route, bytes.NewReader(raw))
				r.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()
				handler(w, r)
				return w
			}
			for _, value := range []any{nil, map[string]any{}, map[string]any{"present": nil}} {
				for _, key := range []string{"hubcaps", "bolts", "lock"} {
					input[key] = value
				}
				w := call()
				if w.Code != 422 || !strings.Contains(w.Body.String(), "hubcaps.present") {
					t.Fatal("missing answer accepted", w.Code, w.Body)
				}
			}
			for _, key := range []string{"hubcaps", "bolts", "lock"} {
				input[key] = map[string]any{"present": false}
			}
			input["hubcaps"] = map[string]any{"present": true, "location": ""}
			if w := call(); w.Code != 422 {
				t.Fatal(w.Code, w.Body)
			}
			input["hubcaps"] = map[string]any{"present": true, "location": "Unbekannter Ort (fiktiv)"}
			if w := call(); w.Code != 422 || !strings.Contains(w.Body.String(), "accessory_options") {
				t.Fatal(w.Code, w.Body)
			}
			input["hubcaps"] = map[string]any{"present": false}
			if w := call(); w.Code != 201 {
				t.Fatal("explicit no rejected", w.Code, w.Body)
			}
		})
	}
}
func TestAccessoryAdminProtectionAndLiveOptions(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "admin-accessories.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	a, err := NewAdmin(db, receipts.Header{})
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	a.Register(mux)
	mux.HandleFunc("GET /api/storage/options", StorageOptions(db))
	call := func(method, path string, body any, auth, csrf bool) *httptest.ResponseRecorder {
		raw, _ := json.Marshal(body)
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
	if w := call("GET", "/api/admin/accessory-options", nil, false, false); w.Code != 401 {
		t.Fatal(w.Code)
	}
	account, err := database.LoadAdminAccount(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	a.sessions[sha256.Sum256([]byte("fictional-session"))] = adminSession{csrf: "fictional-csrf", expires: time.Now().Add(time.Hour), revision: account.Revision}
	if w := call("GET", "/api/admin/accessory-options", nil, true, false); w.Code != 403 {
		t.Fatal("setup bypass", w.Code)
	}
	if _, err = db.Exec("UPDATE admin_account SET must_change=0"); err != nil {
		t.Fatal(err)
	}
	c, err := database.LoadAccessoryOptions(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	c.Locations["lock"] = []string{"Testfach (fiktiv)"}
	if w := call("POST", "/api/admin/accessory-options", c, true, false); w.Code != 403 {
		t.Fatal("csrf bypass", w.Code)
	}
	if w := call("POST", "/api/admin/accessory-options", c, true, true); w.Code != 200 {
		t.Fatal(w.Code, w.Body)
	}
	if w := call("POST", "/api/admin/accessory-options", c, true, true); w.Code != 409 {
		t.Fatal("revision bypass", w.Code)
	}
	if w := call("GET", "/api/storage/options", nil, false, false); w.Code != 200 || !strings.Contains(w.Body.String(), "Testfach (fiktiv)") {
		t.Fatal("options not live", w.Code, w.Body)
	}
	// Keep the persisted historical accessory representation unchanged.
	raw, _ := json.Marshal(models.Accessory{Present: false})
	if string(raw) != `{"present":false,"location":""}` {
		t.Fatal("accessory JSON changed", string(raw))
	}
}
