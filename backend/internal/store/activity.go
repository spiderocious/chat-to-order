package store

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

const toolCallCols = `id, tool, args, result, ok, error_code, latency_ms, order_no, request_id, created_at`

func (d *DB) InsertToolCall(ctx context.Context, c ToolCall) error {
	_, err := d.Pool.Exec(ctx, `INSERT INTO tool_calls (tool, args, result, ok, error_code, latency_ms, order_no, request_id)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`, c.Tool, c.Args, c.Result, c.OK, c.ErrorCode, c.LatencyMs, c.OrderNo, c.RequestID)
	return err
}

type ActivityFilter struct {
	Tool    string
	OK      *bool
	OrderNo string
	Cursor  string // numeric id
	Limit   int
}

func (d *DB) ListToolCalls(ctx context.Context, f ActivityFilter) (Page[ToolCall], error) {
	if f.Limit <= 0 || f.Limit > 100 {
		f.Limit = 30
	}
	var where []string
	var args []any
	if f.Tool != "" {
		args = append(args, f.Tool)
		where = append(where, fmt.Sprintf("tool = $%d", len(args)))
	}
	if f.OK != nil {
		args = append(args, *f.OK)
		where = append(where, fmt.Sprintf("ok = $%d", len(args)))
	}
	if f.OrderNo != "" {
		args = append(args, f.OrderNo)
		where = append(where, fmt.Sprintf("order_no = $%d", len(args)))
	}
	if f.Cursor != "" {
		if id, err := strconv.ParseInt(f.Cursor, 10, 64); err == nil {
			args = append(args, id)
			where = append(where, fmt.Sprintf("id < $%d", len(args)))
		}
	}
	q := `SELECT ` + toolCallCols + ` FROM tool_calls`
	if len(where) > 0 {
		q += " WHERE " + strings.Join(where, " AND ")
	}
	args = append(args, f.Limit+1)
	q += fmt.Sprintf(" ORDER BY id DESC LIMIT $%d", len(args))
	rows, err := d.Pool.Query(ctx, q, args...)
	if err != nil {
		return Page[ToolCall]{}, err
	}
	calls, err := pgx.CollectRows(rows, pgx.RowToStructByName[ToolCall])
	if err != nil {
		return Page[ToolCall]{}, err
	}
	page := Page[ToolCall]{Items: []ToolCall{}}
	if len(calls) > f.Limit {
		page.HasMore = true
		calls = calls[:f.Limit]
	}
	page.Items = calls
	if page.HasMore && len(calls) > 0 {
		page.NextCursor = strconv.FormatInt(calls[len(calls)-1].ID, 10)
	}
	return page, nil
}

func (d *DB) ToolCallsForOrder(ctx context.Context, orderNo string) ([]ToolCall, error) {
	rows, err := d.Pool.Query(ctx, `SELECT `+toolCallCols+` FROM tool_calls WHERE order_no = $1 ORDER BY id`, orderNo)
	if err != nil {
		return nil, err
	}
	c, err := pgx.CollectRows(rows, pgx.RowToStructByName[ToolCall])
	if c == nil {
		c = []ToolCall{}
	}
	return c, err
}
