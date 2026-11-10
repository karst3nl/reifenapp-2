package database

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/karst3nl/reifenapp-2/internal/models"
	"github.com/karst3nl/reifenapp-2/migrations"
	_ "github.com/mattn/go-sqlite3"
)

// Open initializes only new databases. Existing databases must already have
// the current schema; future upgrades require an explicit backup workflow.
func Open(path string) (*sql.DB, error) {
	_, err := os.Stat(path)
	fresh := os.IsNotExist(err)
	if err != nil && !fresh {
		return nil, err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	u := url.URL{Scheme: "file", Path: absolute}
	db, err := sql.Open("sqlite3", u.String()+"?_foreign_keys=on&_journal_mode=WAL&_busy_timeout=5000&_txlock=immediate")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if err = db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	if fresh {
		err = migrate(db)
	} else {
		var version int
		err = db.QueryRow("SELECT COALESCE(MAX(version),0) FROM schema_migrations").Scan(&version)
		if err == nil && version != 1 {
			err = fmt.Errorf("unsupported schema version %d; backup and explicit migration required", version)
		}
	}
	if err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func migrate(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec("CREATE TABLE schema_migrations (version INTEGER PRIMARY KEY)"); err != nil {
		return err
	}
	script, err := migrations.Files.ReadFile("001_initial.sql")
	if err != nil {
		return err
	}
	if _, err = tx.Exec(string(script)); err != nil {
		return err
	}
	if _, err = tx.Exec("INSERT INTO schema_migrations VALUES(1)"); err != nil {
		return err
	}
	return tx.Commit()
}

func normalize(s string) string {
	return strings.NewReplacer(" ", "", "-", "", "\t", "", "\n", "").Replace(strings.ToLower(s))
}

func Search(ctx context.Context, db *sql.DB, q string) ([]models.SearchResult, error) {
	result := []models.SearchResult{}
	q = normalize(strings.TrimSpace(q))
	if q == "" {
		return result, nil
	}
	rows, err := db.QueryContext(ctx, `SELECT p.id,c.name,v.plate,v.car,s.updated_at
 FROM storage_claims s JOIN storage_places p ON p.id=s.place_id
 JOIN tire_sets t ON t.id=s.tire_set_id JOIN vehicles v ON v.id=t.vehicle_id
 JOIN customers c ON c.id=v.customer_id WHERE s.status='active'
 ORDER BY s.updated_at DESC,s.id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var r models.SearchResult
		if err = rows.Scan(&r.Place, &r.Name, &r.Plate, &r.Car, &r.Date); err != nil {
			return nil, err
		}
		if strings.Contains(normalize(r.Name), q) || strings.Contains(normalize(r.Plate), q) || strings.Contains(normalize(r.Place), q) {
			result = append(result, r)
		}
	}
	return result, rows.Err()
}

// SeedDevelopment is allowed only on a newly created, separate development DB.
func SeedDevelopment(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, s := range []string{
		`INSERT INTO customers VALUES(1,'Max Mustermann (fiktiv)'),(2,'Erika Beispiel (fiktiv)')`,
		`INSERT INTO vehicles VALUES(1,1,'TEST-MM 123','VW Golf'),(2,2,'TEST-EB 456','Audi A3')`,
		`INSERT INTO tire_sets VALUES(1,1),(2,2)`,
		`INSERT INTO storage_claims VALUES(1,'C1.25',1,'occupancy','active','2026-11-09T14:40:00Z'),(2,'C2.215',2,'occupancy','active','2026-11-08T10:30:00Z')`,
	} {
		if _, err = tx.Exec(s); err != nil {
			return err
		}
	}
	return tx.Commit()
}
