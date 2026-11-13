package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/karst3nl/reifenapp-2/internal/models"
)

var ErrInvalidHistoryFilter = errors.New("Ungültige Historienfilter.")

type HistoryFilter struct {
	Query, Operation, Status, Source string
	From, Until                      time.Time
	WarningsOnly                     bool
	Page, PageSize                   int
	Snapshot                         int64
}
type HistoryPage struct {
	Items      []models.SearchResult `json:"items"`
	Total      int                   `json:"total"`
	Page       int                   `json:"page"`
	PageSize   int                   `json:"page_size"`
	Pages      int                   `json:"pages"`
	Snapshot   int64                 `json:"snapshot"`
	Operations map[string]int        `json:"operations"`
}

// The event-id boundary keeps pagination stable while new events are recorded.
func BrowseHistory(ctx context.Context, db *sql.DB, f HistoryFilter) (HistoryPage, error) {
	result := HistoryPage{Items: []models.SearchResult{}, Page: f.Page, PageSize: f.PageSize, Snapshot: f.Snapshot, Operations: map[string]int{"storage": 0, "wheel_change": 0, "reservation": 0, "release": 0, "imported": 0}}
	if f.Page < 1 || f.Page > 1000000 || f.PageSize < 1 || f.PageSize > 100 {
		return result, ErrInvalidHistoryFilter
	}
	rows, err := db.QueryContext(ctx, `WITH boundary AS (SELECT CASE WHEN ?=0 THEN COALESCE(MAX(id),0) ELSE ? END AS id FROM history_events)
 SELECT h.id,COALESCE(h.claim_id,0),h.operation,h.occurred_at,h.place,h.name,h.plate,h.car,h.source,h.payload,COALESCE(s.status,'archived'),
 COALESCE((s.status='active' AND h.operation!='release'),0),b.id
 FROM history_events h LEFT JOIN storage_claims s ON s.id=h.claim_id CROSS JOIN boundary b
 WHERE h.id<=b.id AND (?='' OR h.operation=?) AND (?='' OR h.source=?)
 ORDER BY julianday(h.occurred_at) DESC,h.id DESC`, f.Snapshot, f.Snapshot, f.Operation, f.Operation, f.Source, f.Source)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	q := normalize(strings.TrimSpace(f.Query))
	offset := (f.Page - 1) * f.PageSize
	for rows.Next() {
		var r models.SearchResult
		var claimID int64
		var payload string
		if err = rows.Scan(&r.EventID, &claimID, &r.Operation, &r.Date, &r.Place, &r.Name, &r.Plate, &r.Car, &r.Source, &payload, &r.Status, &r.Active, &result.Snapshot); err != nil {
			return result, err
		}
		// Compare instants in Go: SQLite julianday rounds sub-millisecond boundaries.
		if !f.From.IsZero() || !f.Until.IsZero() {
			stamp, parseErr := time.Parse(time.RFC3339Nano, r.Date)
			if parseErr != nil || (!f.From.IsZero() && stamp.Before(f.From)) || (!f.Until.IsZero() && !stamp.Before(f.Until)) {
				continue
			}
		}
		if f.Status == "active" && !r.Active || f.Status == "closed" && (r.Active || r.Source == "csv") || f.Status == "archived" && r.Source != "csv" {
			continue
		}
		if err = json.Unmarshal([]byte(payload), &r.Details); err != nil {
			return result, err
		}
		r.TreadWarnings = r.Details.TreadWarnings()
		if f.WarningsOnly && len(r.TreadWarnings) == 0 {
			continue
		}
		label := map[string]string{"storage": "Neueinlagerung", "wheel_change": "Radwechsel", "reservation": "Reservierung", "release": "Auslagerung", "imported": "Altvorgang"}[r.Operation]
		if q != "" {
			match := false
			values := []string{r.Name, r.Plate, r.Place, r.Car, label, r.Details.Notes, r.Details.Mechanic}
			if r.Details.Import != nil {
				for _, field := range r.Details.Import.Fields {
					values = append(values, field.Value)
				}
			}
			for _, value := range values {
				if strings.Contains(normalize(value), q) {
					match = true
					break
				}
			}
			if !match {
				continue
			}
		}
		result.Total++
		result.Operations[r.Operation]++
		if result.Total > offset && len(result.Items) < f.PageSize {
			r.Kind = "occupancy"
			if r.Operation == "reservation" {
				r.Kind = "reservation"
			}
			if r.Operation == "release" {
				r.Kind = "release"
			}
			if r.Source == "csv" {
				r.Kind = "archive"
			}
			if claimID > 0 {
				r.ReceiptURL = models.ReceiptPath(claimID, r.Operation)
			}
			result.Items = append(result.Items, r)
		}
	}
	result.Pages = (result.Total + f.PageSize - 1) / f.PageSize
	return result, rows.Err()
}
