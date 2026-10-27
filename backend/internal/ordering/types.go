package ordering

import (
	"github.com/spiderocious/chat-to-order/backend/internal/clock"
	"github.com/spiderocious/chat-to-order/backend/internal/money"
)

// Menu is what get_menu returns and what the board's stock strip reads.
type Menu struct {
	VendorName    string            `json:"vendor_name"`
	NowWAT        string            `json:"now_wat"`
	Window        string            `json:"window"`
	OrdersOpen    bool              `json:"orders_open"`
	OrdersCloseAt string            `json:"orders_close_at,omitempty"`
	NextWindow    *clock.NextWindow `json:"next_window"`
	PrepMinutes   int               `json:"prep_minutes"`
	Items         []MenuItem        `json:"items"`
	DeliveryZones []MenuZone        `json:"delivery_zones"`
	PickupAddress string            `json:"pickup_address"`
	NextStep      string            `json:"next_step"`
}

type MenuItem struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	Description  string     `json:"description,omitempty"`
	PriceKobo    money.Kobo `json:"price_kobo"`
	PriceDisplay string     `json:"price_display"`
	PortionsLeft int        `json:"portions_left"`
	Window       string     `json:"window"`
}

type MenuZone struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	FeeKobo    money.Kobo `json:"fee_kobo"`
	FeeDisplay string     `json:"fee_display"`
}

type CartLine struct {
	ItemID string `json:"item_id"`
	Qty    int    `json:"qty"`
}

type CreateOrderInput struct {
	CustomerName string
	Items        []CartLine
	Fulfilment   string
	ZoneID       string
	Address      string
	Phone        string
	Note         string
	Source       string // mcp | admin
}

type OrderLineOut struct {
	ItemID        string     `json:"item_id"`
	Name          string     `json:"name"`
	Qty           int        `json:"qty"`
	UnitPriceKobo money.Kobo `json:"unit_price_kobo"`
	UnitDisplay   string     `json:"unit_display"`
	LineKobo      money.Kobo `json:"line_kobo"`
	LineDisplay   string     `json:"line_display"`
}

type CreateOrderOutput struct {
	OrderNo            string         `json:"order_no"`
	Reference          string         `json:"reference"`
	AuthorizationURL   string         `json:"authorization_url"`
	Reused             bool           `json:"reused"`
	Lines              []OrderLineOut `json:"lines"`
	SubtotalKobo       money.Kobo     `json:"subtotal_kobo"`
	SubtotalDisplay    string         `json:"subtotal_display"`
	DeliveryFeeKobo    money.Kobo     `json:"delivery_fee_kobo"`
	DeliveryFeeDisplay string         `json:"delivery_fee_display"`
	TotalKobo          money.Kobo     `json:"total_kobo"`
	TotalDisplay       string         `json:"total_display"`
	Fulfilment         string         `json:"fulfilment"`
	Zone               string         `json:"zone,omitempty"`
	Address            string         `json:"address,omitempty"`
	Window             string         `json:"window"`
	LinkExpiresAtWAT   string         `json:"link_expires_at_wat"`
	NextStep           string         `json:"next_step"`
}

type VerifyOutput struct {
	Reference       string     `json:"reference"`
	OrderNo         string     `json:"order_no"`
	Outcome         string     `json:"outcome"`
	PaystackStatus  string     `json:"paystack_status"`
	AmountKobo      money.Kobo `json:"amount_kobo"`
	AmountDisplay   string     `json:"amount_display"`
	PaidAt          *string    `json:"paid_at"`
	GatewayResponse string     `json:"gateway_response"`
	OrderStatus     string     `json:"order_status"`
	Message         string     `json:"message,omitempty"`
	NextStep        string     `json:"next_step"`
}

type ConfirmOutput struct {
	OrderNo          string           `json:"order_no"`
	Status           string           `json:"status"`
	AlreadyConfirmed bool             `json:"already_confirmed"`
	ConfirmedBy      string           `json:"confirmed_by"`
	CustomerName     string           `json:"customer_name"`
	Lines            []ConfirmLineOut `json:"lines"`
	TotalDisplay     string           `json:"total_display"`
	Fulfilment       string           `json:"fulfilment"`
	Zone             string           `json:"zone,omitempty"`
	Address          string           `json:"address,omitempty"`
	ReadyByWAT       string           `json:"ready_by_wat,omitempty"`
	RiderPhone       string           `json:"rider_phone,omitempty"`
	AttentionReason  string           `json:"attention_reason,omitempty"`
	NextStep         string           `json:"next_step"`
}

type ConfirmLineOut struct {
	Name string `json:"name"`
	Qty  int    `json:"qty"`
}
