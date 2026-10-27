package store

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

const adminCols = `id, email, name, password_hash, token_version, created_at`

func (d *DB) AdminByEmail(ctx context.Context, email string) (Admin, error) {
	rows, err := d.Pool.Query(ctx, `SELECT `+adminCols+` FROM admins WHERE lower(email) = lower($1)`, email)
	if err != nil {
		return Admin{}, err
	}
	a, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[Admin])
	if isNoRows(err) {
		return Admin{}, ErrNotFound
	}
	return a, err
}

func (d *DB) AdminByID(ctx context.Context, id string) (Admin, error) {
	rows, err := d.Pool.Query(ctx, `SELECT `+adminCols+` FROM admins WHERE id = $1`, id)
	if err != nil {
		return Admin{}, err
	}
	a, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[Admin])
	if isNoRows(err) {
		return Admin{}, ErrNotFound
	}
	return a, err
}

// BootstrapAdmin upserts the owner login keyed on email (unique). It runs at
// boot from ADMIN_EMAIL / ADMIN_NAME / ADMIN_PASSWORD, so there is no endpoint
// that can mint or reset an admin and nothing secret lives in the repo. The
// env password is the source of truth: when it no longer matches the stored
// hash the hash is rotated and token_version is bumped, which revokes older
// sessions. When it still matches nothing changes (bcrypt salts differ per
// hash, so comparing hashes would rotate on every restart). An empty password
// keeps whatever is stored. The id is derived from the email so re-runs are
// idempotent.
func (d *DB) BootstrapAdmin(ctx context.Context, email, name, password string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	existing, err := d.AdminByEmail(ctx, email)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return err
	}
	hashFor := func() (string, error) {
		if password == "" {
			return "", nil
		}
		h, err := bcrypt.GenerateFromPassword([]byte(password), 12)
		return string(h), err
	}
	if errors.Is(err, ErrNotFound) {
		hash, err := hashFor()
		if err != nil {
			return err
		}
		_, err = d.Pool.Exec(ctx, `INSERT INTO admins (id, email, name, password_hash)
			VALUES ('adm_' || left(md5($1), 12), $1, $2, $3)`, email, name, hash)
		return err
	}
	if password == "" || bcrypt.CompareHashAndPassword([]byte(existing.PasswordHash), []byte(password)) == nil {
		_, err = d.Pool.Exec(ctx, `UPDATE admins SET name = $2 WHERE id = $1`, existing.ID, name)
		return err
	}
	hash, err := hashFor()
	if err != nil {
		return err
	}
	_, err = d.Pool.Exec(ctx, `UPDATE admins SET name = $2, password_hash = $3, token_version = token_version + 1 WHERE id = $1`,
		existing.ID, name, hash)
	return err
}
