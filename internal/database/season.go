package database

import (
	"context"
	"database/sql"
	"time"

	"github.com/karst3nl/reifenapp-2/internal/models"
)

type SeasonProgress struct {
	Season    models.Season `json:"season"`
	Completed int           `json:"completed"`
	Open      int           `json:"open"`
	Total     int           `json:"total"`
	Unknown   int           `json:"unknown"`
	AllSeason int           `json:"all_season"`
	Active    int           `json:"active"`
}

// Seasonal progress describes current occupied sets, not reservations or an
// inferred physical swap. A wheel-change booking is the evidence for completion.
func CurrentSeasonProgress(ctx context.Context, db *sql.DB, now time.Time) (SeasonProgress, error) {
	result := SeasonProgress{Season: models.CurrentSeason(now)}
	rows, err := db.QueryContext(ctx, `SELECT COALESCE(h.operation,''),COALESCE(h.occurred_at,''),COALESCE(h.source,''),
 CASE WHEN h.id IS NOT NULL THEN COALESCE(json_extract(h.payload,'$.tire_type'),'') ELSE COALESCE(d.tire_type,'') END
 FROM storage_claims s LEFT JOIN storage_details d ON d.claim_id=s.id
 LEFT JOIN history_events h ON h.id=(SELECT e.id FROM history_events e WHERE e.claim_id=s.id AND e.operation IN ('storage','wheel_change') ORDER BY e.id DESC LIMIT 1)
 WHERE s.status='active' AND s.kind='occupancy'`)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		var operation, stamp, source, kind string
		if err = rows.Scan(&operation, &stamp, &source, &kind); err != nil {
			return result, err
		}
		result.Active++
		switch kind {
		case "Ganzjahr":
			result.AllSeason++
		case result.Season.MountedType:
			result.Open++
		case result.Season.StoredType:
			date, parseErr := time.Parse(time.RFC3339Nano, stamp)
			if operation == "wheel_change" && source == "recorded" && parseErr == nil && !date.Before(result.Season.Start) && date.Before(result.Season.End) && !date.After(now) {
				result.Completed++
			} else {
				result.Unknown++
			}
		default:
			result.Unknown++
		}
	}
	result.Total = result.Completed + result.Open
	return result, rows.Err()
}
