// Package mcpserver exposes the four ordering tools over MCP Streamable HTTP.
// Each handler is thin: decode, call the domain, encode the result or the
// domain error. Every call is recorded by the activity logger.
package mcpserver

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/spiderocious/chat-to-order/backend/internal/activity"
	"github.com/spiderocious/chat-to-order/backend/internal/ordering"
)

type Deps struct {
	Svc      *ordering.Service
	Activity *activity.Logger
	Token    string
	Version  string
	OnCall   func(tool string, ok bool, code string, d time.Duration)
}

// Handler returns the http.Handler for POST /mcp, with bearer auth applied.
func Handler(d Deps) http.Handler {
	srv := mcp.NewServer(&mcp.Implementation{Name: "iya-bisi-ordering", Version: d.Version}, nil)
	register(srv, d)
	h := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return srv }, &mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true})
	return bearer(d.Token, h)
}

func bearer(token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if got == "" || subtle.ConstantTimeCompare([]byte(got), []byte(token)) != 1 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":{"code":"unauthenticated","message":"Missing or invalid bearer token.","severity":"auth"}}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ---------- inputs ----------

type CreateOrderInput struct {
	CustomerName string          `json:"customer_name" jsonschema:"The customer's name, 1 to 80 characters"`
	Items        []CartLineInput `json:"items" jsonschema:"One line per menu item: item_id from get_menu and qty 1 to 20"`
	Fulfilment   string          `json:"fulfilment" jsonschema:"delivery or pickup"`
	ZoneID       string          `json:"zone_id,omitempty" jsonschema:"Delivery zone id from get_menu; required for delivery"`
	Address      string          `json:"address,omitempty" jsonschema:"Street address, 5 to 200 characters; required for delivery"`
	Phone        string          `json:"phone,omitempty" jsonschema:"Customer phone number, optional"`
	Note         string          `json:"note,omitempty" jsonschema:"Special instruction, up to 140 characters, optional"`
}

type CartLineInput struct {
	ItemID string `json:"item_id" jsonschema:"Menu item id, exactly as returned by get_menu"`
	Qty    int    `json:"qty" jsonschema:"Quantity, 1 to 20"`
}

type ReferenceInput struct {
	Reference string `json:"reference" jsonschema:"The Paystack reference returned by create_order, like IBK-0231-a1b2"`
}

// toolError renders a DomainError as the MCP tool error payload.
func toolError(e *ordering.DomainError) *mcp.CallToolResult {
	body := map[string]any{"error": e.Code, "message": e.Message, "retryable": e.Retryable}
	for k, v := range e.Fields {
		body[k] = v
	}
	b, _ := json.Marshal(body)
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(b)}}, IsError: true}
}

func toolOK(v any) *mcp.CallToolResult {
	b, _ := json.Marshal(v)
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(b)}}, StructuredContent: v}
}

// wrap records the call and converts errors.
func wrap[In any](d Deps, name string, fn func(ctx context.Context, in In) (any, string, error)) mcp.ToolHandlerFor[In, any] {
	return func(ctx context.Context, req *mcp.CallToolRequest, in In) (*mcp.CallToolResult, any, error) {
		start := time.Now()
		var reqID string
		if req != nil && req.Extra != nil && req.Extra.Header != nil {
			reqID = req.Extra.Header.Get("X-Request-ID")
		}
		out, orderNo, err := fn(ctx, in)
		lat := time.Since(start)
		entry := activity.Entry{Tool: name, Args: in, Latency: lat, OrderNo: orderNo, RequestID: reqID}
		var res *mcp.CallToolResult
		if err != nil {
			var de *ordering.DomainError
			if !errors.As(err, &de) {
				de = ordering.ErrStoreUnavailable("read")
			}
			entry.ErrorCode = de.Code
			entry.Result = map[string]any{"error": de.Code, "message": de.Message}
			res = toolError(de)
		} else {
			entry.Result = out
			res = toolOK(out)
		}
		d.Activity.Record(ctx, entry)
		if d.OnCall != nil {
			d.OnCall(name, err == nil, entry.ErrorCode, lat)
		}
		// Return nil output so the SDK does not re-populate content from it.
		return res, nil, nil
	}
}

func register(srv *mcp.Server, d Deps) {
	mcp.AddTool(srv, &mcp.Tool{Name: "get_menu", Description: descGetMenu, InputSchema: emptySchema},
		wrap(d, "get_menu", func(ctx context.Context, _ map[string]any) (any, string, error) {
			m, err := d.Svc.Menu(ctx)
			return m, "", err
		}))
	mcp.AddTool(srv, &mcp.Tool{Name: "create_order", Description: descCreateOrder, InputSchema: createOrderSchema},
		wrap(d, "create_order", func(ctx context.Context, raw map[string]any) (any, string, error) {
			in := decodeCreateOrder(raw)
			lines := make([]ordering.CartLine, 0, len(in.Items))
			for _, l := range in.Items {
				lines = append(lines, ordering.CartLine{ItemID: l.ItemID, Qty: l.Qty})
			}
			out, err := d.Svc.CreateOrder(ctx, ordering.CreateOrderInput{
				CustomerName: in.CustomerName, Items: lines, Fulfilment: strings.ToLower(in.Fulfilment),
				ZoneID: in.ZoneID, Address: in.Address, Phone: in.Phone, Note: in.Note, Source: "mcp",
			})
			return out, out.OrderNo, err
		}))
	mcp.AddTool(srv, &mcp.Tool{Name: "verify_payment", Description: descVerify, InputSchema: referenceSchema},
		wrap(d, "verify_payment", func(ctx context.Context, raw map[string]any) (any, string, error) {
			out, err := d.Svc.VerifyPayment(ctx, decodeReference(raw), "mcp")
			return out, out.OrderNo, err
		}))
	mcp.AddTool(srv, &mcp.Tool{Name: "confirm_order", Description: descConfirm, InputSchema: referenceSchema},
		wrap(d, "confirm_order", func(ctx context.Context, raw map[string]any) (any, string, error) {
			out, err := d.Svc.ConfirmOrder(ctx, decodeReference(raw), "mcp")
			return out, out.OrderNo, err
		}))
}

const descGetMenu = `Get what Iya Bisi's Kitchen is offering right now: the current window (lunch, evening or closed), whether orders are open, every item with its price_display and portions_left, delivery zones with fees, and the pickup address. Call this at the start of every order and again if a few minutes have passed. Never quote a price or a portion count from memory; always from this tool.`

const descCreateOrder = `Create an order and get a Paystack payment link. Validates every item id and quantity against current stock and the cut-off time, computes the total with the delivery fee, and returns the lines, totals as display strings, the reference and the authorization_url. Repeat the lines and total_display to the customer exactly, send the link, and ask them to say 'paid' when done. If the same cart is sent again while the first is still open, the same order and link come back (reused=true). On item_unavailable, offer max_qty or a swap; on window_closed, say when the next window opens.`

const descVerify = `Ask Paystack whether a reference has been paid. Returns outcome: success, not_confirmed, failed or amount_mismatch, with a message and next_step. Call this when the customer says they have paid. On success, call confirm_order next. not_confirmed means not yet (never say 'failed'); failed carries the bank's reason; amount_mismatch is flagged for the vendor.`

const descConfirm = `Confirm a paid order: re-checks the payment with Paystack, reserves the portions and marks the order paid with a ready-by time. Idempotent: calling it again for the same reference returns the same confirmed order. Only after this returns status 'paid' may you tell the customer the order is confirmed. If status is 'paid_needs_attention', an item ran out after payment; apologise and say Iya Bisi will contact them, and do not promise a refund.`
