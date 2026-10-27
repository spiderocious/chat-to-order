// Package activity records every tool call: what the model asked, what it got,
// how long it took. The board reads it; the field notes quote it.
package activity

import (
	"context"
	"encoding/json"
	"log/slog"
	"regexp"
	"time"

	"github.com/spiderocious/chat-to-order/backend/internal/store"
)

type Logger struct {
	db  *store.DB
	log *slog.Logger
}

func New(db *store.DB, log *slog.Logger) *Logger { return &Logger{db: db, log: log} }

type Entry struct {
	Tool      string
	Args      any
	Result    any
	ErrorCode string
	Latency   time.Duration
	OrderNo   string
	RequestID string
}

var phoneRe = regexp.MustCompile(`\d{7,}`)

// mask hides phone digits and street addresses in the stored args.
func mask(args any) any {
	m, ok := args.(map[string]any)
	if !ok {
		b, _ := json.Marshal(args)
		var mm map[string]any
		if json.Unmarshal(b, &mm) != nil {
			return args
		}
		m = mm
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		switch k {
		case "phone":
			if s, ok := v.(string); ok && len(s) > 4 {
				out[k] = phoneRe.ReplaceAllStringFunc(s, func(d string) string { return d[:4] + "•••" + d[len(d)-4:] })
				continue
			}
		case "address":
			if s, ok := v.(string); ok && len(s) > 6 {
				out[k] = "•••• " + s[len(s)-6:]
				continue
			}
		}
		out[k] = v
	}
	return out
}

func (l *Logger) Record(ctx context.Context, e Entry) {
	args, _ := json.Marshal(mask(e.Args))
	var result []byte
	if e.Result != nil {
		result, _ = json.Marshal(e.Result)
	}
	tc := store.ToolCall{
		Tool: e.Tool, Args: args, Result: result, OK: e.ErrorCode == "",
		LatencyMs: int(e.Latency.Milliseconds()),
	}
	if e.ErrorCode != "" {
		tc.ErrorCode = &e.ErrorCode
	}
	if e.OrderNo != "" {
		tc.OrderNo = &e.OrderNo
	}
	if e.RequestID != "" {
		tc.RequestID = &e.RequestID
	}
	// Never let logging fail a tool call; use a detached context so a cancelled
	// request still gets its row.
	wctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
	defer cancel()
	if err := l.db.InsertToolCall(wctx, tc); err != nil {
		l.log.Error("activity.insert", "err", err, "tool", e.Tool)
	}
	lvl := slog.LevelInfo
	if e.ErrorCode != "" {
		lvl = slog.LevelWarn
	}
	l.log.Log(ctx, lvl, "tool", "tool", e.Tool, "ok", e.ErrorCode == "", "error_code", e.ErrorCode,
		"latency_ms", e.Latency.Milliseconds(), "order_no", e.OrderNo, "request_id", e.RequestID)
}
