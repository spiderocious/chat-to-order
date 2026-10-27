package httpapi

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/spiderocious/chat-to-order/backend/internal/clock"
	"github.com/spiderocious/chat-to-order/backend/internal/money"
	"github.com/spiderocious/chat-to-order/backend/internal/ordering"
	"github.com/spiderocious/chat-to-order/backend/internal/store"
)

// Wire views. snake_case, money twice, ISO dates plus WAT labels. This file
// is the serialiser the frontend types are copied from.

type itemView struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	Description  string     `json:"description"`
	PriceKobo    money.Kobo `json:"price_kobo"`
	PriceDisplay string     `json:"price_display"`
	Portions     int        `json:"portions"`
	Available    bool       `json:"available"`
	Window       string     `json:"window"`
	Sort         int        `json:"sort"`
	PhotoURL     *string    `json:"photo_url"`
	OfferedNow   bool       `json:"offered_now"`
	Low          bool       `json:"low"`
	Version      int        `json:"version"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

func toItemView(it store.Item, window string, lowThreshold int) itemView {
	offered := it.Available && it.Portions > 0 && (it.Window == "all" || it.Window == window)
	return itemView{ID: it.ID, Name: it.Name, Description: it.Description, PriceKobo: it.PriceKobo, PriceDisplay: it.PriceKobo.Format(),
		Portions: it.Portions, Available: it.Available, Window: it.Window, Sort: it.Sort, PhotoURL: it.PhotoURL,
		OfferedNow: offered, Low: it.Portions <= lowThreshold, Version: it.Version, UpdatedAt: it.UpdatedAt}
}

type zoneView struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	FeeKobo    money.Kobo `json:"fee_kobo"`
	FeeDisplay string     `json:"fee_display"`
	Active     bool       `json:"active"`
	Sort       int        `json:"sort"`
}

func toZoneView(z store.Zone) zoneView {
	return zoneView{ID: z.ID, Name: z.Name, FeeKobo: z.FeeKobo, FeeDisplay: z.FeeKobo.Format(), Active: z.Active, Sort: z.Sort}
}

type windowView struct {
	Open        string `json:"open"`
	Close       string `json:"close"`
	OrdersClose string `json:"orders_close"`
}

type nowView struct {
	WAT           string            `json:"wat"`
	Window        string            `json:"window"`
	OrdersOpen    bool              `json:"orders_open"`
	OrdersCloseAt string            `json:"orders_close_at,omitempty"`
	NextWindow    *clock.NextWindow `json:"next_window"`
}

type vendorView struct {
	VendorName        string                `json:"vendor_name"`
	PickupAddress     string                `json:"pickup_address"`
	RiderPhone        string                `json:"rider_phone"`
	OrderPrefix       string                `json:"order_prefix"`
	PrepMinutes       int                   `json:"prep_minutes"`
	PendingTTLMinutes int                   `json:"pending_ttl_minutes"`
	LowStockThreshold int                   `json:"low_stock_threshold"`
	Windows           map[string]windowView `json:"windows"`
	Now               nowView               `json:"now"`
	Version           int                   `json:"version"`
	UpdatedAt         time.Time             `json:"updated_at"`
}

func toVendorView(s store.Settings, st clock.WindowState) vendorView {
	return vendorView{
		VendorName: s.VendorName, PickupAddress: s.PickupAddress, RiderPhone: s.RiderPhone, OrderPrefix: s.OrderPrefix,
		PrepMinutes: s.PrepMinutes, PendingTTLMinutes: s.PendingTTLMinutes, LowStockThreshold: s.LowStockThreshold,
		Windows: map[string]windowView{
			"lunch":   {Open: s.LunchOpen, Close: s.LunchClose, OrdersClose: s.LunchOrdersClose},
			"evening": {Open: s.EveningOpen, Close: s.EveningClose, OrdersClose: s.EveningOrdersClose},
		},
		Now:     nowView{WAT: st.NowWAT, Window: st.Name, OrdersOpen: st.OrdersOpen, OrdersCloseAt: st.OrdersCloseAt, NextWindow: st.Next},
		Version: s.Version, UpdatedAt: s.UpdatedAt,
	}
}

type zoneRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type orderRow struct {
	OrderNo         string     `json:"order_no"`
	Status          string     `json:"status"`
	CustomerName    string     `json:"customer_name"`
	PhoneMasked     *string    `json:"phone_masked"`
	Fulfilment      string     `json:"fulfilment"`
	Zone            *zoneRef   `json:"zone"`
	Address         *string    `json:"address"`
	Window          string     `json:"window"`
	LinesSummary    string     `json:"lines_summary"`
	TotalKobo       money.Kobo `json:"total_kobo"`
	TotalDisplay    string     `json:"total_display"`
	Reference       *string    `json:"reference"`
	PaidAt          *time.Time `json:"paid_at"`
	ReadyBy         *time.Time `json:"ready_by"`
	ReadyByWAT      *string    `json:"ready_by_wat"`
	ConfirmedBy     *string    `json:"confirmed_by"`
	AttentionReason *string    `json:"attention_reason"`
	ExpiresAt       time.Time  `json:"expires_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type lineView struct {
	ItemID        string     `json:"item_id"`
	Name          string     `json:"name"`
	Qty           int        `json:"qty"`
	UnitPriceKobo money.Kobo `json:"unit_price_kobo"`
	UnitDisplay   string     `json:"unit_display"`
	LineKobo      money.Kobo `json:"line_kobo"`
	LineDisplay   string     `json:"line_display"`
}

type eventView struct {
	Type  string          `json:"type"`
	Actor string          `json:"actor"`
	Data  json.RawMessage `json:"data"`
	At    time.Time       `json:"at"`
}

type checkView struct {
	CheckedAt       time.Time `json:"checked_at"`
	Source          string    `json:"source"`
	PaystackStatus  *string   `json:"paystack_status"`
	Outcome         string    `json:"outcome"`
	AmountKobo      *int64    `json:"amount_kobo"`
	GatewayResponse *string   `json:"gateway_response"`
	LatencyMs       *int      `json:"latency_ms"`
}

type toolCallBrief struct {
	ID        int64     `json:"id"`
	Tool      string    `json:"tool"`
	OK        bool      `json:"ok"`
	ErrorCode *string   `json:"error_code"`
	LatencyMs int       `json:"latency_ms"`
	CreatedAt time.Time `json:"created_at"`
}

type orderDetail struct {
	orderRow
	Phone              *string         `json:"phone"`
	Note               *string         `json:"note"`
	Source             string          `json:"source"`
	AuthorizationURL   *string         `json:"authorization_url"`
	SubtotalKobo       money.Kobo      `json:"subtotal_kobo"`
	SubtotalDisplay    string          `json:"subtotal_display"`
	DeliveryFeeKobo    money.Kobo      `json:"delivery_fee_kobo"`
	DeliveryFeeDisplay string          `json:"delivery_fee_display"`
	Lines              []lineView      `json:"lines"`
	Events             []eventView     `json:"events"`
	PaymentChecks      []checkView     `json:"payment_checks"`
	ToolCalls          []toolCallBrief `json:"tool_calls"`
	ResolvedAt         *time.Time      `json:"resolved_at"`
	ResolutionNote     *string         `json:"resolution_note"`
	CancelReason       *string         `json:"cancel_reason"`
}

func maskPhone(p *string) *string {
	if p == nil || len(*p) < 8 {
		return p
	}
	s := *p
	m := s[:4] + "•••" + s[len(s)-4:]
	return &m
}

func linesSummary(lines []store.Line) string {
	parts := make([]string, 0, len(lines))
	for _, l := range lines {
		parts = append(parts, strconv.Itoa(l.Qty)+"× "+l.NameSnapshot)
	}
	return strings.Join(parts, ", ")
}

func toOrderRow(o store.Order, snap ordering.Snapshot) orderRow {
	row := orderRow{
		OrderNo: o.OrderNo, Status: o.Status, CustomerName: o.CustomerName, PhoneMasked: maskPhone(o.Phone),
		Fulfilment: o.Fulfilment, Address: o.Address, Window: o.Window, LinesSummary: linesSummary(o.Lines),
		TotalKobo: o.TotalKobo, TotalDisplay: o.TotalKobo.Format(), Reference: o.Reference, PaidAt: o.PaidAt,
		ReadyBy: o.ReadyBy, ConfirmedBy: o.ConfirmedBy, AttentionReason: o.AttentionReason,
		ExpiresAt: o.ExpiresAt, CreatedAt: o.CreatedAt, UpdatedAt: o.UpdatedAt,
	}
	if o.ReadyBy != nil {
		w := clock.WAT(*o.ReadyBy)
		row.ReadyByWAT = &w
	}
	if o.ZoneID != nil {
		for _, z := range snap.Zones {
			if z.ID == *o.ZoneID {
				row.Zone = &zoneRef{ID: z.ID, Name: z.Name}
			}
		}
	}
	return row
}

func toOrderDetail(o store.Order, snap ordering.Snapshot, events []store.OrderEvent, checks []store.PaymentCheck, calls []store.ToolCall) orderDetail {
	d := orderDetail{
		orderRow: toOrderRow(o, snap), Phone: o.Phone, Note: o.Note, Source: o.Source, AuthorizationURL: o.AuthorizationURL,
		SubtotalKobo: o.SubtotalKobo, SubtotalDisplay: o.SubtotalKobo.Format(),
		DeliveryFeeKobo: o.DeliveryFeeKobo, DeliveryFeeDisplay: o.DeliveryFeeKobo.Format(),
		Lines: []lineView{}, Events: []eventView{}, PaymentChecks: []checkView{}, ToolCalls: []toolCallBrief{},
		ResolvedAt: o.ResolvedAt, ResolutionNote: o.ResolutionNote, CancelReason: o.CancelReason,
	}
	for _, l := range o.Lines {
		d.Lines = append(d.Lines, lineView{ItemID: l.ItemID, Name: l.NameSnapshot, Qty: l.Qty, UnitPriceKobo: l.UnitPriceKobo,
			UnitDisplay: l.UnitPriceKobo.Format(), LineKobo: l.LineKobo, LineDisplay: l.LineKobo.Format()})
	}
	for _, e := range events {
		data := json.RawMessage(e.Data)
		if len(data) == 0 {
			data = json.RawMessage("{}")
		}
		d.Events = append(d.Events, eventView{Type: e.Type, Actor: e.Actor, Data: data, At: e.CreatedAt})
	}
	for _, c := range checks {
		d.PaymentChecks = append(d.PaymentChecks, checkView{CheckedAt: c.CheckedAt, Source: c.Source, PaystackStatus: c.PaystackStatus,
			Outcome: c.Outcome, AmountKobo: c.AmountKobo, GatewayResponse: c.GatewayResponse, LatencyMs: c.LatencyMs})
	}
	for _, t := range calls {
		d.ToolCalls = append(d.ToolCalls, toolCallBrief{ID: t.ID, Tool: t.Tool, OK: t.OK, ErrorCode: t.ErrorCode, LatencyMs: t.LatencyMs, CreatedAt: t.CreatedAt})
	}
	return d
}

type toolCallView struct {
	toolCallBrief
	OrderNo   *string         `json:"order_no"`
	RequestID *string         `json:"request_id"`
	Args      json.RawMessage `json:"args"`
	Result    json.RawMessage `json:"result"`
}

func toToolCallView(t store.ToolCall) toolCallView {
	res := json.RawMessage(t.Result)
	if len(res) == 0 {
		res = json.RawMessage("null")
	}
	return toolCallView{toolCallBrief: toolCallBrief{ID: t.ID, Tool: t.Tool, OK: t.OK, ErrorCode: t.ErrorCode, LatencyMs: t.LatencyMs, CreatedAt: t.CreatedAt},
		OrderNo: t.OrderNo, RequestID: t.RequestID, Args: json.RawMessage(t.Args), Result: res}
}
