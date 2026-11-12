package database

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
)

var ErrAreaInput = errors.New("Lagerort: 1–12 Großbuchstaben, Ziffern, _ oder -. Nummern: 1–100000, höchstens 10000 Plätze je Bereich.")
var ErrAreaConflict = errors.New("Bereich wurde inzwischen geändert. Bitte neu laden.")
var ErrAreaOccupied = errors.New("Der neue Zahlenbereich würde belegte oder reservierte Plätze ausschließen. Zuerst diese Vorgänge abschließen.")

type StorageArea struct {
	Code     string `json:"code"`
	First    int    `json:"first_number"`
	Last     int    `json:"last_number"`
	Revision int    `json:"revision"`
	Occupied int    `json:"occupied"`
	Reserved int    `json:"reserved"`
}

var areaCode = regexp.MustCompile(`^[A-Z0-9_-]{1,12}$`)

func StorageAreas(ctx context.Context, db *sql.DB) ([]StorageArea, error) {
	rows, err := db.QueryContext(ctx, `SELECT a.code,a.first_number,a.last_number,a.revision,
 (SELECT count(*) FROM storage_claims c JOIN storage_places p ON p.id=c.place_id WHERE p.area=a.code AND c.status='active' AND c.kind='occupancy'),
 (SELECT count(*) FROM storage_claims c JOIN storage_places p ON p.id=c.place_id WHERE p.area=a.code AND c.status='active' AND c.kind='reservation')
 FROM storage_areas a ORDER BY a.code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []StorageArea{}
	for rows.Next() {
		var a StorageArea
		if err = rows.Scan(&a.Code, &a.First, &a.Last, &a.Revision, &a.Occupied, &a.Reserved); err != nil {
			return nil, err
		}
		result = append(result, a)
	}
	return result, rows.Err()
}

// SaveStorageArea changes only the selectable catalogue; historical identifiers never change.
func SaveStorageArea(ctx context.Context, db *sql.DB, a StorageArea) error {
	if !areaCode.MatchString(a.Code) || a.First < 1 || a.Last < a.First || a.Last > 100000 || a.Last-a.First >= 10000 || a.Revision < 0 {
		return ErrAreaInput
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var revision int
	err = tx.QueryRowContext(ctx, "SELECT revision FROM storage_areas WHERE code=?", a.Code).Scan(&revision)
	if errors.Is(err, sql.ErrNoRows) {
		if a.Revision != 0 {
			return ErrAreaConflict
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO storage_areas(code,first_number,last_number) VALUES(?,?,?)", a.Code, a.First, a.Last); err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else {
		if revision != a.Revision {
			return ErrAreaConflict
		}
		var blocked int
		if err = tx.QueryRowContext(ctx, `SELECT count(*) FROM storage_claims c JOIN storage_places p ON p.id=c.place_id WHERE p.area=? AND c.status='active' AND (p.number<? OR p.number>?)`, a.Code, a.First, a.Last).Scan(&blocked); err != nil {
			return err
		}
		if blocked > 0 {
			return ErrAreaOccupied
		}
		if _, err = tx.ExecContext(ctx, "UPDATE storage_areas SET first_number=?,last_number=?,revision=revision+1 WHERE code=?", a.First, a.Last, a.Code); err != nil {
			return err
		}
	}
	if _, err = tx.ExecContext(ctx, "UPDATE storage_places SET enabled=0 WHERE area=? AND (number<? OR number>?)", a.Code, a.First, a.Last); err != nil {
		return err
	}
	stmt, err := tx.PrepareContext(ctx, `INSERT INTO storage_places(id,area,number,enabled) VALUES(? || '.' || ?,?,?,1) ON CONFLICT(area,number) DO UPDATE SET enabled=1`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for n := a.First; n <= a.Last; n++ {
		if _, err = stmt.ExecContext(ctx, a.Code, n, a.Code, n); err != nil {
			return err
		}
	}
	return tx.Commit()
}
