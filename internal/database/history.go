package database

import (
	"context"
	"database/sql"
	"fmt"
)

func recordHistory(ctx context.Context, tx *sql.Tx, claimID int64, operation, stamp string) error {
	result, err := tx.ExecContext(ctx, `INSERT INTO history_events(claim_id,operation,occurred_at,place,name,plate,car,payload,source)
 SELECT claim_id,?,?,place,name,plate,car,payload,'recorded' FROM claim_history_snapshot WHERE claim_id=?`, operation, stamp, claimID)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("history snapshot missing for claim %d", claimID)
	}
	return nil
}
