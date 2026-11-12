package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/karst3nl/reifenapp-2/internal/receipts"
)

var ErrLayoutConflict = errors.New("Die Vorlage wurde inzwischen geändert. Bitte neu laden.")

type PDFLayout struct {
	Operation string            `json:"operation"`
	Template  receipts.Template `json:"template"`
	Enabled   bool              `json:"enabled"`
	Revision  int               `json:"revision"`
	UpdatedAt string            `json:"updated_at"`
}

func LoadPDFLayout(ctx context.Context, db *sql.DB, operation string) (PDFLayout, error) {
	result := PDFLayout{Operation: operation, Template: receipts.DefaultTemplate(operation)}
	var data string
	err := db.QueryRowContext(ctx, "SELECT layout,enabled,revision,updated_at FROM pdf_layouts WHERE operation=?", operation).Scan(&data, &result.Enabled, &result.Revision, &result.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return result, nil
	}
	if err != nil {
		return result, err
	}
	err = json.Unmarshal([]byte(data), &result.Template)
	return result, err
}
func SavePDFLayout(ctx context.Context, db *sql.DB, layout PDFLayout) (PDFLayout, error) {
	if !receipts.ValidOperation(layout.Operation) {
		return layout, errors.New("Ungültige Vorgangsart.")
	}
	if err := layout.Template.Validate(); err != nil {
		return layout, err
	}
	if layout.Revision < 0 {
		return layout, ErrLayoutConflict
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return layout, err
	}
	defer tx.Rollback()
	var revision int
	err = tx.QueryRowContext(ctx, "SELECT revision FROM pdf_layouts WHERE operation=?", layout.Operation).Scan(&revision)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return layout, err
	}
	if revision != layout.Revision {
		return layout, ErrLayoutConflict
	}
	data, err := json.Marshal(layout.Template)
	if err != nil {
		return layout, err
	}
	layout.Revision++
	layout.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	_, err = tx.ExecContext(ctx, `INSERT INTO pdf_layouts(operation,layout,enabled,revision,updated_at) VALUES(?,?,?,?,?) ON CONFLICT(operation) DO UPDATE SET layout=excluded.layout,enabled=excluded.enabled,revision=excluded.revision,updated_at=excluded.updated_at`, layout.Operation, string(data), layout.Enabled, layout.Revision, layout.UpdatedAt)
	if err != nil {
		return layout, err
	}
	return layout, tx.Commit()
}
