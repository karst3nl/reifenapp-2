package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/karst3nl/reifenapp-2/internal/models"
)

type LabelConfig struct {
	Settings models.LabelSettings `json:"settings"`
	Revision int                  `json:"revision"`
}

func labelConfig(ctx context.Context, q batchQuerier) (LabelConfig, error) {
	var result LabelConfig
	var raw string
	err := q.QueryRowContext(ctx, "SELECT config,revision FROM label_printer_settings WHERE id=1").Scan(&raw, &result.Revision)
	if err != nil {
		return result, err
	}
	if err = json.Unmarshal([]byte(raw), &result.Settings); err != nil {
		return result, err
	}
	if result.Settings.Version == 0 {
		result.Settings = models.DefaultLabelSettings()
	}
	return result, nil
}
func LabelConfiguration(ctx context.Context, db *sql.DB) (LabelConfig, error) {
	return labelConfig(ctx, db)
}

var ErrLabelRevision = errors.New("Druckerkonfiguration wurde inzwischen geändert. Bitte neu laden.")

func SaveLabelConfiguration(ctx context.Context, db *sql.DB, c LabelConfig) error {
	if err := c.Settings.Validate(); err != nil {
		return err
	}
	raw, err := json.Marshal(c.Settings)
	if err != nil {
		return err
	}
	result, err := db.ExecContext(ctx, "UPDATE label_printer_settings SET config=?,revision=revision+1 WHERE id=1 AND revision=?", string(raw), c.Revision)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrLabelRevision
	}
	return nil
}
func enqueueLabel(ctx context.Context, tx *sql.Tx, claim int64, operation string) error {
	c, err := labelConfig(ctx, tx)
	if err != nil {
		return err
	}
	s := c.Settings
	if !s.Enabled || (operation == "storage" && !s.Storage) || (operation == "wheel_change" && !s.WheelChange) {
		return nil
	}
	var r models.Receipt
	var payload string
	err = tx.QueryRowContext(ctx, "SELECT id,claim_id,operation,occurred_at,place,name,plate,car,source,payload FROM history_events WHERE claim_id=? AND operation=?", claim, operation).Scan(&r.EventID, &r.ClaimID, &r.Operation, &r.Date, &r.Place, &r.Name, &r.Plate, &r.Car, &r.Source, &payload)
	if err != nil {
		return err
	}
	r.Payload = []byte(payload)
	_, err = insertLabelJob(ctx, tx, s, r)
	return err
}
func insertLabelJob(ctx context.Context, tx *sql.Tx, s models.LabelSettings, r models.Receipt) (int64, error) {
	config, err := json.Marshal(s)
	if err != nil {
		return 0, err
	}
	receipt, err := json.Marshal(r)
	if err != nil {
		return 0, err
	}
	var event any
	if r.EventID > 0 {
		event = r.EventID
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	result, err := tx.ExecContext(ctx, "INSERT INTO label_print_jobs(event_id,receipt,config,status,created_at,updated_at) VALUES(?,?,?,'pending',?,?)", event, string(receipt), string(config), now, now)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}
func TestLabelJob(ctx context.Context, db *sql.DB, s models.LabelSettings, r models.Receipt) (int64, error) {
	r.EventID = 0
	r.ClaimID = 0
	if err := s.Validate(); err != nil {
		return 0, err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	id, err := insertLabelJob(ctx, tx, s, r)
	if err != nil {
		return 0, err
	}
	return id, tx.Commit()
}

type LabelJob struct {
	ID        int64                `json:"id"`
	EventID   int64                `json:"event_id"`
	Status    string               `json:"status"`
	CreatedAt string               `json:"created_at"`
	UpdatedAt string               `json:"updated_at"`
	CUPSID    string               `json:"cups_id"`
	Message   string               `json:"message"`
	Attempts  int                  `json:"attempts"`
	Receipt   models.Receipt       `json:"-"`
	Settings  models.LabelSettings `json:"-"`
}

func scanLabelJob(row interface{ Scan(...any) error }) (LabelJob, error) {
	var j LabelJob
	var receipt, config string
	err := row.Scan(&j.ID, &j.EventID, &j.Status, &j.CreatedAt, &j.UpdatedAt, &j.CUPSID, &j.Message, &j.Attempts, &receipt, &config)
	if err != nil {
		return j, err
	}
	if err = json.Unmarshal([]byte(receipt), &j.Receipt); err != nil {
		return j, err
	}
	err = json.Unmarshal([]byte(config), &j.Settings)
	return j, err
}

const labelColumns = "id,COALESCE(event_id,0),status,created_at,updated_at,cups_id,message,attempts,receipt,config"

func LabelJobs(ctx context.Context, db *sql.DB) ([]LabelJob, error) {
	rows, err := db.QueryContext(ctx, "SELECT "+labelColumns+" FROM label_print_jobs ORDER BY id DESC LIMIT 100")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []LabelJob{}
	for rows.Next() {
		j, err := scanLabelJob(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, j)
	}
	return result, rows.Err()
}
func LabelJobStatus(ctx context.Context, db *sql.DB, id int64) (LabelJob, error) {
	return scanLabelJob(db.QueryRowContext(ctx, "SELECT "+labelColumns+" FROM label_print_jobs WHERE id=?", id))
}
func LabelJobForClaim(ctx context.Context, db *sql.DB, claim int64, operation string) (int64, error) {
	var id int64
	err := db.QueryRowContext(ctx, "SELECT j.id FROM label_print_jobs j JOIN history_events h ON h.id=j.event_id WHERE h.claim_id=? AND h.operation=?", claim, operation).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return id, err
}
func ClaimLabelJob(ctx context.Context, db *sql.DB) (LabelJob, error) {
	return scanLabelJob(db.QueryRowContext(ctx, "UPDATE label_print_jobs SET status='processing',attempts=attempts+1,updated_at=? WHERE id=(SELECT id FROM label_print_jobs WHERE status='pending' ORDER BY id LIMIT 1) AND status='pending' RETURNING "+labelColumns, time.Now().UTC().Format(time.RFC3339Nano)))
}
func FinishLabelJob(ctx context.Context, db *sql.DB, id int64, status, cupsID, message string) error {
	_, err := db.ExecContext(ctx, "UPDATE label_print_jobs SET status=?,cups_id=?,message=?,updated_at=? WHERE id=? AND status='processing'", status, cupsID, message, time.Now().UTC().Format(time.RFC3339Nano), id)
	return err
}
func RecoverLabelJobs(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, "UPDATE label_print_jobs SET status='uncertain',message='Server während Druckübergabe beendet. Vor erneutem Druck CUPS und Drucker prüfen.' WHERE status='processing'")
	return err
}
func UpdateLabelJob(ctx context.Context, db *sql.DB, id int64, retry bool) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	query := "UPDATE label_print_jobs SET status='cancelled',message='Ausstehender Auftrag abgebrochen.',updated_at=? WHERE id=? AND status='pending'"
	args := []any{time.Now().UTC().Format(time.RFC3339Nano), id}
	if retry {
		c, err := labelConfig(ctx, tx)
		if err != nil {
			return err
		}
		if err = c.Settings.Validate(); err != nil {
			return err
		}
		if c.Settings.Queue == "" {
			return errors.New("Drucker zuerst konfigurieren.")
		}
		config, err := json.Marshal(c.Settings)
		if err != nil {
			return err
		}
		query = "UPDATE label_print_jobs SET status='pending',message='',cups_id='',config=?,updated_at=? WHERE id=? AND status IN ('failed','uncertain')"
		args = []any{string(config), time.Now().UTC().Format(time.RFC3339Nano), id}
	}
	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return errors.New("Auftragsstatus hat sich geändert oder Aktion ist nicht zulässig.")
	}
	return tx.Commit()
}
