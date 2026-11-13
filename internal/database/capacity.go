package database

import (
	"context"
	"database/sql"
)

type Capacity struct {
	Total    int `json:"total"`
	Reserved int `json:"reserved"`
	Occupied int `json:"occupied"`
	Free     int `json:"free"`
}

// Count enabled places once, excluding historical and inactive claims.
func StorageCapacity(ctx context.Context, db *sql.DB) (Capacity, error) {
	var result Capacity
	err := db.QueryRowContext(ctx, `SELECT count(*),
 COALESCE(sum(CASE WHEN c.kind='reservation' THEN 1 ELSE 0 END),0),
 COALESCE(sum(CASE WHEN c.kind='occupancy' THEN 1 ELSE 0 END),0),
 COALESCE(sum(CASE WHEN c.id IS NULL THEN 1 ELSE 0 END),0)
 FROM storage_places p LEFT JOIN storage_claims c ON c.place_id=p.id AND c.status='active'
 WHERE p.enabled=1`).Scan(&result.Total, &result.Reserved, &result.Occupied, &result.Free)
	return result, err
}
