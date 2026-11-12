package handlers

import (
	"errors"
	"github.com/karst3nl/reifenapp-2/internal/database"
	"net/http"
)

func (a *Admin) areas(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		var input database.StorageArea
		if !readStorageJSON(w, r, &input) {
			return
		}
		err := database.SaveStorageArea(r.Context(), a.db, input)
		if errors.Is(err, database.ErrAreaInput) {
			writeJSON(w, 422, map[string]string{"error": err.Error()})
			return
		}
		if errors.Is(err, database.ErrAreaConflict) || errors.Is(err, database.ErrAreaOccupied) {
			writeJSON(w, 409, map[string]string{"error": err.Error()})
			return
		}
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "Lagerbereich konnte nicht gespeichert werden."})
			return
		}
	}
	areas, err := database.StorageAreas(r.Context(), a.db)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "Lagerbereiche nicht verfügbar."})
		return
	}
	writeJSON(w, 200, areas)
}
