package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

var ErrInvalidPlace = errors.New("Ungültiger Lagerplatz.")

type PlaceCheck struct {
	Place     string `json:"place"`
	Status    string `json:"status"`
	Allowed   bool   `json:"allowed"`
	Matches   bool   `json:"matches"`
	Message   string `json:"message"`
	claimID   int64
	tireID    sql.NullInt64
	vehicleID sql.NullInt64
}
type rowQuerier interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func CheckPlace(ctx context.Context, db *sql.DB, area string, number int, name, plate string) (PlaceCheck, error) {
	return checkPlace(ctx, db, area, number, name, plate)
}
func checkPlace(ctx context.Context, q rowQuerier, area string, number int, name, plate string) (PlaceCheck, error) {
	result := PlaceCheck{Place: fmt.Sprintf("%s.%d", area, number)}
	var kind, customer, registration sql.NullString
	var claim sql.NullInt64
	err := q.QueryRowContext(ctx, `SELECT s.id,s.kind,s.tire_set_id,v.id,c.name,v.plate
 FROM storage_places p LEFT JOIN storage_claims s ON s.place_id=p.id AND s.status='active'
 LEFT JOIN tire_sets t ON t.id=s.tire_set_id LEFT JOIN vehicles v ON v.id=t.vehicle_id
 LEFT JOIN customers c ON c.id=v.customer_id WHERE p.id=? AND p.enabled=1`, result.Place).Scan(&claim, &kind, &result.tireID, &result.vehicleID, &customer, &registration)
	if errors.Is(err, sql.ErrNoRows) {
		return result, ErrInvalidPlace
	}
	if err != nil {
		return result, err
	}
	if !claim.Valid {
		result.Status = "free"
		result.Allowed = true
		result.Message = "Lagerplatz " + result.Place + " ist frei."
		return result, nil
	}
	result.claimID = claim.Int64
	if kind.String == "occupancy" {
		result.Status = "occupied"
		result.Message = "Lagerplatz " + result.Place + " ist bereits belegt."
		return result, nil
	}
	result.Status = "reserved"
	cleanName := func(s string) string { return strings.ToLower(strings.Join(strings.Fields(s), " ")) }
	result.Matches = customer.Valid && registration.Valid && cleanName(name) != "" && normalize(plate) != "" && cleanName(name) == cleanName(customer.String) && normalize(plate) == normalize(registration.String)
	result.Allowed = result.Matches
	if result.Matches {
		result.Message = "Reservierung für " + result.Place + " passt zu Kundenname und Kennzeichen. Einlagerung möglich."
	} else if !customer.Valid || !registration.Valid {
		result.Message = "Lagerplatz " + result.Place + " ist reserviert. Ohne eindeutige Kunden- und Fahrzeugzuordnung ist keine Einlagerung möglich."
	} else {
		result.Message = "Lagerplatz " + result.Place + " ist reserviert. Kundenname und Kennzeichen müssen beide mit der Reservierung übereinstimmen."
	}
	return result, nil
}
