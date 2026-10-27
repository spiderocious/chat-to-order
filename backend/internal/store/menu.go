package store

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const itemCols = `id, name, description, price_kobo, portions, available, "window", sort, photo_url, version, created_at, updated_at`

func (d *DB) Items(ctx context.Context) ([]Item, error) {
	rows, err := d.Pool.Query(ctx, `SELECT `+itemCols+` FROM menu_items ORDER BY sort, id`)
	if err != nil {
		return nil, err
	}
	items, err := pgx.CollectRows(rows, pgx.RowToStructByName[Item])
	if items == nil {
		items = []Item{}
	}
	return items, err
}

func (d *DB) Item(ctx context.Context, id string) (Item, error) {
	rows, err := d.Pool.Query(ctx, `SELECT `+itemCols+` FROM menu_items WHERE id = $1`, id)
	if err != nil {
		return Item{}, err
	}
	it, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[Item])
	if isNoRows(err) {
		return Item{}, ErrNotFound
	}
	return it, err
}

var ErrConflict = errors.New("store: conflict")

func (d *DB) CreateItem(ctx context.Context, it Item) (Item, error) {
	rows, err := d.Pool.Query(ctx, `
		INSERT INTO menu_items (id, name, description, price_kobo, portions, available, "window", sort, photo_url)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING `+itemCols,
		it.ID, it.Name, it.Description, it.PriceKobo, it.Portions, it.Available, it.Window, it.Sort, it.PhotoURL)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return Item{}, ErrConflict
		}
		return Item{}, err
	}
	return pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[Item])
}

// ItemPatch carries the subset of fields to change; nil means untouched.
type ItemPatch struct {
	Name        *string
	Description *string
	PriceKobo   *int64
	Portions    *int
	Available   *bool
	Window      *string
	Sort        *int
	PhotoURL    *string
}

// PatchItem updates under optimistic concurrency and a row lock, so it
// serialises with a confirm that is decrementing the same row.
func (d *DB) PatchItem(ctx context.Context, id string, p ItemPatch, expectedVersion int) (Item, error) {
	var set []string
	var args []any
	add := func(col string, v any) {
		args = append(args, v)
		set = append(set, fmt.Sprintf("%s = $%d", col, len(args)))
	}
	if p.Name != nil {
		add("name", *p.Name)
	}
	if p.Description != nil {
		add("description", *p.Description)
	}
	if p.PriceKobo != nil {
		add("price_kobo", *p.PriceKobo)
	}
	if p.Portions != nil {
		add("portions", *p.Portions)
	}
	if p.Available != nil {
		add("available", *p.Available)
	}
	if p.Window != nil {
		add(`"window"`, *p.Window)
	}
	if p.Sort != nil {
		add("sort", *p.Sort)
	}
	if p.PhotoURL != nil {
		add("photo_url", *p.PhotoURL)
	}
	if len(set) == 0 {
		return d.Item(ctx, id)
	}
	args = append(args, id, expectedVersion)
	q := fmt.Sprintf(`UPDATE menu_items SET %s, version = version + 1, updated_at = now()
		WHERE id = $%d AND version = $%d RETURNING `+itemCols, strings.Join(set, ", "), len(args)-1, len(args))

	var out Item
	err := d.Tx(ctx, func(tx pgx.Tx) error {
		// Lock first so the version check and the write see the same row.
		var cur int
		if err := tx.QueryRow(ctx, `SELECT version FROM menu_items WHERE id = $1 FOR UPDATE`, id).Scan(&cur); err != nil {
			if isNoRows(err) {
				return ErrNotFound
			}
			return err
		}
		if cur != expectedVersion {
			return ErrVersionConflict
		}
		rows, err := tx.Query(ctx, q, args...)
		if err != nil {
			return err
		}
		out, err = pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[Item])
		return err
	})
	return out, err
}

func (d *DB) Zones(ctx context.Context) ([]Zone, error) {
	rows, err := d.Pool.Query(ctx, `SELECT id, name, fee_kobo, active, sort FROM delivery_zones ORDER BY sort, id`)
	if err != nil {
		return nil, err
	}
	z, err := pgx.CollectRows(rows, pgx.RowToStructByName[Zone])
	if z == nil {
		z = []Zone{}
	}
	return z, err
}

func (d *DB) CreateZone(ctx context.Context, z Zone) (Zone, error) {
	rows, err := d.Pool.Query(ctx, `INSERT INTO delivery_zones (id, name, fee_kobo, active, sort) VALUES ($1,$2,$3,$4,$5)
		RETURNING id, name, fee_kobo, active, sort`, z.ID, z.Name, z.FeeKobo, z.Active, z.Sort)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return Zone{}, ErrConflict
		}
		return Zone{}, err
	}
	return pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[Zone])
}

type ZonePatch struct {
	Name    *string
	FeeKobo *int64
	Active  *bool
	Sort    *int
}

func (d *DB) PatchZone(ctx context.Context, id string, p ZonePatch) (Zone, error) {
	rows, err := d.Pool.Query(ctx, `UPDATE delivery_zones SET
		name = COALESCE($2, name), fee_kobo = COALESCE($3, fee_kobo), active = COALESCE($4, active), sort = COALESCE($5, sort)
		WHERE id = $1 RETURNING id, name, fee_kobo, active, sort`, id, p.Name, p.FeeKobo, p.Active, p.Sort)
	if err != nil {
		return Zone{}, err
	}
	z, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[Zone])
	if isNoRows(err) {
		return Zone{}, ErrNotFound
	}
	return z, err
}
