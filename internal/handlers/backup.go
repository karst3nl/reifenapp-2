package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/karst3nl/reifenapp-2/internal/database"
)

func (a *Admin) backupInfo(w http.ResponseWriter, r *http.Request) {
	info, err := database.ReadBackupInfo(r.Context(), a.db)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "Datenbankinformationen derzeit nicht verfügbar."})
		return
	}
	writeJSON(w, 200, info)
}
func (a *Admin) backup(w http.ResponseWriter, r *http.Request) {
	var input struct{}
	if !readStorageJSON(w, r, &input) {
		return
	}
	if !a.backupMu.TryLock() {
		w.Header().Set("Retry-After", "5")
		writeJSON(w, 429, map[string]string{"error": "Eine Sicherung wird bereits erstellt. Bitte kurz warten."})
		return
	}
	defer a.backupMu.Unlock()
	// Backups may need longer than ordinary API calls; no global timeout changes.
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(5 * time.Minute))
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()
	header, err := json.MarshalIndent(a.header, "", "  ")
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "Firmenkopf konnte nicht gesichert werden."})
		return
	}
	backup, err := database.CreateBackup(ctx, a.db, header)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "Backup konnte nicht erstellt werden. Bitte freien Speicherplatz prüfen und erneut versuchen."})
		return
	}
	defer backup.Close()
	file, err := os.Open(backup.Path)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "Backup konnte nicht bereitgestellt werden."})
		return
	}
	defer file.Close()
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+backup.Filename+"\"")
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeContent(w, r, backup.Filename, backup.Manifest.CreatedAt, file)
}
