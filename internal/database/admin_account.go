package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
)

const DefaultAdminPassword = "Reifenapp-Start2026!"

var ErrAdminPolicy = errors.New("Ungültiges neues Passwort")
var ErrAdminCredentials = errors.New("Anmeldedaten sind ungültig.")
var ErrAdminRevision = errors.New("Die Anmeldung wurde inzwischen geändert. Bitte neu anmelden.")

type AdminAccount struct {
	Hash       string `json:"-"`
	MustChange bool
	Revision   int
}

func LoadAdminAccount(ctx context.Context, db *sql.DB) (AdminAccount, error) {
	var account AdminAccount
	err := db.QueryRowContext(ctx, "SELECT password_hash,must_change,revision FROM admin_account WHERE id=1").Scan(&account.Hash, &account.MustChange, &account.Revision)
	return account, err
}
func EnsureAdminAccount(ctx context.Context, db *sql.DB) error {
	_, err := LoadAdminAccount(ctx, db)
	if err == nil {
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(DefaultAdminPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, "INSERT INTO admin_account(id,username,password_hash,must_change,revision,updated_at) VALUES(1,'admin',?,1,1,?) ON CONFLICT(id) DO NOTHING", string(hash), time.Now().UTC().Format(time.RFC3339Nano))
	return err
}
func ChangeAdminPassword(ctx context.Context, db *sql.DB, revision int, current, password string) (AdminAccount, error) {
	var empty AdminAccount
	if utf8.RuneCountInString(password) < 12 || len(password) > 72 || strings.TrimSpace(password) == "" {
		return empty, fmt.Errorf("%w: mindestens 12 Zeichen, höchstens 72 Bytes.", ErrAdminPolicy)
	}
	if password == DefaultAdminPassword {
		return empty, fmt.Errorf("%w: Das Startpasswort darf nicht erneut verwendet werden.", ErrAdminPolicy)
	}
	account, err := LoadAdminAccount(ctx, db)
	if err != nil {
		return empty, err
	}
	if account.Revision != revision {
		return empty, ErrAdminRevision
	}
	if !account.MustChange && bcrypt.CompareHashAndPassword([]byte(account.Hash), []byte(current)) != nil {
		return empty, ErrAdminCredentials
	}
	if bcrypt.CompareHashAndPassword([]byte(account.Hash), []byte(password)) == nil {
		return empty, fmt.Errorf("%w: Bitte ein anderes Passwort als bisher wählen.", ErrAdminPolicy)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return empty, err
	}
	result, err := db.ExecContext(ctx, "UPDATE admin_account SET password_hash=?,must_change=0,revision=revision+1,updated_at=? WHERE id=1 AND revision=?", string(hash), time.Now().UTC().Format(time.RFC3339Nano), revision)
	if err != nil {
		return empty, err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return empty, err
	}
	if n != 1 {
		return empty, ErrAdminRevision
	}
	return AdminAccount{Hash: string(hash), MustChange: false, Revision: revision + 1}, nil
}
