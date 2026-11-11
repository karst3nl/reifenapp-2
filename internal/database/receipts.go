package database

import (
	"context"
	"database/sql"

	"github.com/karst3nl/reifenapp-2/internal/models"
)

func Receipt(ctx context.Context, db *sql.DB, claimID int64, operation string) (models.Receipt, error) {
	var r models.Receipt
	var payload string
	err := db.QueryRowContext(ctx, `SELECT id,claim_id,operation,occurred_at,place,name,plate,car,source,payload
 FROM history_events WHERE claim_id=? AND operation=?`, claimID, operation).Scan(&r.EventID, &r.ClaimID, &r.Operation, &r.Date, &r.Place, &r.Name, &r.Plate, &r.Car, &r.Source, &payload)
	r.Payload = []byte(payload)
	return r, err
}
