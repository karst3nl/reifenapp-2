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

// StorageOptions exposes the live catalogue to every workshop form.
func StorageOptions(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		areas, err := database.StorageAreas(r.Context(), db)
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "Lagerbereiche nicht verfügbar."})
			return
		}
		options := map[string]any{}
		for key, value := range models.StorageOptions() {
			options[key] = value
		}
		codes := []string{}
		for _, a := range areas {
			codes = append(codes, a.Code)
		}
		options["area"] = codes
		options["storage_areas"] = areas
		writeJSON(w, 200, options)
	}
}

func Storage(db *sql.DB) http.HandlerFunc      { return storageHandler(db, false) }
func WheelChanges(db *sql.DB) http.HandlerFunc { return storageHandler(db, true) }
func storageHandler(db *sql.DB, wheelChange bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input models.StorageInput
		if !readStorageJSON(w, r, &input) {
			return
		}
		if fields := input.Validate(); len(fields) > 0 {
			writeJSON(w, 422, map[string]any{"error": "Bitte die markierten Felder prüfen.", "fields": fields})
			return
		}
		create := database.CreateStorage
		operation, message := "storage", "Neueinlagerung gespeichert."
		if wheelChange {
			create = database.CreateWheelChange
			operation = "wheel_change"
			message = "Radwechsel gespeichert."
		}
		id, err := create(r.Context(), db, input)
		if errors.Is(err, database.ErrInvalidPlace) {
			writeJSON(w, 422, map[string]any{"error": err.Error(), "fields": map[string]string{"number": err.Error()}})
			return
		}
		if errors.Is(err, database.ErrWheelChangeOwner) {
			writeJSON(w, 409, map[string]any{"error": database.ErrWheelChangeOwner.Error(), "fields": map[string]string{"name": database.ErrWheelChangeOwner.Error(), "plate": database.ErrWheelChangeOwner.Error()}})
			return
		}
		if errors.Is(err, database.ErrWheelChangeSameType) {
			writeJSON(w, 409, map[string]any{"error": database.ErrWheelChangeSameType.Error(), "fields": map[string]string{"tire_type": database.ErrWheelChangeSameType.Error()}})
			return
		}
		if errors.Is(err, database.ErrPlaceUnavailable) {
			writeJSON(w, 409, map[string]string{"error": err.Error()})
			return
		}
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "Speichern derzeit nicht möglich. Bitte Eingaben behalten und erneut versuchen."})
			return
		}
		writeJSON(w, 201, map[string]any{"id": id, "place": input.Place(), "message": message, "operation": operation, "pdf_url": models.ReceiptPath(id, operation)})
	}
}

func readStorageJSON(w http.ResponseWriter, r *http.Request, input any) bool {
	return readLimitedJSON(w, r, input, 32768)
}
func readLimitedJSON(w http.ResponseWriter, r *http.Request, input any, limit int64) bool {
	// Reject browser cross-origin writes; no CORS access is granted.
	if origin := r.Header.Get("Origin"); origin != "" {
		u, err := url.Parse(origin)
		if err != nil || u.Host != r.Host || (u.Scheme != "http" && u.Scheme != "https") {
			writeJSON(w, 403, map[string]string{"error": "Fremder Ursprung nicht erlaubt."})
			return false
		}
	}
	media, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if media != "application/json" {
		writeJSON(w, 415, map[string]string{"error": "JSON wird erwartet."})
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, limit)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	err := decoder.Decode(input)
	if err == nil {
		var extra any
		if decoder.Decode(&extra) != io.EOF {
			err = errors.New("additional JSON")
		}
	}
	if err != nil {
		writeJSON(w, 400, map[string]string{"error": "Ungültiges JSON oder Anfrage zu groß."})
		return false
	}

	return true
}
