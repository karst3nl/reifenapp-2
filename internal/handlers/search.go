package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"unicode/utf8"

	"github.com/karst3nl/reifenapp-2/internal/database"
)

func Search(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		q := r.URL.Query().Get("q")
		if utf8.RuneCountInString(q) > 200 {
			http.Error(w, `{"error":"Suchtext zu lang"}`, http.StatusBadRequest)
			return
		}
		result, err := database.Search(r.Context(), db, q)
		if err != nil {
			http.Error(w, `{"error":"Suche derzeit nicht verfügbar"}`, http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(result)
	}
}
