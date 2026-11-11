package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"github.com/karst3nl/reifenapp-2/internal/database"
)

func CheckRelease(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Area   string `json:"area"`
			Number int    `json:"number"`
		}
		if !readStorageJSON(w, r, &input) {
			return
		}
		target, err := database.LookupRelease(r.Context(), db, input.Area, input.Number)
		if errors.Is(err, database.ErrInvalidPlace) {
			writeJSON(w, 422, map[string]string{"error": err.Error()})
			return
		}
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "Belegung derzeit nicht abrufbar."})
			return
		}
		writeJSON(w, 200, target)
	}
}
func Releases(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Area      string `json:"area"`
			Number    int    `json:"number"`
			ClaimID   int64  `json:"claim_id"`
			Confirmed bool   `json:"confirmed"`
		}
		if !readStorageJSON(w, r, &input) {
			return
		}
		if !input.Confirmed || input.ClaimID <= 0 {
			writeJSON(w, 422, map[string]string{"error": "Bitte die angezeigte Belegung prüfen und die Auslagerung bestätigen."})
			return
		}
		err := database.ReleaseStorage(r.Context(), db, input.Area, input.Number, input.ClaimID)
		if errors.Is(err, database.ErrInvalidPlace) {
			writeJSON(w, 422, map[string]string{"error": err.Error()})
			return
		}
		if errors.Is(err, database.ErrReleaseChanged) {
			writeJSON(w, 409, map[string]string{"error": err.Error()})
			return
		}
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "Auslagerung konnte nicht bestätigt werden. Bitte Belegung erneut prüfen."})
			return
		}
		writeJSON(w, 200, map[string]any{"place": fmt.Sprintf("%s.%d", input.Area, input.Number), "message": "Auslagerung gespeichert. Lagerplatz ist wieder frei."})
	}
}
