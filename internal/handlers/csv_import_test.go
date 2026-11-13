package handlers

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/karst3nl/reifenapp-2/internal/database"
	"github.com/karst3nl/reifenapp-2/internal/receipts"
)

func TestCSVImportPermissionsAndConfirmation(t *testing.T) {
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
	raw := "DATUM,TÄTIGKEIT,NAME,KENNZEICHEN,LAGERPLATZ\n01.08.2024,Auslagerung,Fiktiver Test,MZ-FT 123,C1.222\n"
	upload := func(endpoint, csrf, origin, hash string, auth bool) *httptest.ResponseRecorder {
		var body bytes.Buffer
		w := multipart.NewWriter(&body)
		f, _ := w.CreateFormFile("file", "fiktiv.csv")
		f.Write([]byte(raw))
		if hash != "" {
			w.WriteField("confirmed", "true")
			w.WriteField("expected_sha256", hash)
		}
		w.Close()
		r := httptest.NewRequest("POST", "http://example.com/api/admin/imports/"+endpoint, &body)
		r.Header.Set("Content-Type", w.FormDataContentType())
		r.Header.Set("X-CSRF-Token", csrf)
		r.Header.Set("Origin", origin)
		if auth {
			r.AddCookie(&http.Cookie{Name: "reifenapp_admin", Value: "fiktive-session"})
		}
		out := httptest.NewRecorder()
		mux.ServeHTTP(out, r)
		return out
	}
	if w := upload("preview", "", "", "", false); w.Code != 401 {
		t.Fatal(w.Code)
	}
	account, err := database.LoadAdminAccount(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	a.sessions[sha256.Sum256([]byte("fiktive-session"))] = adminSession{csrf: "fiktive-csrf", expires: time.Now().Add(time.Hour), revision: account.Revision}
	if w := upload("preview", "fiktive-csrf", "", "", true); w.Code != 403 {
		t.Fatal("setup bypass", w.Code)
	}
	if _, err = db.Exec("UPDATE admin_account SET must_change=0"); err != nil {
		t.Fatal(err)
	}
	if w := upload("preview", "", "", "", true); w.Code != 403 {
		t.Fatal("csrf bypass", w.Code)
	}
	if w := upload("preview", "fiktive-csrf", "https://foreign.invalid", "", true); w.Code != 403 {
		t.Fatal("origin bypass", w.Code)
	}
	w := upload("preview", "fiktive-csrf", "http://example.com", "", true)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body)
	}
	var preview database.CSVPreview
	if err = json.Unmarshal(w.Body.Bytes(), &preview); err != nil {
		t.Fatal(err)
	}
	if w = upload("commit", "fiktive-csrf", "", "", true); w.Code != 409 {
		t.Fatal("confirmation bypass", w.Code)
	}
	if w = upload("commit", "fiktive-csrf", "", "wrong", true); w.Code != 409 {
		t.Fatal("hash bypass", w.Code)
	}
	if w = upload("commit", "fiktive-csrf", "", preview.Hash, true); w.Code != 200 {
		t.Fatal(w.Code, w.Body)
	}
	for _, endpoint := range []string{"/api/admin/imports", "/api/admin/imports/1/original"} {
		r := httptest.NewRequest("GET", endpoint, nil)
		out := httptest.NewRecorder()
		mux.ServeHTTP(out, r)
		if out.Code != 401 {
			t.Fatal("download permission", out.Code)
		}
	}
}
