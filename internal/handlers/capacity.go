package handlers

import (
	"database/sql"
	"github.com/karst3nl/reifenapp-2/internal/database"
	"net/http"
)

func Capacity(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		capacity, err := database.StorageCapacity(r.Context(), db)
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "Lagerplatzübersicht derzeit nicht verfügbar."})
			return
		}
		writeJSON(w, 200, capacity)
	}
}
