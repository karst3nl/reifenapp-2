package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/karst3nl/reifenapp-2/internal/models"
)

type AccessoryOptions struct {
	Revision  int                 `json:"revision"`
	Locations map[string][]string `json:"locations"`
}

var ErrAccessoryOptions = errors.New("Für jedes Zubehör sind 1–50 eindeutige Aufbewahrungsorte mit jeweils 1–200 Zeichen ohne Steuerzeichen erforderlich.")
var ErrAccessoryRevision = errors.New("Zubehörlisten wurden inzwischen geändert. Bitte neu laden.")

type AccessoryFieldsError struct{ Fields map[string]string }

func (e *AccessoryFieldsError) Error() string {
	return "Bitte die Aufbewahrungsorte aus den aktuellen Zubehörlisten auswählen."
}
func accessoryOptions(ctx context.Context, q batchQuerier) (AccessoryOptions, error) {
	var c AccessoryOptions
	var raw string
	err := q.QueryRowContext(ctx, "SELECT revision,locations FROM accessory_options WHERE id=1").Scan(&c.Revision, &raw)
	if err != nil {
		return c, err
	}
	err = json.Unmarshal([]byte(raw), &c.Locations)
	return c, err
}
func LoadAccessoryOptions(ctx context.Context, db *sql.DB) (AccessoryOptions, error) {
	return accessoryOptions(ctx, db)
}
func SaveAccessoryOptions(ctx context.Context, db *sql.DB, c AccessoryOptions) error {
	if len(c.Locations) != 3 {
		return ErrAccessoryOptions
	}
	for _, key := range []string{"hubcaps", "bolts", "lock"} {
		values := c.Locations[key]
		if len(values) < 1 || len(values) > 50 {
			return ErrAccessoryOptions
		}
		seen := map[string]bool{}
		for i, value := range values {
			value = strings.TrimSpace(value)
			if value == "" || utf8.RuneCountInString(value) > 200 {
				return ErrAccessoryOptions
			}
			for _, r := range value {
				if unicode.IsControl(r) {
					return ErrAccessoryOptions
				}
			}
			folded := strings.ToLower(value)
			if seen[folded] {
				return ErrAccessoryOptions
			}
			seen[folded] = true
			values[i] = value
		}
	}
	raw, err := json.Marshal(c.Locations)
	if err != nil {
		return err
	}
	result, err := db.ExecContext(ctx, "UPDATE accessory_options SET locations=?,revision=revision+1 WHERE id=1 AND revision=?", string(raw), c.Revision)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrAccessoryRevision
	}
	return nil
}
func validateAccessoryLocations(ctx context.Context, tx *sql.Tx, input models.StorageInput) error {
	options, err := accessoryOptions(ctx, tx)
	if err != nil {
		return err
	}
	fields := map[string]string{}
	for key, a := range map[string]models.Accessory{"hubcaps": input.Hubcaps, "bolts": input.Bolts, "lock": input.Lock} {
		if !a.Present {
			continue
		}
		found := false
		for _, location := range options.Locations[key] {
			if a.Location == location {
				found = true
				break
			}
		}
		if !found {
			fields[key+".location"] = "Bitte einen aktuellen Aufbewahrungsort auswählen. Die Liste wurde möglicherweise geändert."
		}
	}
	if len(fields) > 0 {
		return &AccessoryFieldsError{Fields: fields}
	}
	return nil
}
