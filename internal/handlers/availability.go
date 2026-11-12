package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"unicode/utf8"

	"github.com/karst3nl/reifenapp-2/internal/database"
)

func CheckStorage(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Purpose string `json:"purpose"`
			Area    string `json:"area"`
			Number  int    `json:"number"`
			Name    string `json:"name"`
			Plate   string `json:"plate"`
		}
		if !readStorageJSON(w, r, &input) {
			return
		}
		if (input.Purpose != "" && input.Purpose != "storage" && input.Purpose != "reservation") || utf8.RuneCountInString(input.Name) > 200 || utf8.RuneCountInString(input.Plate) > 30 || len(input.Area) > 12 {
			writeJSON(w, 422, map[string]string{"error": "Ungültige Prüfwerte."})
			return
		}
		checked, err := database.CheckPlace(r.Context(), db, input.Area, input.Number, input.Name, input.Plate)
		if errors.Is(err, database.ErrInvalidPlace) {
			writeJSON(w, 422, map[string]string{"error": err.Error()})
			return
		}
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "Lagerplatzprüfung derzeit nicht verfügbar."})
			return
		}
		if input.Purpose == "reservation" && checked.Status == "reserved" {
			checked.Allowed = false
			checked.Message = "Lagerplatz " + checked.Place + " ist bereits reserviert. Keine weitere Reservierung möglich."
		}
		writeJSON(w, 200, checked)
	}
}
