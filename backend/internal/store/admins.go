package store

import (
	"context"

	"github.com/jackc/pgx/v5"
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

// BootstrapAdmin ensures the admin row exists for email and, if its hash is
// empty and a hash is supplied, sets it once.
func (d *DB) BootstrapAdmin(ctx context.Context, email, name, hash string) error {
	_, err := d.Pool.Exec(ctx, `INSERT INTO admins (id, email, name) VALUES ('adm_01', $1, $2)
		ON CONFLICT (id) DO UPDATE SET email = EXCLUDED.email, name = EXCLUDED.name`, email, name)
	if err != nil {
		return err
	}
	if hash == "" {
		return nil
	}
	_, err = d.Pool.Exec(ctx, `UPDATE admins SET password_hash = $1 WHERE id = 'adm_01' AND password_hash = ''`, hash)
	return err
}
