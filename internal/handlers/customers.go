package handlers

import (
	"database/sql"
	"net/http"
	"unicode/utf8"

	"github.com/karst3nl/reifenapp-2/internal/database"
)

func Customers(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		if utf8.RuneCountInString(q) > 200 {
			writeJSON(w, 400, map[string]string{"error": "Suchtext zu lang."})
			return
		}
		names, err := database.CustomerNames(r.Context(), db, q)
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "Namensvorschläge derzeit nicht verfügbar."})
			return
		}
		writeJSON(w, 200, names)
	}
}
