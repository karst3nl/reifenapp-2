package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/karst3nl/reifenapp-2/internal/models"
	"github.com/mattn/go-sqlite3"
)

var ErrPlaceUnavailable = errors.New("Lagerplatz ist bereits belegt oder aktiv reserviert.")
var ErrInvalidStorage = errors.New("Ungültige Einlagerungsdaten.")
var ErrWheelChangeOwner = errors.New("Kundenname und Kennzeichen müssen zur bestehenden Belegung dieses Lagerplatzes passen.")
var ErrWheelChangeSameType = errors.New("Dieser Reifentyp ist bereits eingelagert. Bitte die Daten des anderen Radsatzes eingeben.")

func CreateStorage(ctx context.Context, db *sql.DB, input models.StorageInput) (int64, error) {
	return createStorage(ctx, db, input, "storage")
}
func CreateWheelChange(ctx context.Context, db *sql.DB, input models.StorageInput) (int64, error) {
	return createStorage(ctx, db, input, "wheel_change")
}
func createStorage(ctx context.Context, db *sql.DB, input models.StorageInput, operation string) (int64, error) {
	if len(input.Validate()) > 0 {
		return 0, ErrInvalidStorage
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	if err = validateAccessoryLocations(ctx, tx, input); err != nil {
		return 0, err
	}
	insertID := func(query string, args ...any) (int64, error) {
		r, e := tx.ExecContext(ctx, query, args...)
		if e != nil {
			return 0, e
		}
		return r.LastInsertId()
	}
	checked, err := checkPlace(ctx, tx, input.Area, input.Number, input.Name, input.Plate)
	if err != nil {
		return 0, err
	}
	swap := operation == "wheel_change" && checked.Status == "occupied"
	if swap && !checked.Matches {
		return 0, fmt.Errorf("%w: %w", ErrPlaceUnavailable, ErrWheelChangeOwner)
	}
	if !checked.Allowed && !swap {
		return 0, ErrPlaceUnavailable
	}
	stamp := time.Now().UTC().Format(time.RFC3339Nano)
	var tire int64
	if swap {
		var previousType string
		err = tx.QueryRowContext(ctx, "SELECT tire_type FROM storage_details WHERE claim_id=?", checked.claimID).Scan(&previousType)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return 0, err
		}
		if previousType != "" && previousType == input.TireType {
			return 0, fmt.Errorf("%w: %w", ErrPlaceUnavailable, ErrWheelChangeSameType)
		}
		// Retire only the old occupancy. A wheel change is not an Auslagerung:
		// no storage_releases entry and no separate release history/PDF.
		retired, retireErr := tx.ExecContext(ctx, "UPDATE storage_claims SET status='released' WHERE id=? AND kind='occupancy' AND status='active'", checked.claimID)
		if retireErr != nil {
			return 0, retireErr
		}
		affected, retireErr := retired.RowsAffected()
		if retireErr != nil {
			return 0, retireErr
		}
		if affected != 1 {
			return 0, ErrReleaseChanged
		}
		// Reuse the returning seasonal set for this vehicle and place when known.
		err = tx.QueryRowContext(ctx, `SELECT t.id FROM tire_sets t
   JOIN storage_claims c ON c.tire_set_id=t.id JOIN storage_details d ON d.claim_id=c.id
   WHERE t.vehicle_id=? AND c.place_id=? AND c.status='released' AND d.tire_type=? AND t.id!=?
   AND NOT EXISTS(SELECT 1 FROM storage_claims active WHERE active.tire_set_id=t.id AND active.status='active')
   ORDER BY c.id DESC LIMIT 1`, checked.vehicleID.Int64, input.Place(), input.TireType, checked.tireID.Int64).Scan(&tire)
		if errors.Is(err, sql.ErrNoRows) {
			tire, err = insertID("INSERT INTO tire_sets(vehicle_id) VALUES(?)", checked.vehicleID.Int64)
		}
		if err != nil {
			return 0, err
		}
	}
	if checked.Status == "reserved" {
		// The immediate transaction serializes the identity check, conversion and insert.
		if _, err = tx.ExecContext(ctx, "UPDATE storage_claims SET status='converted',updated_at=? WHERE id=? AND status='active'", stamp, checked.claimID); err != nil {
			return 0, err
		}
		tire = checked.tireID.Int64
	} else if !swap {
		customer, err := insertID("INSERT INTO customers(name) VALUES(?)", input.Name)
		if err != nil {
			return 0, err
		}
		vehicle, err := insertID("INSERT INTO vehicles(customer_id,plate,car,brand,model) VALUES(?,?,?,?,?)", customer, input.Plate, input.Brand+" "+input.Model, input.Brand, input.Model)
		if err != nil {
			return 0, err
		}
		tire, err = insertID("INSERT INTO tire_sets(vehicle_id) VALUES(?)", vehicle)
		if err != nil {
			return 0, err
		}
	}
	if swap || checked.Status == "reserved" {
		if _, err = tx.ExecContext(ctx, "UPDATE vehicles SET car=?,brand=?,model=? WHERE id=?", input.Brand+" "+input.Model, input.Brand, input.Model, checked.vehicleID.Int64); err != nil {
			return 0, err
		}
	}
	claim, err := insertID("INSERT INTO storage_claims(place_id,tire_set_id,kind,status,updated_at) VALUES(?,?,'occupancy','active',?)", input.Place(), tire, stamp)
	if err != nil {
		var se sqlite3.Error
		if errors.As(err, &se) && se.ExtendedCode == sqlite3.ErrConstraintUnique {
			return 0, ErrPlaceUnavailable
		}
		return 0, err
	}
	front, _ := json.Marshal(input.Front)
	rear, _ := json.Marshal(input.Rear)
	accessories, _ := json.Marshal(map[string]models.Accessory{"hubcaps": input.Hubcaps, "bolts": input.Bolts, "lock": input.Lock})
	_, err = tx.ExecContext(ctx, `INSERT INTO storage_details(claim_id,tire_type,rim_type,front_size,rear_size,front_depth,rear_depth,accessories,mechanic,notes,operation) VALUES(?,?,?,?,?,?,?,?,?,?,?)`, claim, input.TireType, input.RimType, string(front), string(rear), input.FrontDepth, input.RearDepth, string(accessories), input.Mechanic, input.Notes, operation)
	if err != nil {
		return 0, err
	}
	if err = recordHistory(ctx, tx, claim, operation, stamp); err != nil {
		return 0, err
	}
	if err = enqueueLabel(ctx, tx, claim, operation); err != nil {
		return 0, err
	}
	if err = tx.Commit(); err != nil {
		return 0, err
	}
	return claim, nil
}
