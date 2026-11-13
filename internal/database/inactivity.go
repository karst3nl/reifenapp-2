package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/karst3nl/reifenapp-2/internal/models"
)

var ErrInactivityMonths = errors.New("Die Frist muss zwischen 1 und 120 Monaten liegen.")

type InactivitySettings struct {
	Months int `json:"months"`
}
type InactivePlace struct {
	models.SearchResult
	LastMovement string `json:"last_movement"`
	Days         int    `json:"days"`
}
type InactivityReport struct {
	Months  int             `json:"months"`
	Count   int             `json:"count"`
	Unknown int             `json:"unknown"`
	Items   []InactivePlace `json:"items"`
}

func ReadInactivitySettings(ctx context.Context, db *sql.DB) (InactivitySettings, error) {
	var settings InactivitySettings
	err := db.QueryRowContext(ctx, "SELECT months FROM inactivity_settings WHERE id=1").Scan(&settings.Months)
	return settings, err
}
func SaveInactivitySettings(ctx context.Context, db *sql.DB, settings InactivitySettings) error {
	if settings.Months < 1 || settings.Months > 120 {
		return ErrInactivityMonths
	}
	_, err := db.ExecContext(ctx, "UPDATE inactivity_settings SET months=? WHERE id=1", settings.Months)
	return err
}

// Only recorded events of the current occupancy prove physical movement.
// CSV archive records, reservations and reconstructed legacy timestamps do not.
func InactivePlaces(ctx context.Context, db *sql.DB, now time.Time) (InactivityReport, error) {
	result := InactivityReport{Items: []InactivePlace{}}
	settings, err := ReadInactivitySettings(ctx, db)
	if err != nil {
		return result, err
	}
	result.Months = settings.Months
	rows, err := db.QueryContext(ctx, `SELECT COALESCE(h.id,0),s.id,s.place_id,COALESCE(h.name,c.name,''),COALESCE(h.plate,v.plate,''),COALESCE(h.car,v.car,''),COALESCE(h.operation,''),COALESCE(h.occurred_at,''),COALESCE(h.payload,'{}')
 FROM storage_claims s LEFT JOIN tire_sets t ON t.id=s.tire_set_id LEFT JOIN vehicles v ON v.id=t.vehicle_id LEFT JOIN customers c ON c.id=v.customer_id
 LEFT JOIN history_events h ON h.id=(SELECT e.id FROM history_events e WHERE e.claim_id=s.id AND e.source='recorded' AND e.operation IN ('storage','wheel_change') ORDER BY julianday(e.occurred_at) DESC,e.id DESC LIMIT 1)
 WHERE s.status='active' AND s.kind='occupancy'
 ORDER BY julianday(h.occurred_at),s.place_id`)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	location, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		return result, err
	}
	for rows.Next() {
		var item InactivePlace
		var claim int64
		var payload string
		if err = rows.Scan(&item.EventID, &claim, &item.Place, &item.Name, &item.Plate, &item.Car, &item.Operation, &item.Date, &payload); err != nil {
			return result, err
		}
		stamp, e := time.Parse(time.RFC3339Nano, item.Date)
		if e != nil || stamp.After(now) {
			result.Unknown++
			continue
		}
		if stamp.In(location).AddDate(0, settings.Months, 0).After(now) {
			continue
		}
		if err = json.Unmarshal([]byte(payload), &item.Details); err != nil {
			return result, err
		}
		item.LastMovement = item.Date
		item.Days = int(now.Sub(stamp).Hours() / 24)
		item.Active = true
		item.Status = "active"
		item.Kind = "occupancy"
		item.Source = "recorded"
		item.ReceiptURL = models.ReceiptPath(claim, item.Operation)
		item.TreadWarnings = item.Details.TreadWarnings()
		result.Items = append(result.Items, item)
	}
	result.Count = len(result.Items)
	return result, rows.Err()
}
