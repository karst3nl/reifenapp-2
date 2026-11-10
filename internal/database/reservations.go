package database

import (
	"context"
	"database/sql"
	"time"

	"github.com/karst3nl/reifenapp-2/internal/models"
)

func CreateReservation(ctx context.Context, db *sql.DB, input models.ReservationInput) (int64, error) {
	if len(input.Validate()) > 0 {
		return 0, ErrInvalidStorage
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	checked, err := checkPlace(ctx, tx, input.Area, input.Number, input.Name, input.Plate)
	if err != nil {
		return 0, err
	}
	// Matching an existing reservation permits conversion, never a second reservation.
	if checked.Status != "free" {
		return 0, ErrPlaceUnavailable
	}
	insertID := func(query string, args ...any) (int64, error) {
		r, e := tx.ExecContext(ctx, query, args...)
		if e != nil {
			return 0, e
		}
		return r.LastInsertId()
	}
	customer, err := insertID("INSERT INTO customers(name) VALUES(?)", input.Name)
	if err != nil {
		return 0, err
	}
	vehicle, err := insertID("INSERT INTO vehicles(customer_id,plate,car,brand,model) VALUES(?,?,?,?,?)", customer, input.Plate, input.Brand+" "+input.Model, input.Brand, input.Model)
	if err != nil {
		return 0, err
	}
	tire, err := insertID("INSERT INTO tire_sets(vehicle_id) VALUES(?)", vehicle)
	if err != nil {
		return 0, err
	}
	claim, err := insertID("INSERT INTO storage_claims(place_id,tire_set_id,kind,status,updated_at) VALUES(?,?,'reservation','active',?)", input.Place(), tire, time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return 0, err
	}
	// Empty JSON objects mean not captured; no invented axle/accessory values.
	_, err = tx.ExecContext(ctx, `INSERT INTO storage_details(claim_id,tire_type,rim_type,front_size,rear_size,front_depth,rear_depth,accessories,mechanic,notes) VALUES(?,?,?,'{}','{}','','','{}','','')`, claim, input.TireType, input.RimType)
	if err != nil {
		return 0, err
	}
	if err = tx.Commit(); err != nil {
		return 0, err
	}
	return claim, nil
}
