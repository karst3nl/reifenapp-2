package handlers

import (
	"database/sql"
	"net/http"
)

// Health verifies database access without returning workshop or customer data.
func Health(db *sql.DB, version string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := db.PingContext(r.Context()); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unavailable"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "version": version})
	}
}
