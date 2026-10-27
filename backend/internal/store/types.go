package store

import (
	"time"

	"github.com/spiderocious/chat-to-order/backend/internal/money"
)

type Settings struct {
	VendorName         string    `db:"vendor_name"`
	PickupAddress      string    `db:"pickup_address"`
	RiderPhone         string    `db:"rider_phone"`
	OrderPrefix        string    `db:"order_prefix"`
	PrepMinutes        int       `db:"prep_minutes"`
	LunchOpen          string    `db:"lunch_open"`
	LunchClose         string    `db:"lunch_close"`
	LunchOrdersClose   string    `db:"lunch_orders_close"`
	EveningOpen        string    `db:"evening_open"`
	EveningClose       string    `db:"evening_close"`
	EveningOrdersClose string    `db:"evening_orders_close"`
	PendingTTLMinutes  int       `db:"pending_ttl_minutes"`
	LowStockThreshold  int       `db:"low_stock_threshold"`
	Version            int       `db:"version"`
	UpdatedAt          time.Time `db:"updated_at"`
}

type Item struct {
	ID          string     `db:"id"`
	Name        string     `db:"name"`
	Description string     `db:"description"`
	PriceKobo   money.Kobo `db:"price_kobo"`
	Portions    int        `db:"portions"`
	Available   bool       `db:"available"`
	Window      string     `db:"window"`
	Sort        int        `db:"sort"`
	PhotoURL    *string    `db:"photo_url"`
	Version     int        `db:"version"`
	CreatedAt   time.Time  `db:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at"`
}

type Zone struct {
	ID      string     `db:"id"`
	Name    string     `db:"name"`
	FeeKobo money.Kobo `db:"fee_kobo"`
	Active  bool       `db:"active"`
	Sort    int        `db:"sort"`
}

type Order struct {
	ID              string     `db:"id"`
	OrderNo         string     `db:"order_no"`
	Status          string     `db:"status"`
	CustomerName    string     `db:"customer_name"`
	Phone           *string    `db:"phone"`
	Note            *string    `db:"note"`
	Fulfilment      string     `db:"fulfilment"`
	ZoneID          *string    `db:"zone_id"`
	Address         *string    `db:"address"`
	Window          string     `db:"window"`
	SubtotalKobo    money.Kobo `db:"subtotal_kobo"`
	DeliveryFeeKobo money.Kobo `db:"delivery_fee_kobo"`
	TotalKobo       money.Kobo `db:"total_kobo"`
	Reference       *string    `db:"reference"`
	AuthorizationURL *string   `db:"authorization_url"`
	CartHash        *string    `db:"cart_hash"`
	Source          string     `db:"source"`
	PaidAt          *time.Time `db:"paid_at"`
	ReadyBy         *time.Time `db:"ready_by"`
	ConfirmedBy     *string    `db:"confirmed_by"`
	AttentionReason *string    `db:"attention_reason"`
	ResolvedAt      *time.Time `db:"resolved_at"`
	ResolutionNote  *string    `db:"resolution_note"`
	CancelReason    *string    `db:"cancel_reason"`
	ExpiresAt       time.Time  `db:"expires_at"`
	ReconciledAt    *time.Time `db:"reconciled_at"`
	CreatedAt       time.Time  `db:"created_at"`
	UpdatedAt       time.Time  `db:"updated_at"`

	Lines []Line `db:"-"`
}

type Line struct {
	ID            int64      `db:"id"`
	OrderID       string     `db:"order_id"`
	ItemID        string     `db:"item_id"`
	NameSnapshot  string     `db:"name_snapshot"`
	UnitPriceKobo money.Kobo `db:"unit_price_kobo"`
	Qty           int        `db:"qty"`
	LineKobo      money.Kobo `db:"line_kobo"`
}

type OrderEvent struct {
	ID        int64     `db:"id"`
	OrderID   string    `db:"order_id"`
	Type      string    `db:"type"`
	Actor     string    `db:"actor"`
	Data      []byte    `db:"data"`
	CreatedAt time.Time `db:"created_at"`
}

type PaymentCheck struct {
	ID              int64      `db:"id"`
	OrderID         *string    `db:"order_id"`
	Reference       string     `db:"reference"`
	Source          string     `db:"source"`
	PaystackStatus  *string    `db:"paystack_status"`
	AmountKobo      *int64     `db:"amount_kobo"`
	Currency        *string    `db:"currency"`
	GatewayResponse *string    `db:"gateway_response"`
	Outcome         string     `db:"outcome"`
	LatencyMs       *int       `db:"latency_ms"`
	Raw             []byte     `db:"raw"`
	CheckedAt       time.Time  `db:"checked_at"`
}

type ToolCall struct {
	ID        int64     `db:"id"`
	Tool      string    `db:"tool"`
	Args      []byte    `db:"args"`
	Result    []byte    `db:"result"`
	OK        bool      `db:"ok"`
	ErrorCode *string   `db:"error_code"`
	LatencyMs int       `db:"latency_ms"`
	OrderNo   *string   `db:"order_no"`
	RequestID *string   `db:"request_id"`
	CreatedAt time.Time `db:"created_at"`
}

type Admin struct {
	ID           string    `db:"id"`
	Email        string    `db:"email"`
	Name         string    `db:"name"`
	PasswordHash string    `db:"password_hash"`
	TokenVersion int       `db:"token_version"`
	CreatedAt    time.Time `db:"created_at"`
}

// Order status vocabulary. The CHECK constraint in the schema is the source of
// truth; these constants are the Go spelling.
const (
	StatusPending        = "pending_payment"
	StatusPaid           = "paid"
	StatusNeedsAttention = "paid_needs_attention"
	StatusResolved       = "resolved"
	StatusCancelled      = "cancelled"
	StatusExpired        = "expired"
)

// Terminal reports whether an order can no longer change through payment.
func Terminal(status string) bool {
	switch status {
	case StatusPaid, StatusNeedsAttention, StatusResolved, StatusCancelled:
		return true
	}
	return false
}
