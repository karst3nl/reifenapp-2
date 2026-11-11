package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/karst3nl/reifenapp-2/internal/database"
	"github.com/karst3nl/reifenapp-2/internal/receipts"
)

func Receipts(db *sql.DB, headers ...receipts.Header) http.HandlerFunc {
	header := receipts.Header{}
	if len(headers) > 0 {
		header = headers[0]
	}
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "private, no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		id, err := strconv.ParseInt(r.PathValue("claim"), 10, 64)
		operation, ok := strings.CutSuffix(r.PathValue("operation"), ".pdf")
		if err != nil || id <= 0 || !ok || (operation != "storage" && operation != "wheel_change" && operation != "reservation" && operation != "release") {
			http.Error(w, "Ungültiger PDF-Beleg.", 400)
			return
		}
		receipt, err := database.Receipt(r.Context(), db, id, operation)
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "Vorgangsbeleg nicht gefunden.", 404)
			return
		}
		if err != nil {
			http.Error(w, "Beleg derzeit nicht abrufbar. Bitte erneut versuchen.", 500)
			return
		}
		data, err := receipts.GenerateWithHeader(receipt, header)
		if err != nil {
			http.Error(w, "PDF konnte nicht erstellt werden. Der Vorgang bleibt gespeichert. Bitte erneut versuchen.", 500)
			return
		}
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="reifenapp-%s-%d.pdf"`, operation, id))
		w.Write(data)
	}
}
