package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var ErrReleaseChanged = errors.New("Die Belegung hat sich geändert oder ist nicht mehr aktiv. Bitte den Lagerplatz erneut prüfen.")

type ReleaseTarget struct {
	Place   string `json:"place"`
	Status  string `json:"status"`
	ClaimID int64  `json:"claim_id,omitempty"`
	Name    string `json:"name,omitempty"`
	Plate   string `json:"plate,omitempty"`
	Car     string `json:"car,omitempty"`
	Message string `json:"message"`
}

func LookupRelease(ctx context.Context, db *sql.DB, area string, number int) (ReleaseTarget, error) {
	return lookupRelease(ctx, db, area, number)
}
func lookupRelease(ctx context.Context, q rowQuerier, area string, number int) (ReleaseTarget, error) {
	result := ReleaseTarget{Place: fmt.Sprintf("%s.%d", area, number)}
	var id sql.NullInt64
	var kind, name, plate, car sql.NullString
	err := q.QueryRowContext(ctx, `SELECT s.id,s.kind,c.name,v.plate,v.car FROM storage_places p
 LEFT JOIN storage_claims s ON s.place_id=p.id AND s.status='active'
 LEFT JOIN tire_sets t ON t.id=s.tire_set_id LEFT JOIN vehicles v ON v.id=t.vehicle_id
 LEFT JOIN customers c ON c.id=v.customer_id WHERE p.id=? AND p.enabled=1`, result.Place).Scan(&id, &kind, &name, &plate, &car)
	if errors.Is(err, sql.ErrNoRows) {
		return result, ErrInvalidPlace
	}
	if err != nil {
		return result, err
	}
	if !id.Valid {
		result.Status = "free"
		result.Message = "Lagerplatz " + result.Place + " ist bereits frei. Keine Auslagerung möglich."
		return result, nil
	}
	if kind.String == "reservation" {
		result.Status = "reserved"
		result.Message = "Lagerplatz " + result.Place + " ist reserviert. Reservierungen können hier nicht ausgelagert werden."
		return result, nil
	}
	result.Status = "occupied"
	result.ClaimID = id.Int64
	result.Name = name.String
	result.Plate = plate.String
	result.Car = car.String
	result.Message = "Bitte Kunde und Kennzeichen prüfen und die Auslagerung bestätigen."
	return result, nil
}
func ReleaseStorage(ctx context.Context, db *sql.DB, area string, number int, claimID int64) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	target, err := lookupRelease(ctx, tx, area, number)
	if err != nil {
		return err
	}
	if target.Status != "occupied" || claimID <= 0 || target.ClaimID != claimID {
		return ErrReleaseChanged
	}
	if err = releaseClaim(ctx, tx, claimID, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		return err
	}
	return tx.Commit()
}

// releaseClaim records an explicitly confirmed Auslagerung in its caller's transaction.
func releaseClaim(ctx context.Context, tx *sql.Tx, claimID int64, stamp string) error {
	result, err := tx.ExecContext(ctx, "UPDATE storage_claims SET status='released' WHERE id=? AND kind='occupancy' AND status='active'", claimID)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrReleaseChanged
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO storage_releases(claim_id,released_at) VALUES(?,?)", claimID, stamp); err != nil {
		return err
	}
	if err = recordHistory(ctx, tx, claimID, "release", stamp); err != nil {
		return err
	}
	return nil
}
