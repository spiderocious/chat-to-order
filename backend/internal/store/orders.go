package store

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const orderCols = `id, order_no, status, customer_name, phone, note, fulfilment, zone_id, address, "window",
  subtotal_kobo, delivery_fee_kobo, total_kobo, reference, authorization_url, cart_hash, source,
  paid_at, ready_by, confirmed_by, attention_reason, resolved_at, resolution_note, cancel_reason,
  expires_at, reconciled_at, created_at, updated_at`

const lineCols = `id, order_id, item_id, name_snapshot, unit_price_kobo, qty, line_kobo`

// NextOrderNo draws the next number inside tx.
func NextOrderNo(ctx context.Context, tx pgx.Tx, prefix string) (string, error) {
	var n int64
	if err := tx.QueryRow(ctx, `SELECT nextval('order_no_seq')`).Scan(&n); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%04d", prefix, n), nil
}

// InsertOrder writes the order and its lines inside tx. ErrConflict means an
// open order with the same cart_hash already exists.
func InsertOrder(ctx context.Context, tx pgx.Tx, o Order) error {
	_, err := tx.Exec(ctx, `INSERT INTO orders (id, order_no, status, customer_name, phone, note, fulfilment, zone_id, address, "window",
		subtotal_kobo, delivery_fee_kobo, total_kobo, cart_hash, source, expires_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`,
		o.ID, o.OrderNo, o.Status, o.CustomerName, o.Phone, o.Note, o.Fulfilment, o.ZoneID, o.Address, o.Window,
		o.SubtotalKobo, o.DeliveryFeeKobo, o.TotalKobo, o.CartHash, o.Source, o.ExpiresAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrConflict
		}
		return err
	}
	for _, l := range o.Lines {
		if _, err := tx.Exec(ctx, `INSERT INTO order_lines (order_id, item_id, name_snapshot, unit_price_kobo, qty, line_kobo)
			VALUES ($1,$2,$3,$4,$5,$6)`, o.ID, l.ItemID, l.NameSnapshot, l.UnitPriceKobo, l.Qty, l.LineKobo); err != nil {
			return err
		}
	}
	return nil
}

// AddEvent appends to the order's audit trail.
func AddEvent(ctx context.Context, q Querier, orderID, typ, actor string, data any) error {
	b := []byte("{}")
	if data != nil {
		var err error
		if b, err = json.Marshal(data); err != nil {
			return err
		}
	}
	_, err := q.Exec(ctx, `INSERT INTO order_events (order_id, type, actor, data) VALUES ($1,$2,$3,$4)`, orderID, typ, actor, b)
	return err
}

func (d *DB) OpenOrderByCartHash(ctx context.Context, hash string) (Order, error) {
	rows, err := d.Pool.Query(ctx, `SELECT `+orderCols+` FROM orders WHERE cart_hash = $1 AND status = 'pending_payment'`, hash)
	if err != nil {
		return Order{}, err
	}
	o, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[Order])
	if isNoRows(err) {
		return Order{}, ErrNotFound
	}
	if err != nil {
		return Order{}, err
	}
	return d.withLines(ctx, d.Pool, o)
}

func (d *DB) OrderByReference(ctx context.Context, ref string) (Order, error) {
	return d.oneOrder(ctx, d.Pool, `reference = $1`, ref)
}

func (d *DB) OrderByNo(ctx context.Context, no string) (Order, error) {
	return d.oneOrder(ctx, d.Pool, `order_no = $1`, no)
}

func (d *DB) OrderByID(ctx context.Context, id string) (Order, error) {
	return d.oneOrder(ctx, d.Pool, `id = $1`, id)
}

func (d *DB) oneOrder(ctx context.Context, q Querier, where string, arg any) (Order, error) {
	rows, err := q.Query(ctx, `SELECT `+orderCols+` FROM orders WHERE `+where, arg)
	if err != nil {
		return Order{}, err
	}
	o, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[Order])
	if isNoRows(err) {
		return Order{}, ErrNotFound
	}
	if err != nil {
		return Order{}, err
	}
	return d.withLines(ctx, q, o)
}

func (d *DB) withLines(ctx context.Context, q Querier, o Order) (Order, error) {
	rows, err := q.Query(ctx, `SELECT `+lineCols+` FROM order_lines WHERE order_id = $1 ORDER BY id`, o.ID)
	if err != nil {
		return o, err
	}
	o.Lines, err = pgx.CollectRows(rows, pgx.RowToStructByName[Line])
	if o.Lines == nil {
		o.Lines = []Line{}
	}
	return o, err
}

// LockOrder re-reads the order FOR UPDATE inside tx.
func LockOrder(ctx context.Context, tx pgx.Tx, id string) (Order, error) {
	rows, err := tx.Query(ctx, `SELECT `+orderCols+` FROM orders WHERE id = $1 FOR UPDATE`, id)
	if err != nil {
		return Order{}, err
	}
	o, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[Order])
	if isNoRows(err) {
		return Order{}, ErrNotFound
	}
	if err != nil {
		return Order{}, err
	}
	lrows, err := tx.Query(ctx, `SELECT `+lineCols+` FROM order_lines WHERE order_id = $1 ORDER BY id`, id)
	if err != nil {
		return o, err
	}
	o.Lines, err = pgx.CollectRows(lrows, pgx.RowToStructByName[Line])
	return o, err
}

// LockPortions locks the given items (ordered by id to avoid deadlocks) and
// returns id → portions.
func LockPortions(ctx context.Context, tx pgx.Tx, ids []string) (map[string]int, error) {
	rows, err := tx.Query(ctx, `SELECT id, portions FROM menu_items WHERE id = ANY($1) ORDER BY id FOR UPDATE`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var id string
		var p int
		if err := rows.Scan(&id, &p); err != nil {
			return nil, err
		}
		out[id] = p
	}
	return out, rows.Err()
}

func DecrementPortions(ctx context.Context, tx pgx.Tx, itemID string, qty int) error {
	tag, err := tx.Exec(ctx, `UPDATE menu_items SET portions = portions - $2, version = version + 1, updated_at = now()
		WHERE id = $1 AND portions >= $2`, itemID, qty)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("store: decrement %s by %d refused", itemID, qty)
	}
	return nil
}

// SetReference records the Paystack transaction on a freshly created order.
func (d *DB) SetReference(ctx context.Context, id, reference, authURL string) error {
	_, err := d.Pool.Exec(ctx, `UPDATE orders SET reference = $2, authorization_url = $3, updated_at = now() WHERE id = $1`, id, reference, authURL)
	return err
}

// MarkPaid flips the order to paid inside tx.
func MarkPaid(ctx context.Context, tx pgx.Tx, id string, paidAt, readyBy time.Time, by string) error {
	_, err := tx.Exec(ctx, `UPDATE orders SET status = 'paid', paid_at = $2, ready_by = $3, confirmed_by = $4, updated_at = now() WHERE id = $1`,
		id, paidAt, readyBy, by)
	return err
}

func MarkNeedsAttention(ctx context.Context, tx pgx.Tx, id string, paidAt time.Time, by, reason string) error {
	_, err := tx.Exec(ctx, `UPDATE orders SET status = 'paid_needs_attention', paid_at = $2, confirmed_by = $3, attention_reason = $4, updated_at = now() WHERE id = $1`,
		id, paidAt, by, reason)
	return err
}

func (d *DB) SetStatus(ctx context.Context, id, status string) error {
	_, err := d.Pool.Exec(ctx, `UPDATE orders SET status = $2, updated_at = now() WHERE id = $1`, id, status)
	return err
}

func (d *DB) Resolve(ctx context.Context, id, note string) error {
	_, err := d.Pool.Exec(ctx, `UPDATE orders SET status = 'resolved', resolved_at = now(), resolution_note = $2, updated_at = now() WHERE id = $1`, id, note)
	return err
}

func (d *DB) Cancel(ctx context.Context, id, reason string) error {
	_, err := d.Pool.Exec(ctx, `UPDATE orders SET status = 'cancelled', cancel_reason = $2, updated_at = now() WHERE id = $1`, id, reason)
	return err
}

func (d *DB) TouchReconciled(ctx context.Context, id string) error {
	_, err := d.Pool.Exec(ctx, `UPDATE orders SET reconciled_at = now() WHERE id = $1`, id)
	return err
}

// ---- listing with cursor pagination ----

type OrdersFilter struct {
	Statuses []string
	Today    bool
	Cursor   string
	Limit    int
}

type Page[T any] struct {
	Items      []T
	NextCursor string
	HasMore    bool
}

func encodeCursor(t time.Time, id string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(t.UTC().Format(time.RFC3339Nano) + "|" + id))
}

func decodeCursor(c string) (time.Time, string, bool) {
	b, err := base64.RawURLEncoding.DecodeString(c)
	if err != nil {
		return time.Time{}, "", false
	}
	parts := strings.SplitN(string(b), "|", 2)
	if len(parts) != 2 {
		return time.Time{}, "", false
	}
	t, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return time.Time{}, "", false
	}
	return t, parts[1], true
}

func (d *DB) ListOrders(ctx context.Context, f OrdersFilter, lagosDayStart time.Time) (Page[Order], error) {
	if f.Limit <= 0 || f.Limit > 100 {
		f.Limit = 50
	}
	var where []string
	var args []any
	if len(f.Statuses) > 0 {
		args = append(args, f.Statuses)
		where = append(where, fmt.Sprintf("status = ANY($%d)", len(args)))
	}
	if f.Today {
		args = append(args, lagosDayStart)
		where = append(where, fmt.Sprintf("created_at >= $%d", len(args)))
	}
	if f.Cursor != "" {
		t, id, ok := decodeCursor(f.Cursor)
		if ok {
			args = append(args, t, id)
			where = append(where, fmt.Sprintf("(created_at, id) < ($%d, $%d)", len(args)-1, len(args)))
		}
	}
	q := `SELECT ` + orderCols + ` FROM orders`
	if len(where) > 0 {
		q += " WHERE " + strings.Join(where, " AND ")
	}
	args = append(args, f.Limit+1)
	q += fmt.Sprintf(" ORDER BY created_at DESC, id DESC LIMIT $%d", len(args))

	rows, err := d.Pool.Query(ctx, q, args...)
	if err != nil {
		return Page[Order]{}, err
	}
	orders, err := pgx.CollectRows(rows, pgx.RowToStructByName[Order])
	if err != nil {
		return Page[Order]{}, err
	}
	page := Page[Order]{Items: []Order{}}
	if len(orders) > f.Limit {
		page.HasMore = true
		orders = orders[:f.Limit]
	}
	for i := range orders {
		orders[i], err = d.withLines(ctx, d.Pool, orders[i])
		if err != nil {
			return page, err
		}
	}
	page.Items = orders
	if page.HasMore && len(orders) > 0 {
		last := orders[len(orders)-1]
		page.NextCursor = encodeCursor(last.CreatedAt, last.ID)
	}
	return page, nil
}

// PendingForReconcile returns pending orders with a reference, created more
// than minAge ago, not reconciled in the last recheck interval.
func (d *DB) PendingForReconcile(ctx context.Context, minAge, recheck time.Duration, limit int) ([]Order, error) {
	rows, err := d.Pool.Query(ctx, `SELECT `+orderCols+` FROM orders
		WHERE status IN ('pending_payment','expired') AND reference IS NOT NULL
		  AND created_at < now() - $1::interval
		  AND (reconciled_at IS NULL OR reconciled_at < now() - $2::interval)
		ORDER BY created_at LIMIT $3`, minAge, recheck, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[Order])
}

// CancelReferenceless cancels pending orders that never got a Paystack reference.
func (d *DB) CancelReferenceless(ctx context.Context, olderThan time.Duration) ([]string, error) {
	rows, err := d.Pool.Query(ctx, `UPDATE orders SET status = 'cancelled', cancel_reason = 'init_lost', updated_at = now()
		WHERE status = 'pending_payment' AND reference IS NULL AND created_at < now() - $1::interval RETURNING id`, olderThan)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

// ExpirePending marks pending orders past their TTL as expired.
func (d *DB) ExpirePending(ctx context.Context) ([]string, error) {
	rows, err := d.Pool.Query(ctx, `UPDATE orders SET status = 'expired', updated_at = now()
		WHERE status = 'pending_payment' AND expires_at < now() RETURNING id`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

func (d *DB) Events(ctx context.Context, orderID string) ([]OrderEvent, error) {
	rows, err := d.Pool.Query(ctx, `SELECT id, order_id, type, actor, data, created_at FROM order_events WHERE order_id = $1 ORDER BY id`, orderID)
	if err != nil {
		return nil, err
	}
	ev, err := pgx.CollectRows(rows, pgx.RowToStructByName[OrderEvent])
	if ev == nil {
		ev = []OrderEvent{}
	}
	return ev, err
}

func (d *DB) InsertPaymentCheck(ctx context.Context, c PaymentCheck) error {
	_, err := d.Pool.Exec(ctx, `INSERT INTO payment_checks (order_id, reference, source, paystack_status, amount_kobo, currency, gateway_response, outcome, latency_ms, raw)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		c.OrderID, c.Reference, c.Source, c.PaystackStatus, c.AmountKobo, c.Currency, c.GatewayResponse, c.Outcome, c.LatencyMs, c.Raw)
	return err
}

func (d *DB) PaymentChecks(ctx context.Context, orderID string) ([]PaymentCheck, error) {
	rows, err := d.Pool.Query(ctx, `SELECT id, order_id, reference, source, paystack_status, amount_kobo, currency, gateway_response, outcome, latency_ms, raw, checked_at
		FROM payment_checks WHERE order_id = $1 ORDER BY id`, orderID)
	if err != nil {
		return nil, err
	}
	pc, err := pgx.CollectRows(rows, pgx.RowToStructByName[PaymentCheck])
	if pc == nil {
		pc = []PaymentCheck{}
	}
	return pc, err
}

// Stats for today's board tiles.
type DayStats struct {
	PaidCount      int
	PaidTotalKobo  int64
	AwaitingCount  int
	AttentionCount int
	CancelledCount int
	ExpiredCount   int
}

func (d *DB) StatsSince(ctx context.Context, since time.Time) (DayStats, error) {
	var s DayStats
	err := d.Pool.QueryRow(ctx, `SELECT
		count(*) FILTER (WHERE status IN ('paid','resolved')),
		COALESCE(sum(total_kobo) FILTER (WHERE status IN ('paid','resolved','paid_needs_attention')), 0),
		count(*) FILTER (WHERE status = 'pending_payment'),
		count(*) FILTER (WHERE status = 'paid_needs_attention'),
		count(*) FILTER (WHERE status = 'cancelled'),
		count(*) FILTER (WHERE status = 'expired')
		FROM orders WHERE created_at >= $1`, since).
		Scan(&s.PaidCount, &s.PaidTotalKobo, &s.AwaitingCount, &s.AttentionCount, &s.CancelledCount, &s.ExpiredCount)
	return s, err
}

// Prune removes old diagnostic rows.
func (d *DB) Prune(ctx context.Context, olderThan time.Duration) error {
	for _, q := range []string{
		`DELETE FROM tool_calls WHERE created_at < now() - $1::interval`,
		`DELETE FROM payment_checks WHERE checked_at < now() - $1::interval`,
	} {
		if _, err := d.Pool.Exec(ctx, q, olderThan); err != nil {
			return err
		}
	}
	return nil
}
