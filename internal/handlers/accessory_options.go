package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/karst3nl/reifenapp-2/internal/database"
	"github.com/karst3nl/reifenapp-2/internal/models"
)

// Preserve historical bool-based accessory records; require explicit answers on new HTTP requests.
type storageRequest struct {
	Input   models.StorageInput
	Answers map[string]bool
}

func (r *storageRequest) UnmarshalJSON(raw []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&r.Input); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return err
	}
	r.Answers = map[string]bool{}
	for _, key := range []string{"hubcaps", "bolts", "lock"} {
		var accessory map[string]json.RawMessage
		if err := json.Unmarshal(fields[key], &accessory); err != nil {
			continue
		}
		value := string(bytes.TrimSpace(accessory["present"]))
		r.Answers[key] = value == "true" || value == "false"
	}
	return nil
}
func (a *Admin) accessoryOptions(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		var c database.AccessoryOptions
		if !readLimitedJSON(w, r, &c, 128*1024) {
			return
		}
		err := database.SaveAccessoryOptions(r.Context(), a.db, c)
		if errors.Is(err, database.ErrAccessoryOptions) {
			writeJSON(w, 422, map[string]string{"error": err.Error()})
			return
		}
		if errors.Is(err, database.ErrAccessoryRevision) {
			writeJSON(w, 409, map[string]string{"error": err.Error()})
			return
		}
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "Zubehörlisten konnten nicht gespeichert werden."})
			return
		}
	}
	c, err := database.LoadAccessoryOptions(r.Context(), a.db)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "Zubehörlisten derzeit nicht verfügbar."})
		return
	}
	writeJSON(w, 200, c)
}
