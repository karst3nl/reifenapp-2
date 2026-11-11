package database

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/karst3nl/reifenapp-2/internal/models"
	"github.com/karst3nl/reifenapp-2/migrations"
	_ "github.com/mattn/go-sqlite3"
)

const currentSchema = 4

// Open opens the current schema without upgrading an existing database.
func Open(path string) (*sql.DB, error) { return OpenWithMigration(path, false) }

// OpenWithMigration requires explicit authorization for upgrades and creates
// a consistent SQLite backup before changing an existing schema.
func OpenWithMigration(path string, upgrade bool) (*sql.DB, error) {
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
	fail := func(err error) (*sql.DB, error) { db.Close(); return nil, err }
	if err = db.Ping(); err != nil {
		return fail(err)
	}
	version := 0
	if !fresh {
		if err = db.QueryRow("SELECT COALESCE(MAX(version),0) FROM schema_migrations").Scan(&version); err != nil {
			return fail(err)
		}
		if version < 1 || version > currentSchema {
			return fail(fmt.Errorf("unbekannte Schemaversion %d", version))
		}
		if version < currentSchema {
			if !upgrade {
				return fail(fmt.Errorf("Schemaversion %d: Server stoppen, dann mit -migrate starten; vorher wird automatisch ein Backup erzeugt", version))
			}
			var integrity string
			if err = db.QueryRow("PRAGMA integrity_check").Scan(&integrity); err != nil {
				return fail(err)
			}
			if integrity != "ok" {
				return fail(fmt.Errorf("Integritätsprüfung fehlgeschlagen; kein Upgrade"))
			}
			backup := absolute + ".backup-" + time.Now().UTC().Format("20060102T150405.000000000Z")
			if _, err = db.Exec("VACUUM INTO ?", backup); err != nil {
				return fail(fmt.Errorf("Backup fehlgeschlagen: %w", err))
			}
			if err = os.Chmod(backup, 0600); err != nil {
				return fail(err)
			}
		}
	}
	if err = migrate(db, version); err != nil {
		return fail(err)
	}
	return db, nil
}

func migrate(db *sql.DB, version int) error {
	if version == currentSchema {
		return nil
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if version == 0 {
		if _, err = tx.Exec("CREATE TABLE schema_migrations(version INTEGER PRIMARY KEY)"); err != nil {
			return err
		}
	}
	files := []string{"001_initial.sql", "002_storage_details.sql", "003_storage_operation.sql", "004_storage_releases.sql"}
	for i := version; i < len(files); i++ {
		script, err := migrations.Files.ReadFile(files[i])
		if err != nil {
			return err
		}
		if _, err = tx.Exec(string(script)); err != nil {
			return err
		}
		if _, err = tx.Exec("INSERT INTO schema_migrations VALUES(?)", i+1); err != nil {
			return err
		}
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
	rows, err := db.QueryContext(ctx, `SELECT p.id,c.name,v.plate,v.car,s.updated_at,s.kind,CASE WHEN s.kind='reservation' THEN 'reservation' ELSE COALESCE(d.operation,'storage') END
 FROM storage_claims s JOIN storage_places p ON p.id=s.place_id
 JOIN tire_sets t ON t.id=s.tire_set_id JOIN vehicles v ON v.id=t.vehicle_id
 JOIN customers c ON c.id=v.customer_id LEFT JOIN storage_details d ON d.claim_id=s.id WHERE s.status='active'
 ORDER BY s.updated_at DESC,s.id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var r models.SearchResult
		if err = rows.Scan(&r.Place, &r.Name, &r.Plate, &r.Car, &r.Date, &r.Kind, &r.Operation); err != nil {
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
		`INSERT INTO vehicles(id,customer_id,plate,car) VALUES(1,1,'TEST-MM 123','VW Golf'),(2,2,'TEST-EB 456','Audi A3')`,
		`INSERT INTO tire_sets VALUES(1,1),(2,2)`,
		`INSERT INTO storage_claims VALUES(1,'C1.25',1,'occupancy','active','2026-11-09T14:40:00Z'),(2,'C2.215',2,'occupancy','active','2026-11-08T10:30:00Z')`,
	} {
		if _, err = tx.Exec(s); err != nil {
			return err
		}
	}
	return tx.Commit()
}
