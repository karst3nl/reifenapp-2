package handlers

import (
	"database/sql"
	"github.com/karst3nl/reifenapp-2/internal/database"
	"net/http"
)

func TreadWarnings(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		scope := r.URL.Query().Get("scope")
		if scope == "" {
			scope = "active"
		}
		if scope != "active" && scope != "all" {
			writeJSON(w, 400, map[string]string{"error": "Ungültige Warnungsansicht."})
			return
		}
		items, err := database.TreadWarnings(r.Context(), db, scope == "active")
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "Reifenwarnungen derzeit nicht verfügbar."})
			return
		}
		axles := 0
		for _, item := range items {
			axles += len(item.TreadWarnings)
		}
		writeJSON(w, 200, map[string]any{"scope": scope, "count": len(items), "axle_count": axles, "items": items})
	}
}
