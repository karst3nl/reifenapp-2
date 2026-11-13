package handlers

import (
	"database/sql"
	"errors"
	"github.com/karst3nl/reifenapp-2/internal/database"
	"net/http"
	"time"
)

func Inactivity(db *sql.DB) http.HandlerFunc { return inactivity(db, time.Now) }
func inactivity(db *sql.DB, now func() time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		report, err := database.InactivePlaces(r.Context(), db, now())
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "Lagerbewegungen derzeit nicht verfügbar."})
			return
		}
		writeJSON(w, 200, report)
	}
}
func (a *Admin) inactivitySettings(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		var settings database.InactivitySettings
		if !readStorageJSON(w, r, &settings) {
			return
		}
		err := database.SaveInactivitySettings(r.Context(), a.db, settings)
		if errors.Is(err, database.ErrInactivityMonths) {
			writeJSON(w, 422, map[string]string{"error": err.Error()})
			return
		}
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "Frist konnte nicht gespeichert werden."})
			return
		}
	}
	settings, err := database.ReadInactivitySettings(r.Context(), a.db)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "Frist derzeit nicht verfügbar."})
		return
	}
	writeJSON(w, 200, settings)
}
