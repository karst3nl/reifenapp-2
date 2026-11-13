package handlers

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"errors"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/karst3nl/reifenapp-2/internal/database"
	"github.com/karst3nl/reifenapp-2/internal/receipts"
	"golang.org/x/crypto/bcrypt"
)

type adminSession struct {
	csrf       string
	expires    time.Time
	revision   int
	mustChange bool
}
type loginAttempts struct {
	count int
	until time.Time
}
type Admin struct {
	db       *sql.DB
	header   receipts.Header
	mu       sync.Mutex
	backupMu sync.Mutex
	sessions map[[32]byte]adminSession
	attempts map[string]loginAttempts
	now      func() time.Time
}

func NewAdmin(db *sql.DB, header receipts.Header) (*Admin, error) {
	if err := database.EnsureAdminAccount(context.Background(), db); err != nil {
		return nil, err
	}
	return &Admin{db: db, header: header, sessions: map[[32]byte]adminSession{}, attempts: map[string]loginAttempts{}, now: time.Now}, nil
}
func token() string {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(b[:])
}
func (a *Admin) session(r *http.Request) (adminSession, bool) {
	cookie, err := r.Cookie("reifenapp_admin")
	if err != nil {
		return adminSession{}, false
	}
	key := sha256.Sum256([]byte(cookie.Value))
	a.mu.Lock()
	s, ok := a.sessions[key]
	if ok && !a.now().Before(s.expires) {
		delete(a.sessions, key)
		ok = false
	}
	a.mu.Unlock()
	if !ok {
		return s, false
	}
	account, err := database.LoadAdminAccount(r.Context(), a.db)
	if err != nil || account.Revision != s.revision {
		return s, false
	}
	s.mustChange = account.MustChange
	return s, true
}
func (a *Admin) protect(next func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return a.authorize(next, false)
}
func (a *Admin) authorize(next func(http.ResponseWriter, *http.Request), allowSetup bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, ok := a.session(r)
		if !ok {
			writeJSON(w, 401, map[string]string{"error": "Bitte als Administrator anmelden."})
			return
		}
		if r.Method != "GET" && subtle.ConstantTimeCompare([]byte(r.Header.Get("X-CSRF-Token")), []byte(s.csrf)) != 1 {
			writeJSON(w, 403, map[string]string{"error": "Ungültige Sitzung. Bitte neu anmelden."})
			return
		}
		if s.mustChange && !allowSetup {
			writeJSON(w, 403, map[string]any{"error": "Bitte zuerst ein neues Admin-Passwort festlegen.", "must_change": true})
			return
		}
		next(w, r)
	}
}
func (a *Admin) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/admin/session", func(w http.ResponseWriter, r *http.Request) {
		if s, ok := a.session(r); ok {
			writeJSON(w, 200, map[string]any{"csrf": s.csrf, "expires": s.expires, "username": "admin", "must_change": s.mustChange})
			return
		}
		writeJSON(w, 401, map[string]any{"enabled": true, "error": "Bitte anmelden."})
	})
	mux.HandleFunc("POST /api/admin/login", a.login)
	mux.HandleFunc("POST /api/admin/password", a.authorize(a.changePassword, true))
	mux.HandleFunc("POST /api/admin/logout", a.authorize(a.logout, true))
	mux.HandleFunc("GET /api/admin/layout", a.protect(a.load))
	mux.HandleFunc("POST /api/admin/layout", a.protect(a.save))
	mux.HandleFunc("POST /api/admin/preview", a.protect(a.preview))
	mux.HandleFunc("GET /api/admin/areas", a.protect(a.areas))
	mux.HandleFunc("GET /api/admin/backup/info", a.protect(a.backupInfo))
	mux.HandleFunc("POST /api/admin/backup", a.protect(a.backup))
	mux.HandleFunc("POST /api/admin/areas", a.protect(a.areas))
	mux.HandleFunc("GET /api/admin/fields", a.protect(func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, receipts.Fields) }))
	mux.HandleFunc("GET /api/admin/values", a.protect(func(w http.ResponseWriter, r *http.Request) {
		op := r.URL.Query().Get("operation")
		if !receipts.ValidOperation(op) {
			writeJSON(w, 400, map[string]string{"error": "Ungültige Vorgangsart."})
			return
		}
		v, err := receipts.FieldValues(receipts.ExampleReceipt(op), a.header)
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "Vorschau nicht verfügbar."})
			return
		}
		writeJSON(w, 200, v)
	}))
	mux.HandleFunc("GET /api/admin/font", a.protect(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "font/ttf")
		w.Header().Set("Cache-Control", "private, no-store")
		if r.URL.Query().Get("bold") == "1" {
			w.Write(receipts.BoldFontBytes())
		} else {
			w.Write(receipts.FontBytes())
		}
	}))
}
func (a *Admin) putSession(w http.ResponseWriter, r *http.Request, value, csrf string, revision int, mustChange, clearAll bool) {
	now := a.now()
	a.mu.Lock()
	if clearAll {
		a.sessions = map[[32]byte]adminSession{}
	}
	if old, err := r.Cookie("reifenapp_admin"); err == nil {
		delete(a.sessions, sha256.Sum256([]byte(old.Value)))
	}
	if len(a.sessions) >= 64 {
		for k := range a.sessions {
			delete(a.sessions, k)
			break
		}
	}
	a.sessions[sha256.Sum256([]byte(value))] = adminSession{csrf: csrf, expires: now.Add(30 * time.Minute), revision: revision, mustChange: mustChange}
	a.mu.Unlock()
	http.SetCookie(w, &http.Cookie{Name: "reifenapp_admin", Value: value, Path: "/api/admin", HttpOnly: true, SameSite: http.SameSiteStrictMode, Secure: r.TLS != nil, MaxAge: 1800})
	writeJSON(w, 200, map[string]any{"csrf": csrf, "expires": now.Add(30 * time.Minute), "username": "admin", "must_change": mustChange})
}
func (a *Admin) login(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if !readStorageJSON(w, r, &input) {
		return
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ip = r.RemoteAddr
	}
	now := a.now()
	a.mu.Lock()
	for k, v := range a.attempts {
		if !now.Before(v.until) {
			delete(a.attempts, k)
		}
	}
	for k, v := range a.sessions {
		if !now.Before(v.expires) {
			delete(a.sessions, k)
		}
	}
	attempt := a.attempts[ip]
	if attempt.count >= 6 && now.Before(attempt.until) {
		a.mu.Unlock()
		w.Header().Set("Retry-After", "60")
		writeJSON(w, 429, map[string]string{"error": "Zu viele Anmeldeversuche. Bitte in einer Minute erneut versuchen."})
		return
	}
	if len(a.attempts) >= 1024 {
		a.mu.Unlock()
		writeJSON(w, 429, map[string]string{"error": "Bitte später erneut versuchen."})
		return
	}
	if attempt.count == 0 {
		attempt.until = now.Add(time.Minute)
	}
	attempt.count++
	a.attempts[ip] = attempt
	a.mu.Unlock()
	account, err := database.LoadAdminAccount(r.Context(), a.db)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "Anmeldung derzeit nicht möglich."})
		return
	}
	match := bcrypt.CompareHashAndPassword([]byte(account.Hash), []byte(input.Password)) == nil
	if input.Username != "admin" || len(input.Password) > 72 || !match {
		writeJSON(w, 401, map[string]string{"error": "Anmeldung fehlgeschlagen."})
		return
	}
	value, csrf := token(), token()
	if value == "" || csrf == "" {
		writeJSON(w, 500, map[string]string{"error": "Anmeldung derzeit nicht möglich."})
		return
	}
	a.putSession(w, r, value, csrf, account.Revision, account.MustChange, false)
}
func (a *Admin) changePassword(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Current string `json:"current_password"`
		New     string `json:"new_password"`
		Confirm string `json:"confirm_password"`
	}
	if !readStorageJSON(w, r, &input) {
		return
	}
	if input.New != input.Confirm {
		writeJSON(w, 422, map[string]string{"error": "Die neuen Passwörter stimmen nicht überein."})
		return
	}
	s, ok := a.session(r)
	if !ok {
		writeJSON(w, 401, map[string]string{"error": "Bitte neu anmelden."})
		return
	}
	value, csrf := token(), token()
	if value == "" || csrf == "" {
		writeJSON(w, 500, map[string]string{"error": "Passwortänderung derzeit nicht möglich."})
		return
	}
	account, err := database.ChangeAdminPassword(r.Context(), a.db, s.revision, input.Current, input.New)
	if errors.Is(err, database.ErrAdminRevision) {
		writeJSON(w, 409, map[string]string{"error": err.Error()})
		return
	}
	if errors.Is(err, database.ErrAdminPolicy) || errors.Is(err, database.ErrAdminCredentials) {
		writeJSON(w, 422, map[string]string{"error": err.Error()})
		return
	}
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "Passwort konnte nicht gespeichert werden. Bitte erneut versuchen."})
		return
	}
	a.putSession(w, r, value, csrf, account.Revision, false, true)
}
func (a *Admin) logout(w http.ResponseWriter, r *http.Request) {
	var body struct{}
	if !readStorageJSON(w, r, &body) {
		return
	}
	cookie, _ := r.Cookie("reifenapp_admin")
	a.mu.Lock()
	delete(a.sessions, sha256.Sum256([]byte(cookie.Value)))
	a.mu.Unlock()
	http.SetCookie(w, &http.Cookie{Name: "reifenapp_admin", Path: "/api/admin", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteStrictMode, Secure: r.TLS != nil})
	writeJSON(w, 200, map[string]bool{"ok": true})
}
func (a *Admin) load(w http.ResponseWriter, r *http.Request) {
	op := r.URL.Query().Get("operation")
	if !receipts.ValidOperation(op) {
		writeJSON(w, 400, map[string]string{"error": "Ungültige Vorgangsart."})
		return
	}
	layout, err := database.LoadPDFLayout(r.Context(), a.db, op)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "Vorlage nicht verfügbar."})
		return
	}
	writeJSON(w, 200, map[string]any{"layout": layout, "default": receipts.DefaultTemplate(op)})
}
func (a *Admin) save(w http.ResponseWriter, r *http.Request) {
	var layout database.PDFLayout
	if !readLimitedJSON(w, r, &layout, 8*1024*1024) {
		return
	}
	if !receipts.ValidOperation(layout.Operation) {
		writeJSON(w, 422, map[string]string{"error": "Ungültige Vorgangsart."})
		return
	}
	if err := layout.Template.Validate(); err != nil {
		writeJSON(w, 422, map[string]string{"error": err.Error()})
		return
	}
	if layout.Enabled {
		if _, err := receipts.GenerateTemplate(receipts.ExampleReceipt(layout.Operation), a.header, layout.Template); err != nil {
			writeJSON(w, 422, map[string]string{"error": err.Error()})
			return
		}
	}
	saved, err := database.SavePDFLayout(r.Context(), a.db, layout)
	if errors.Is(err, database.ErrLayoutConflict) {
		writeJSON(w, 409, map[string]string{"error": err.Error()})
		return
	}
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "Vorlage konnte nicht gespeichert werden."})
		return
	}
	writeJSON(w, 200, saved)
}
func (a *Admin) preview(w http.ResponseWriter, r *http.Request) {
	var layout database.PDFLayout
	if !readLimitedJSON(w, r, &layout, 8*1024*1024) {
		return
	}
	if !receipts.ValidOperation(layout.Operation) {
		writeJSON(w, 422, map[string]string{"error": "Ungültige Vorgangsart."})
		return
	}
	data, err := receipts.GenerateTemplate(receipts.ExampleReceipt(layout.Operation), a.header, layout.Template)
	if err != nil {
		writeJSON(w, 422, map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `inline; filename="layout-vorschau.pdf"`)
	w.Write(data)
}
