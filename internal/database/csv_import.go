package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/karst3nl/reifenapp-2/internal/importcsv"
	"github.com/karst3nl/reifenapp-2/internal/models"
)

type CSVBatch struct {
	ID          int64  `json:"id"`
	Filename    string `json:"filename"`
	ImportedAt  string `json:"imported_at"`
	Encoding    string `json:"encoding"`
	Delimiter   string `json:"delimiter"`
	Rows        int    `json:"rows"`
	Imported    int    `json:"imported"`
	Duplicates  int    `json:"duplicates"`
	WarningRows int    `json:"warning_rows"`
}
type CSVPreview struct {
	Hash            string          `json:"hash"`
	Filename        string          `json:"filename"`
	Encoding        string          `json:"encoding"`
	Delimiter       string          `json:"delimiter"`
	Rows            int             `json:"rows"`
	New             int             `json:"new"`
	Duplicates      int             `json:"duplicates"`
	WarningRows     int             `json:"warning_rows"`
	AlreadyImported bool            `json:"already_imported"`
	Preview         []importcsv.Row `json:"preview"`
}
type CSVImportResult struct {
	CSVBatch
	AlreadyImported bool   `json:"already_imported"`
	Backup          string `json:"backup"`
}

func PreviewCSV(ctx context.Context, db *sql.DB, doc *importcsv.Document, filename string) (CSVPreview, error) {
	result := CSVPreview{Hash: doc.Hash, Filename: filename, Encoding: doc.Encoding, Delimiter: doc.Delimiter, Rows: len(doc.Rows), Preview: []importcsv.Row{}}
	areas, err := StorageAreas(ctx, db)
	if err != nil {
		return result, err
	}
	for i := range doc.Rows {
		row := &doc.Rows[i]
		validPlace := false
		split := strings.LastIndex(row.Place, ".")
		if split > 0 {
			number, e := strconv.Atoi(row.Place[split+1:])
			if e == nil {
				for _, area := range areas {
					if area.Code == row.Place[:split] && number >= area.First && number <= area.Last {
						validPlace = true
					}
				}
			}
		}
		if !validPlace {
			warning := "Historischer Lagerplatz ist im aktuellen Lagerkatalog nicht vorhanden. Es wird keine Belegung angelegt."
			found := false
			for _, existing := range row.Warnings {
				if existing == warning {
					found = true
				}
			}
			if !found {
				row.Warnings = append(row.Warnings, warning)
			}
		}
	}
	var existing int
	if err = db.QueryRowContext(ctx, "SELECT count(*) FROM csv_import_batches WHERE file_sha256=?", doc.Hash).Scan(&existing); err != nil {
		return result, err
	}
	result.AlreadyImported = existing > 0
	fingerprints := map[string]bool{}
	for _, row := range doc.Rows {
		var known int
		if err = db.QueryRowContext(ctx, "SELECT count(*) FROM history_events WHERE import_key=?", row.Fingerprint).Scan(&known); err != nil {
			return result, err
		}
		if known > 0 || fingerprints[row.Fingerprint] {
			result.Duplicates++
		} else {
			result.New++
		}
		fingerprints[row.Fingerprint] = true
		if len(row.Warnings) > 0 {
			result.WarningRows++
		}
		if len(result.Preview) < 50 {
			result.Preview = append(result.Preview, row)
		}
	}
	return result, nil
}

func ImportCSV(ctx context.Context, db *sql.DB, doc *importcsv.Document, filename string) (CSVImportResult, error) {
	var result CSVImportResult
	existing, err := findCSVBatch(ctx, db, doc.Hash)
	if err == nil {
		return CSVImportResult{CSVBatch: existing, AlreadyImported: true}, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return result, err
	}
	// Back up the untouched database before adding any archive records.
	backup, err := backupBeforeCSV(ctx, db)
	if err != nil {
		return result, err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return result, err
	}
	defer tx.Rollback()
	existing, err = findCSVBatch(ctx, tx, doc.Hash)
	if err == nil {
		return CSVImportResult{CSVBatch: existing, AlreadyImported: true, Backup: filepath.Base(backup)}, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return result, err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	headers, _ := json.Marshal(doc.Headers)
	inserted, err := tx.ExecContext(ctx, "INSERT INTO csv_import_batches(file_sha256,filename,imported_at,encoding,delimiter,headers,row_count,imported_count,duplicate_count,warning_count,raw_csv) VALUES(?,?,?,?,?,?,?,0,0,0,?)", doc.Hash, filename, now, doc.Encoding, doc.Delimiter, string(headers), len(doc.Rows), doc.Raw)
	if err != nil {
		return result, err
	}
	batchID, err := inserted.LastInsertId()
	if err != nil {
		return result, err
	}
	result.CSVBatch = CSVBatch{ID: batchID, Filename: filename, ImportedAt: now, Encoding: doc.Encoding, Delimiter: doc.Delimiter, Rows: len(doc.Rows)}
	result.Backup = filepath.Base(backup)
	for _, row := range doc.Rows {
		details := row.Details
		details.Import = &models.CSVOrigin{Filename: filename, Line: row.Line, ImportedAt: now, Fields: row.Fields, Warnings: row.Warnings}
		for _, field := range row.Fields {
			if strings.EqualFold(strings.TrimSpace(field.Column), "TÄTIGKEIT") {
				details.Import.Operation = field.Value
			}
		}
		payload, err := json.Marshal(details)
		if err != nil {
			return result, err
		}
		var eventID int64
		err = tx.QueryRowContext(ctx, "INSERT INTO history_events(claim_id,operation,occurred_at,place,name,plate,car,payload,source,import_key) VALUES(NULL,?,?,?,?,?,?,?,'csv',?) ON CONFLICT(import_key) DO NOTHING RETURNING id", row.Operation, row.Date, row.Place, row.Name, row.Plate, row.Car, string(payload), row.Fingerprint).Scan(&eventID)
		if errors.Is(err, sql.ErrNoRows) {
			if err = tx.QueryRowContext(ctx, "SELECT id FROM history_events WHERE import_key=?", row.Fingerprint).Scan(&eventID); err != nil {
				return result, err
			}
			result.Duplicates++
		} else if err != nil {
			return result, err
		} else {
			result.Imported++
		}
		values, _ := json.Marshal(row.Values)
		warnings, _ := json.Marshal(row.Warnings)
		if _, err = tx.ExecContext(ctx, "INSERT INTO csv_import_rows(batch_id,event_id,line_number,original_values,warnings) VALUES(?,?,?,?,?)", batchID, eventID, row.Line, string(values), string(warnings)); err != nil {
			return result, err
		}
		if len(row.Warnings) > 0 {
			result.WarningRows++
		}
	}
	if _, err = tx.ExecContext(ctx, "UPDATE csv_import_batches SET imported_count=?,duplicate_count=?,warning_count=? WHERE id=?", result.Imported, result.Duplicates, result.WarningRows, batchID); err != nil {
		return result, err
	}
	if err = tx.Commit(); err != nil {
		return result, err
	}
	return result, nil
}

type batchQuerier interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func findCSVBatch(ctx context.Context, q batchQuerier, hash string) (CSVBatch, error) {
	var batch CSVBatch
	err := q.QueryRowContext(ctx, "SELECT id,filename,imported_at,encoding,delimiter,row_count,imported_count,duplicate_count,warning_count FROM csv_import_batches WHERE file_sha256=?", hash).Scan(&batch.ID, &batch.Filename, &batch.ImportedAt, &batch.Encoding, &batch.Delimiter, &batch.Rows, &batch.Imported, &batch.Duplicates, &batch.WarningRows)
	return batch, err
}
func CSVImports(ctx context.Context, db *sql.DB) ([]CSVBatch, error) {
	rows, err := db.QueryContext(ctx, "SELECT id,filename,imported_at,encoding,delimiter,row_count,imported_count,duplicate_count,warning_count FROM csv_import_batches ORDER BY id DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []CSVBatch{}
	for rows.Next() {
		var batch CSVBatch
		if err = rows.Scan(&batch.ID, &batch.Filename, &batch.ImportedAt, &batch.Encoding, &batch.Delimiter, &batch.Rows, &batch.Imported, &batch.Duplicates, &batch.WarningRows); err != nil {
			return nil, err
		}
		result = append(result, batch)
	}
	return result, rows.Err()
}
func OriginalCSV(ctx context.Context, db *sql.DB, id int64) (string, []byte, error) {
	var name string
	var data []byte
	err := db.QueryRowContext(ctx, "SELECT filename,raw_csv FROM csv_import_batches WHERE id=?", id).Scan(&name, &data)
	return name, data, err
}
func backupBeforeCSV(ctx context.Context, db *sql.DB) (string, error) {
	rows, err := db.QueryContext(ctx, "PRAGMA database_list")
	if err != nil {
		return "", err
	}
	var path string
	for rows.Next() {
		var seq int
		var name, file string
		if err = rows.Scan(&seq, &name, &file); err != nil {
			rows.Close()
			return "", err
		}
		if name == "main" {
			path = file
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return "", err
	}
	if path == "" {
		return "", fmt.Errorf("Dateibasierte Datenbank erforderlich")
	}
	backup := path + ".backup-csv-" + time.Now().UTC().Format("20060102T150405.000000000Z")
	if _, err = db.ExecContext(ctx, "VACUUM INTO ?", backup); err != nil {
		return "", err
	}
	if err = os.Chmod(backup, 0600); err != nil {
		return "", err
	}
	u := url.URL{Scheme: "file", Path: backup}
	copyDB, err := sql.Open("sqlite3", u.String()+"?mode=ro&immutable=1")
	if err != nil {
		return "", err
	}
	defer copyDB.Close()
	var check string
	if err = copyDB.QueryRowContext(ctx, "PRAGMA integrity_check").Scan(&check); err != nil {
		return "", err
	}
	if check != "ok" {
		return "", fmt.Errorf("Backup-Integrität fehlgeschlagen")
	}
	return backup, nil
}
