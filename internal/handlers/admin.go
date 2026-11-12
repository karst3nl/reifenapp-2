package handlers

import (
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
)

type adminSession struct {
	csrf    string
	expires time.Time
}
type loginAttempts struct {
	count int
	until time.Time
}
type Admin struct {
	db       *sql.DB
	header   receipts.Header
	password [32]byte
	enabled  bool
	mu       sync.Mutex
	sessions map[[32]byte]adminSession
	attempts map[string]loginAttempts
	now      func() time.Time
}

func NewAdmin(db *sql.DB, header receipts.Header, password string) *Admin {
	return &Admin{db: db, header: header, password: sha256.Sum256([]byte(password)), enabled: len(password) >= 12 && len(password) <= 128, sessions: map[[32]byte]adminSession{}, attempts: map[string]loginAttempts{}, now: time.Now}
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
	defer a.mu.Unlock()
	s, ok := a.sessions[key]
	if ok && !a.now().Before(s.expires) {
		delete(a.sessions, key)
		ok = false
	}
	return s, ok
}
func (a *Admin) protect(next func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !a.enabled {
			writeJSON(w, 503, map[string]any{"error": "Administration ist noch nicht eingerichtet.", "enabled": false})
			return
		}
		s, ok := a.session(r)
		if !ok {
			writeJSON(w, 401, map[string]string{"error": "Bitte als Administrator anmelden."})
			return
		}
		if r.Method != "GET" && subtle.ConstantTimeCompare([]byte(r.Header.Get("X-CSRF-Token")), []byte(s.csrf)) != 1 {
			writeJSON(w, 403, map[string]string{"error": "Ungültige Sitzung. Bitte neu anmelden."})
			return
		}
		next(w, r)
	}
}
func (a *Admin) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/admin/session", func(w http.ResponseWriter, r *http.Request) {
		if !a.enabled {
			writeJSON(w, 503, map[string]any{"enabled": false, "error": "Administration ist noch nicht eingerichtet."})
			return
		}
		if s, ok := a.session(r); ok {
			writeJSON(w, 200, map[string]any{"csrf": s.csrf, "expires": s.expires})
			return
		}
		writeJSON(w, 401, map[string]any{"enabled": true, "error": "Bitte anmelden."})
	})
	mux.HandleFunc("POST /api/admin/login", a.login)
	mux.HandleFunc("POST /api/admin/logout", a.protect(a.logout))
	mux.HandleFunc("GET /api/admin/layout", a.protect(a.load))
	mux.HandleFunc("POST /api/admin/layout", a.protect(a.save))
	mux.HandleFunc("POST /api/admin/preview", a.protect(a.preview))
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
func (a *Admin) login(w http.ResponseWriter, r *http.Request) {
	if !a.enabled {
		writeJSON(w, 503, map[string]string{"error": "Administration ist noch nicht eingerichtet."})
		return
	}
	var input struct {
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
	got := sha256.Sum256([]byte(input.Password))
	if len(input.Password) > 128 || subtle.ConstantTimeCompare(got[:], a.password[:]) != 1 {
		writeJSON(w, 401, map[string]string{"error": "Anmeldung fehlgeschlagen."})
		return
	}
	value, csrf := token(), token()
	if value == "" || csrf == "" {
		writeJSON(w, 500, map[string]string{"error": "Anmeldung derzeit nicht möglich."})
		return
	}
	a.mu.Lock()
	if old, err := r.Cookie("reifenapp_admin"); err == nil {
		delete(a.sessions, sha256.Sum256([]byte(old.Value)))
	}
	if len(a.sessions) >= 64 {
		for k := range a.sessions {
			delete(a.sessions, k)
			break
		}
	}
	a.sessions[sha256.Sum256([]byte(value))] = adminSession{csrf: csrf, expires: now.Add(30 * time.Minute)}
	a.mu.Unlock()
	http.SetCookie(w, &http.Cookie{Name: "reifenapp_admin", Value: value, Path: "/api/admin", HttpOnly: true, SameSite: http.SameSiteStrictMode, Secure: r.TLS != nil, MaxAge: 1800})
	writeJSON(w, 200, map[string]any{"csrf": csrf, "expires": now.Add(30 * time.Minute)})
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
	if !readLimitedJSON(w, r, &layout, 256*1024) {
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
	if !readLimitedJSON(w, r, &layout, 256*1024) {
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
