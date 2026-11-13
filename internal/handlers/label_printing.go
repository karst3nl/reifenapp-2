package handlers

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/karst3nl/reifenapp-2/internal/database"
	"github.com/karst3nl/reifenapp-2/internal/labels"
	"github.com/karst3nl/reifenapp-2/internal/models"
	"github.com/karst3nl/reifenapp-2/internal/receipts"
)

func (a *Admin) StartLabelPrinter(ctx context.Context) (context.CancelFunc, error) {
	return a.printer.Start(ctx)
}
func (a *Admin) labelConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		var c database.LabelConfig
		if !readStorageJSON(w, r, &c) {
			return
		}
		if err := c.Settings.Validate(); err != nil {
			writeJSON(w, 422, map[string]string{"error": err.Error()})
			return
		}
		if c.Settings.Enabled {
			if err := a.printer.Check(r.Context(), c.Settings.Queue); err != nil {
				writeJSON(w, 422, map[string]string{"error": err.Error()})
				return
			}
		}
		err := database.SaveLabelConfiguration(r.Context(), a.db, c)
		if errors.Is(err, database.ErrLabelRevision) {
			writeJSON(w, 409, map[string]string{"error": err.Error()})
			return
		}
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "Druckereinstellungen konnten nicht gespeichert werden."})
			return
		}
	}
	c, err := database.LabelConfiguration(r.Context(), a.db)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "Druckereinstellungen derzeit nicht verfügbar."})
		return
	}
	writeJSON(w, 200, map[string]any{"config": c, "fields": models.LabelFields, "default": models.DefaultLabelSettings()})
}
func (a *Admin) labelPrinters(w http.ResponseWriter, r *http.Request) {
	printers, err := a.printer.Printers(r.Context())
	if err != nil {
		writeJSON(w, 200, map[string]any{"available": false, "printers": []any{}, "message": err.Error()})
		return
	}
	writeJSON(w, 200, map[string]any{"available": true, "printers": printers, "message": "USB-Warteschlangen des App-Servers. Physische Verbindung und Druckerzustand vor Ort prüfen."})
}
func (a *Admin) labelPreview(w http.ResponseWriter, r *http.Request) {
	var s models.LabelSettings
	if !readStorageJSON(w, r, &s) {
		return
	}
	data, err := labels.PDF(s, receipts.ExampleReceipt("storage"))
	if err != nil {
		writeJSON(w, 422, map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Content-Disposition", "inline; filename=etikett-vorschau.pdf")
	w.Write(data)
}
func (a *Admin) labelTest(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Confirmed bool `json:"confirmed"`
	}
	if !readStorageJSON(w, r, &input) {
		return
	}
	if !input.Confirmed {
		writeJSON(w, 422, map[string]string{"error": "Testdruck bitte bestätigen."})
		return
	}
	c, err := database.LabelConfiguration(r.Context(), a.db)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "Einstellungen nicht verfügbar."})
		return
	}
	if err = a.printer.Check(r.Context(), c.Settings.Queue); err != nil {
		writeJSON(w, 422, map[string]string{"error": err.Error()})
		return
	}
	// Test always prints one fictional label, regardless of configured production copies.
	c.Settings.Copies = 1
	rct := receipts.ExampleReceipt("storage")
	rct.EventID = 0
	rct.ClaimID = 0
	id, err := database.TestLabelJob(r.Context(), a.db, c.Settings, rct)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "Testdruck nicht vorgemerkt."})
		return
	}
	writeJSON(w, 202, map[string]any{"id": id})
}
func (a *Admin) labelJobs(w http.ResponseWriter, r *http.Request) {
	jobs, err := database.LabelJobs(r.Context(), a.db)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "Druckaufträge derzeit nicht verfügbar."})
		return
	}
	writeJSON(w, 200, jobs)
}
func (a *Admin) labelJobAction(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		writeJSON(w, 400, map[string]string{"error": "Ungültiger Auftrag."})
		return
	}
	var input struct {
		Action    string `json:"action"`
		Confirmed bool   `json:"confirmed"`
	}
	if !readStorageJSON(w, r, &input) {
		return
	}
	if (input.Action != "retry" && input.Action != "cancel") || !input.Confirmed {
		writeJSON(w, 422, map[string]string{"error": "Aktion bitte ausdrücklich bestätigen. Bei erneutem Druck zuvor CUPS und Drucker prüfen."})
		return
	}
	if err = database.UpdateLabelJob(r.Context(), a.db, id, input.Action == "retry"); err != nil {
		writeJSON(w, 409, map[string]string{"error": "Auftrag nicht geändert. Status neu laden."})
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}
func LabelJobStatus(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil || id < 1 {
			writeJSON(w, 400, map[string]string{"error": "Ungültiger Auftrag."})
			return
		}
		j, err := database.LabelJobStatus(r.Context(), db, id)
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, 404, map[string]string{"error": "Auftrag nicht gefunden."})
			return
		}
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "Druckstatus derzeit nicht verfügbar."})
			return
		}
		writeJSON(w, 200, map[string]any{"id": j.ID, "status": j.Status, "message": j.Message, "cups_id": j.CUPSID})
	}
}
