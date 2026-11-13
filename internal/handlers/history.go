package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/karst3nl/reifenapp-2/internal/database"
	"github.com/karst3nl/reifenapp-2/internal/receipts"
)

func History(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		f := database.HistoryFilter{Query: q.Get("q"), Operation: q.Get("operation"), Status: q.Get("status"), Source: q.Get("source"), Page: 1, PageSize: 20}
		invalid := func() {
			writeJSON(w, 400, map[string]string{"error": "Ungültige Historienfilter. Bitte Suchtext, Zeitraum und Seitenauswahl prüfen."})
		}
		if utf8.RuneCountInString(f.Query) > 200 || (f.Operation != "" && !receipts.ValidOperation(f.Operation)) || (f.Status != "" && f.Status != "active" && f.Status != "closed") || (f.Source != "" && f.Source != "recorded" && f.Source != "legacy") {
			invalid()
			return
		}
		for key, target := range map[string]*int{"page": &f.Page, "page_size": &f.PageSize} {
			if v := q.Get(key); v != "" {
				n, err := strconv.Atoi(v)
				if err != nil || n < 1 || n > 1000000 {
					invalid()
					return
				}
				*target = n
			}
		}
		if f.PageSize != 20 && f.PageSize != 50 && f.PageSize != 100 {
			invalid()
			return
		}
		if v := q.Get("snapshot"); v != "" {
			n, err := strconv.ParseInt(v, 10, 64)
			if err != nil || n < 0 {
				invalid()
				return
			}
			f.Snapshot = n
		}
		for key, target := range map[string]*time.Time{"from": &f.From, "until": &f.Until} {
			if v := q.Get(key); v != "" {
				t, err := time.Parse(time.RFC3339Nano, v)
				if err != nil {
					invalid()
					return
				}
				*target = t
			}
		}
		if !f.From.IsZero() && !f.Until.IsZero() && !f.From.Before(f.Until) {
			invalid()
			return
		}
		if v := q.Get("warnings"); v != "" {
			if v != "true" && v != "false" {
				invalid()
				return
			}
			f.WarningsOnly = v == "true"
		}
		page, err := database.BrowseHistory(r.Context(), db, f)
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "Historie derzeit nicht verfügbar. Bitte erneut versuchen."})
			return
		}
		writeJSON(w, 200, page)
	}
}
