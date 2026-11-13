package database

import (
	"context"
	"database/sql"
	"strings"
)

// CustomerNames returns up to 20 distinct names matching a Unicode-aware prefix.
func CustomerNames(ctx context.Context, db *sql.DB, query string) ([]string, error) {
	result := []string{}
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return result, nil
	}
	rows, err := db.QueryContext(ctx, "SELECT name FROM customers UNION SELECT name FROM history_events WHERE source='csv' AND name!='' ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err = rows.Scan(&name); err != nil {
			return nil, err
		}
		if strings.HasPrefix(strings.ToLower(name), query) {
			result = append(result, name)
			if len(result) == 20 {
				break
			}
		}
	}
	return result, rows.Err()
}
