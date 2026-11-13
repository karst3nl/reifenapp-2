package handlers

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/karst3nl/reifenapp-2/internal/database"
	"github.com/karst3nl/reifenapp-2/internal/importcsv"
)

func (a *Admin) csvList(w http.ResponseWriter, r *http.Request) {
	batches, err := database.CSVImports(r.Context(), a.db)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "Importübersicht derzeit nicht verfügbar."})
		return
	}
	writeJSON(w, 200, batches)
}
func (a *Admin) csvUpload(w http.ResponseWriter, r *http.Request) (*importcsv.Document, string, bool) {
	if origin := r.Header.Get("Origin"); origin != "" {
		u, err := url.Parse(origin)
		if err != nil || u.Host != r.Host || (u.Scheme != "http" && u.Scheme != "https") {
			writeJSON(w, 403, map[string]string{"error": "Fremder Ursprung nicht erlaubt."})
			return nil, "", false
		}
	}
	defer func() {
		if r.MultipartForm != nil {
			r.MultipartForm.RemoveAll()
		}
	}()
	r.Body = http.MaxBytesReader(w, r.Body, importcsv.MaxBytes+1024*1024)
	if err := r.ParseMultipartForm(importcsv.MaxBytes); err != nil {
		writeJSON(w, 400, map[string]string{"error": "CSV-Upload ungültig oder größer als 10 MiB."})
		return nil, "", false
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, 400, map[string]string{"error": "Bitte eine CSV-Datei auswählen."})
		return nil, "", false
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, importcsv.MaxBytes+1))
	if err != nil {
		writeJSON(w, 400, map[string]string{"error": "CSV-Datei konnte nicht gelesen werden."})
		return nil, "", false
	}
	name := path.Base(strings.ReplaceAll(header.Filename, "\\", "/"))
	name = strings.Map(func(c rune) rune {
		if unicode.IsControl(c) {
			return '_'
		}
		return c
	}, name)
	runes := []rune(name)
	if len(runes) > 200 {
		name = string(runes[:200])
	}
	doc, err := importcsv.Parse(data)
	if err != nil {
		writeJSON(w, 422, map[string]string{"error": err.Error()})
		return nil, "", false
	}
	return doc, name, true
}
func (a *Admin) csvPreview(w http.ResponseWriter, r *http.Request) {
	doc, name, ok := a.csvUpload(w, r)
	if !ok {
		return
	}
	preview, err := database.PreviewCSV(r.Context(), a.db, doc, name)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "Importvorschau derzeit nicht verfügbar."})
		return
	}
	writeJSON(w, 200, preview)
}
func (a *Admin) csvCommit(w http.ResponseWriter, r *http.Request) {
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(5 * time.Minute))
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()
	r = r.WithContext(ctx)
	doc, name, ok := a.csvUpload(w, r)
	if !ok {
		return
	}
	if r.FormValue("confirmed") != "true" || r.FormValue("expected_sha256") != doc.Hash {
		writeJSON(w, 409, map[string]string{"error": "Bitte diese Datei zuerst prüfen und den Import ausdrücklich bestätigen."})
		return
	}
	if !a.backupMu.TryLock() {
		writeJSON(w, 429, map[string]string{"error": "Ein Backup oder Import läuft bereits. Bitte kurz warten."})
		return
	}
	defer a.backupMu.Unlock()
	if _, err := database.PreviewCSV(r.Context(), a.db, doc, name); err != nil {
		writeJSON(w, 500, map[string]string{"error": "Importprüfung derzeit nicht verfügbar."})
		return
	}
	result, err := database.ImportCSV(r.Context(), a.db, doc, name)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "Import nicht gespeichert. Bitte Speicherplatz und Datenbank prüfen und erneut versuchen."})
		return
	}
	writeJSON(w, 200, result)
}
func (a *Admin) csvOriginal(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSON(w, 400, map[string]string{"error": "Ungültiger Import."})
		return
	}
	name, data, err := database.OriginalCSV(r.Context(), a.db, id)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, 404, map[string]string{"error": "Import nicht gefunden."})
		return
	}
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "Originaldatei derzeit nicht verfügbar."})
		return
	}
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": name}))
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Write(data)
}
