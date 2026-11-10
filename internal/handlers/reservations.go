package handlers

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/karst3nl/reifenapp-2/internal/database"
	"github.com/karst3nl/reifenapp-2/internal/models"
)

func Reservations(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input models.ReservationInput
		if !readStorageJSON(w, r, &input) {
			return
		}
		if fields := input.Validate(); len(fields) > 0 {
			writeJSON(w, 422, map[string]any{"error": "Bitte die markierten Felder prüfen.", "fields": fields})
			return
		}
		id, err := database.CreateReservation(r.Context(), db, input)
		if errors.Is(err, database.ErrPlaceUnavailable) {
			writeJSON(w, 409, map[string]string{"error": "Lagerplatz ist bereits belegt oder reserviert. Keine weitere Reservierung möglich."})
			return
		}
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "Reservierung derzeit nicht möglich. Bitte erneut versuchen."})
			return
		}
		writeJSON(w, 201, map[string]any{"id": id, "place": input.Place(), "message": "Reservierung gespeichert."})
	}
}
