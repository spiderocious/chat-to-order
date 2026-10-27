package store

import (
	"context"

	"github.com/jackc/pgx/v5"
)

const settingsCols = `vendor_name, pickup_address, rider_phone, order_prefix, prep_minutes,
  lunch_open, lunch_close, lunch_orders_close, evening_open, evening_close, evening_orders_close,
  pending_ttl_minutes, low_stock_threshold, version, updated_at`

func (d *DB) Settings(ctx context.Context) (Settings, error) {
	rows, err := d.Pool.Query(ctx, `SELECT `+settingsCols+` FROM vendor_settings WHERE id = 1`)
	if err != nil {
		return Settings{}, err
	}
	s, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[Settings])
	if isNoRows(err) {
		return Settings{}, ErrNotFound
	}
	return s, err
}

// UpdateSettings writes the whole row under optimistic concurrency.
func (d *DB) UpdateSettings(ctx context.Context, s Settings, expectedVersion int) (Settings, error) {
	rows, err := d.Pool.Query(ctx, `
		UPDATE vendor_settings SET vendor_name=$1, pickup_address=$2, rider_phone=$3, prep_minutes=$4,
		  lunch_open=$5, lunch_close=$6, lunch_orders_close=$7, evening_open=$8, evening_close=$9, evening_orders_close=$10,
		  pending_ttl_minutes=$11, low_stock_threshold=$12, version = version + 1, updated_at = now()
		WHERE id = 1 AND version = $13
		RETURNING `+settingsCols,
		s.VendorName, s.PickupAddress, s.RiderPhone, s.PrepMinutes,
		s.LunchOpen, s.LunchClose, s.LunchOrdersClose, s.EveningOpen, s.EveningClose, s.EveningOrdersClose,
		s.PendingTTLMinutes, s.LowStockThreshold, expectedVersion)
	if err != nil {
		return Settings{}, err
	}
	out, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[Settings])
	if isNoRows(err) {
		return Settings{}, ErrVersionConflict
	}
	return out, err
}
