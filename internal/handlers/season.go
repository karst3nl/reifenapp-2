package handlers

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/karst3nl/reifenapp-2/internal/database"
)

func SeasonProgress(db *sql.DB) http.HandlerFunc { return seasonProgress(db, time.Now) }
func seasonProgress(db *sql.DB, now func() time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		progress, err := database.CurrentSeasonProgress(r.Context(), db, now())
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "Saisonstand derzeit nicht verfügbar."})
			return
		}
		writeJSON(w, 200, progress)
	}
}
