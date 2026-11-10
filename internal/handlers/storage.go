package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"

	"github.com/karst3nl/reifenapp-2/internal/database"
	"github.com/karst3nl/reifenapp-2/internal/models"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
func StorageOptions(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, models.StorageOptions())
}
func Storage(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Reject browser cross-origin writes; no CORS access is granted.
		if origin := r.Header.Get("Origin"); origin != "" {
			u, err := url.Parse(origin)
			if err != nil || u.Host != r.Host || (u.Scheme != "http" && u.Scheme != "https") {
				writeJSON(w, 403, map[string]string{"error": "Fremder Ursprung nicht erlaubt."})
				return
			}
		}
		media, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if media != "application/json" {
			writeJSON(w, 415, map[string]string{"error": "JSON wird erwartet."})
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 32768)
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		var input models.StorageInput
		err := decoder.Decode(&input)
		if err == nil {
			var extra any
			if decoder.Decode(&extra) != io.EOF {
				err = errors.New("additional JSON")
			}
		}
		if err != nil {
			writeJSON(w, 400, map[string]string{"error": "Ungültiges JSON oder Anfrage zu groß."})
			return
		}
		if fields := input.Validate(); len(fields) > 0 {
			writeJSON(w, 422, map[string]any{"error": "Bitte die markierten Felder prüfen.", "fields": fields})
			return
		}
		id, err := database.CreateStorage(r.Context(), db, input)
		if errors.Is(err, database.ErrPlaceUnavailable) {
			writeJSON(w, 409, map[string]string{"error": err.Error()})
			return
		}
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "Speichern derzeit nicht möglich. Bitte Eingaben behalten und erneut versuchen."})
			return
		}
		writeJSON(w, 201, map[string]any{"id": id, "place": input.Place(), "message": "Neueinlagerung gespeichert."})
	}
}
