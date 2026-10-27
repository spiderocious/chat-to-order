// Package store is the Postgres port: pool, migrations, and one function per
// query. It knows SQL and nothing about windows, carts or Paystack.
package store

import (
	"context"
	"database/sql"
	"embed"
	_ "embed"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

//go:embed demo_seed.sql
var demoSeedSQL string

var ErrNotFound = errors.New("store: not found")
var ErrVersionConflict = errors.New("store: version conflict")

type DB struct {
	Pool *pgxpool.Pool
}

func Open(ctx context.Context, url string) (*DB, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("store: parse url: %w", err)
	}
	cfg.MaxConns = 8
	cfg.MinConns = 1
	cfg.MaxConnLifetime = 30 * time.Minute
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("store: connect: %w", err)
	}
	pctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("store: ping: %w", err)
	}
	return &DB{Pool: pool}, nil
}

func (d *DB) Close() { d.Pool.Close() }

// Migrate runs goose over the embedded migrations.
func (d *DB) Migrate(ctx context.Context) error {
	goose.SetBaseFS(migrationsFS)
	goose.SetLogger(goose.NopLogger())
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	sqlDB := stdlib.OpenDBFromPool(d.Pool)
	defer sqlDB.Close()
	return goose.UpContext(ctx, sqlDB, "migrations")
}

// Ping is the health check.
func (d *DB) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return d.Pool.Ping(ctx)
}

// Tx runs fn inside a transaction, committing on nil and rolling back on error or panic.
func (d *DB) Tx(ctx context.Context, fn func(tx pgx.Tx) error) (err error) {
	tx, err := d.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()
	if err = fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Querier is satisfied by both the pool and a transaction.
type Querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// OrdersEmpty reports whether the orders table has no rows (used by demo seeding).
func (d *DB) OrdersEmpty(ctx context.Context) (bool, error) {
	var n int
	err := d.Pool.QueryRow(ctx, `SELECT count(*) FROM orders`).Scan(&n)
	return n == 0, err
}

// SeedDemo applies the demo orders. Safe to call on an empty orders table only.
func (d *DB) SeedDemo(ctx context.Context) error {
	_, err := d.Pool.Exec(ctx, demoSeedSQL)
	return err
}

// Reset truncates transactional data, restores seed stock and re-applies the demo.
func (d *DB) Reset(ctx context.Context, withDemo bool) error {
	return d.Tx(ctx, func(tx pgx.Tx) error {
		stmts := []string{
			`TRUNCATE orders, order_lines, order_events, payment_checks, tool_calls RESTART IDENTITY CASCADE`,
			`ALTER SEQUENCE order_no_seq RESTART WITH 231`,
			`UPDATE menu_items SET portions = s.p, available = true, version = version + 1, updated_at = now()
			   FROM (VALUES ('jollof',20),('fried',15),('ofada',2),('beans',12),('moimoi',30),('egusi',10),('chapman',25),('water',40)) AS s(id,p)
			  WHERE menu_items.id = s.id`,
		}
		for _, s := range stmts {
			if _, err := tx.Exec(ctx, s); err != nil {
				return err
			}
		}
		if withDemo {
			if _, err := tx.Exec(ctx, demoSeedSQL); err != nil {
				return err
			}
		}
		return nil
	})
}

func isNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows)
}
