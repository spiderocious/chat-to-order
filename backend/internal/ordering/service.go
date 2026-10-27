// Package ordering is the domain: menus, windows, carts, orders, stock and the
// money path. It returns typed results and *DomainError for rejections, and a
// plain error only when infrastructure failed. It never sees HTTP, JSON-RPC
// or Redis commands.
package ordering

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/oklog/ulid/v2"

	"github.com/spiderocious/chat-to-order/backend/internal/clock"
	"github.com/spiderocious/chat-to-order/backend/internal/money"
	"github.com/spiderocious/chat-to-order/backend/internal/paystack"
	"github.com/spiderocious/chat-to-order/backend/internal/redisx"
	"github.com/spiderocious/chat-to-order/backend/internal/store"
)

const (
	menuCacheKey = "menu:v1"
	menuTTL      = 5 * time.Second
	maxLines     = 12
	maxQty       = 20
	maxTotal     = money.Kobo(50_000_000) // ₦500,000
)

type Service struct {
	db       *store.DB
	pay      paystack.Client
	cache    redisx.Cache
	locks    redisx.Locker
	bus      redisx.Bus
	log      *slog.Logger
	baseURL  string
	emailDom string
	hooks    Hooks
}

// Hooks let the transport layer observe outcomes without the domain knowing it.
type Hooks struct {
	OnConfirmed func(by string, status string)
}

func New(db *store.DB, pay paystack.Client, cache redisx.Cache, locks redisx.Locker, bus redisx.Bus, log *slog.Logger, baseURL string) *Service {
	return &Service{db: db, pay: pay, cache: cache, locks: locks, bus: bus, log: log, baseURL: baseURL, emailDom: "iyabisi-kitchen.com"}
}

// SetEmailDomain sets the domain used for the per-order payer email Paystack
// requires. Paystack rejects reserved TLDs such as .example.
func (s *Service) SetEmailDomain(d string) {
	if d != "" {
		s.emailDom = d
	}
}

func (s *Service) SetHooks(h Hooks) { s.hooks = h }

// ---------- snapshot and menu ----------

// Snapshot is the cached read model: settings, items and zones together.
type Snapshot struct {
	Settings store.Settings `json:"settings"`
	Items    []store.Item   `json:"items"`
	Zones    []store.Zone   `json:"zones"`
	LoadedAt time.Time      `json:"loaded_at"`
	Layer    string         `json:"-"`
}

func (s *Service) Snapshot(ctx context.Context) (Snapshot, error) {
	if b, ok := s.cache.Get(ctx, menuCacheKey); ok {
		var snap Snapshot
		if json.Unmarshal(b, &snap) == nil {
			snap.Layer = s.cache.Layer()
			return snap, nil
		}
	}
	snap, err := s.loadSnapshot(ctx)
	if err != nil {
		return Snapshot{}, err
	}
	if b, err := json.Marshal(snap); err == nil {
		s.cache.Set(ctx, menuCacheKey, b, menuTTL)
	}
	snap.Layer = "db"
	return snap, nil
}

func (s *Service) loadSnapshot(ctx context.Context) (Snapshot, error) {
	settings, err := s.db.Settings(ctx)
	if err != nil {
		return Snapshot{}, err
	}
	items, err := s.db.Items(ctx)
	if err != nil {
		return Snapshot{}, err
	}
	zones, err := s.db.Zones(ctx)
	if err != nil {
		return Snapshot{}, err
	}
	return Snapshot{Settings: settings, Items: items, Zones: zones, LoadedAt: time.Now()}, nil
}

// Invalidate drops the snapshot and tells the boards.
func (s *Service) Invalidate(ctx context.Context, eventType string) {
	s.cache.Del(ctx, menuCacheKey)
	if eventType != "" {
		s.publish(ctx, redisx.Event{Type: eventType})
	}
}

func (s *Service) publish(ctx context.Context, e redisx.Event) {
	e.At = time.Now().UTC().Format(time.RFC3339)
	s.bus.Publish(ctx, e)
}

func windowDefs(st store.Settings) []clock.WindowDef {
	return []clock.WindowDef{
		{Name: "lunch", Open: clock.HHMM(st.LunchOpen), OrdersClose: clock.HHMM(st.LunchOrdersClose), Close: clock.HHMM(st.LunchClose)},
		{Name: "evening", Open: clock.HHMM(st.EveningOpen), OrdersClose: clock.HHMM(st.EveningOrdersClose), Close: clock.HHMM(st.EveningClose)},
	}
}

// WindowNow resolves the current window from a snapshot.
func WindowNow(snap Snapshot) clock.WindowState {
	return clock.Window(windowDefs(snap.Settings), clock.Now())
}

func offered(it store.Item, window string) bool {
	if !it.Available || it.Portions <= 0 {
		return false
	}
	return it.Window == "all" || it.Window == window
}

// Menu builds the get_menu response.
func (s *Service) Menu(ctx context.Context) (Menu, error) {
	snap, err := s.Snapshot(ctx)
	if err != nil {
		return Menu{}, ErrStoreUnavailable("read")
	}
	st := WindowNow(snap)
	// When closed, show the next window's items so the agent can say what is coming.
	listWindow := st.Name
	if st.Name == "closed" && st.Next != nil {
		listWindow = st.Next.Name
	}
	m := Menu{
		VendorName: snap.Settings.VendorName, NowWAT: st.NowWAT, Window: st.Name, OrdersOpen: st.OrdersOpen,
		OrdersCloseAt: st.OrdersCloseAt, NextWindow: st.Next, PrepMinutes: snap.Settings.PrepMinutes,
		Items: []MenuItem{}, DeliveryZones: []MenuZone{}, PickupAddress: snap.Settings.PickupAddress,
	}
	for _, it := range snap.Items {
		if !offered(it, listWindow) {
			continue
		}
		m.Items = append(m.Items, MenuItem{ID: it.ID, Name: it.Name, Description: it.Description, PriceKobo: it.PriceKobo,
			PriceDisplay: it.PriceKobo.Format(), PortionsLeft: it.Portions, Window: it.Window})
	}
	for _, z := range snap.Zones {
		if z.Active {
			m.DeliveryZones = append(m.DeliveryZones, MenuZone{ID: z.ID, Name: z.Name, FeeKobo: z.FeeKobo, FeeDisplay: z.FeeKobo.Format()})
		}
	}
	switch {
	case st.OrdersOpen:
		m.NextStep = "List the items with price_display and portions_left. Ask what they want, delivery or pickup (with zone and address for delivery), and a name. Then call create_order."
	case st.Name != "closed":
		m.NextStep = fmt.Sprintf("Orders for %s closed at %s. Tell the customer, and offer to take an order when the next window opens.", st.Name, st.OrdersCloseAt)
	default:
		m.NextStep = "The kitchen is closed. Tell the customer when the next window opens and what will be on it."
	}
	return m, nil
}

// ---------- create order ----------

var phoneDigits = regexp.MustCompile(`\D`)

func normalisePhone(p string) string {
	d := phoneDigits.ReplaceAllString(p, "")
	if strings.HasPrefix(d, "234") && len(d) == 13 {
		d = "0" + d[3:]
	}
	return d
}

func cartHash(in CreateOrderInput, lines []CartLine) string {
	parts := []string{strings.ToLower(strings.TrimSpace(in.CustomerName))}
	sorted := append([]CartLine(nil), lines...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ItemID < sorted[j].ItemID })
	for _, l := range sorted {
		parts = append(parts, fmt.Sprintf("%s:%d", l.ItemID, l.Qty))
	}
	parts = append(parts, in.Fulfilment, in.ZoneID, strings.ToLower(strings.Join(strings.Fields(in.Address), " ")))
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:])
}

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(s string) string {
	return strings.Trim(nonSlug.ReplaceAllString(strings.ToLower(s), "-"), "-")
}

func randHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// mergeLines collapses duplicate item ids.
func mergeLines(in []CartLine) []CartLine {
	idx := map[string]int{}
	var out []CartLine
	for _, l := range in {
		if i, ok := idx[l.ItemID]; ok {
			out[i].Qty += l.Qty
			continue
		}
		idx[l.ItemID] = len(out)
		out = append(out, l)
	}
	return out
}

func (s *Service) CreateOrder(ctx context.Context, in CreateOrderInput) (CreateOrderOutput, error) {
	// 1. validation that needs no data
	in.CustomerName = strings.TrimSpace(in.CustomerName)
	if l := len([]rune(in.CustomerName)); l < 1 || l > 80 {
		return CreateOrderOutput{}, ErrValidation("customer_name", "customer_name must be between 1 and 80 characters.")
	}
	if len(in.Items) < 1 || len(in.Items) > maxLines {
		return CreateOrderOutput{}, ErrValidation("items", fmt.Sprintf("items must have between 1 and %d lines.", maxLines))
	}
	for _, l := range in.Items {
		if l.Qty < 1 || l.Qty > maxQty {
			return CreateOrderOutput{}, ErrValidation("items", fmt.Sprintf("qty must be between 1 and %d.", maxQty))
		}
		if strings.TrimSpace(l.ItemID) == "" {
			return CreateOrderOutput{}, ErrValidation("items", "each line needs an item_id from get_menu.")
		}
	}
	if in.Fulfilment != "delivery" && in.Fulfilment != "pickup" {
		return CreateOrderOutput{}, ErrValidation("fulfilment", "fulfilment must be 'delivery' or 'pickup'.")
	}
	if len([]rune(in.Note)) > 140 {
		return CreateOrderOutput{}, ErrValidation("note", "note must be 140 characters or fewer.")
	}
	in.Address = strings.TrimSpace(in.Address)
	in.ZoneID = strings.ToLower(strings.TrimSpace(in.ZoneID))
	if in.Source == "" {
		in.Source = "mcp"
	}
	lines := mergeLines(in.Items)

	// 2. data-dependent validation against the snapshot (soft stock check)
	snap, err := s.Snapshot(ctx)
	if err != nil {
		return CreateOrderOutput{}, ErrStoreUnavailable("create")
	}
	st := WindowNow(snap)
	if !st.OrdersOpen {
		return CreateOrderOutput{}, ErrWindowClosed(st)
	}
	items := map[string]store.Item{}
	for _, it := range snap.Items {
		items[it.ID] = it
		items[strings.ToLower(it.Name)] = it
		items[slugify(it.Name)] = it
	}
	var subtotal money.Kobo
	var outLines []OrderLineOut
	var storeLines []store.Line
	for _, l := range lines {
		key := strings.ToLower(strings.TrimSpace(l.ItemID))
		it, ok := items[key]
		if !ok {
			it, ok = items[slugify(key)]
		}
		if !ok {
			// prefix match on the name: "jollof rice" → jollof
			for _, cand := range snap.Items {
				if strings.HasPrefix(strings.ToLower(cand.Name), key) || strings.HasPrefix(key, strings.ToLower(cand.Name)) {
					it, ok = cand, true
					break
				}
			}
		}
		if !ok {
			return CreateOrderOutput{}, ErrItemUnknown(l.ItemID)
		}
		if !(it.Window == "all" || it.Window == st.Name) {
			return CreateOrderOutput{}, ErrItemNotInWindow(it.Name, it.ID, it.Window)
		}
		if !it.Available || it.Portions < l.Qty {
			max := it.Portions
			if !it.Available {
				max = 0
			}
			return CreateOrderOutput{}, ErrItemUnavailable(it.Name, it.ID, max)
		}
		lineKobo := it.PriceKobo.MulQty(l.Qty)
		subtotal += lineKobo
		outLines = append(outLines, OrderLineOut{ItemID: it.ID, Name: it.Name, Qty: l.Qty, UnitPriceKobo: it.PriceKobo,
			UnitDisplay: it.PriceKobo.Format(), LineKobo: lineKobo, LineDisplay: lineKobo.Format()})
		storeLines = append(storeLines, store.Line{ItemID: it.ID, NameSnapshot: it.Name, UnitPriceKobo: it.PriceKobo, Qty: l.Qty, LineKobo: lineKobo})
	}

	var fee money.Kobo
	var zoneName string
	var zoneID *string
	var address *string
	if in.Fulfilment == "delivery" {
		var names []string
		var zone *store.Zone
		for i := range snap.Zones {
			if snap.Zones[i].Active {
				names = append(names, snap.Zones[i].Name)
				if snap.Zones[i].ID == in.ZoneID {
					zone = &snap.Zones[i]
				}
			}
		}
		if in.ZoneID == "" {
			return CreateOrderOutput{}, ErrZoneRequired(names)
		}
		if zone == nil {
			return CreateOrderOutput{}, ErrZoneUnknown(in.ZoneID, names)
		}
		if l := len([]rune(in.Address)); l < 5 || l > 200 {
			return CreateOrderOutput{}, ErrAddressRequired()
		}
		fee, zoneName = zone.FeeKobo, zone.Name
		zoneID, address = &zone.ID, &in.Address
	}
	total := subtotal + fee
	if total > maxTotal {
		return CreateOrderOutput{}, ErrTotalTooLarge(maxTotal)
	}

	// 3. idempotency: same cart, open order, reuse
	hash := cartHash(in, lines)
	release, _ := redisx.AcquireWait(ctx, s.locks, "cart:"+hash, 10*time.Second, 3*time.Second)
	if release != nil {
		defer release()
	}
	if existing, err := s.db.OpenOrderByCartHash(ctx, hash); err == nil && existing.Reference != nil && existing.ExpiresAt.After(time.Now()) {
		out := s.createOutput(existing, snap, zoneName, true)
		return out, nil
	}

	// 4. write the pending order
	now := time.Now()
	o := store.Order{
		ID: ulid.Make().String(), Status: store.StatusPending, CustomerName: in.CustomerName,
		Fulfilment: in.Fulfilment, ZoneID: zoneID, Address: address, Window: st.Name,
		SubtotalKobo: subtotal, DeliveryFeeKobo: fee, TotalKobo: total, Source: in.Source,
		ExpiresAt: now.Add(time.Duration(snap.Settings.PendingTTLMinutes) * time.Minute), Lines: storeLines,
	}
	o.CartHash = &hash
	if p := normalisePhone(in.Phone); p != "" {
		o.Phone = &p
	}
	if n := strings.TrimSpace(in.Note); n != "" {
		o.Note = &n
	}
	err = s.db.Tx(ctx, func(tx pgx.Tx) error {
		no, err := store.NextOrderNo(ctx, tx, snap.Settings.OrderPrefix)
		if err != nil {
			return err
		}
		o.OrderNo = no
		if err := store.InsertOrder(ctx, tx, o); err != nil {
			return err
		}
		return store.AddEvent(ctx, tx, o.ID, "created", in.Source, map[string]any{"total_kobo": total, "lines": len(storeLines)})
	})
	if errors.Is(err, store.ErrConflict) {
		// A racing identical cart won the unique index; return it.
		if existing, err2 := s.db.OpenOrderByCartHash(ctx, hash); err2 == nil {
			return s.createOutput(existing, snap, zoneName, true), nil
		}
	}
	if err != nil {
		s.log.Error("order.insert", "err", err)
		return CreateOrderOutput{}, ErrStoreUnavailable("create")
	}

	// 5. Paystack
	ref := fmt.Sprintf("%s-%s", o.OrderNo, randHex(2))
	init, err := s.pay.Initialize(ctx, paystack.InitializeReq{
		Email:       fmt.Sprintf("orders+%s@%s", strings.ToLower(o.OrderNo), s.emailDom),
		AmountKobo:  int64(total),
		Currency:    "NGN",
		Reference:   ref,
		CallbackURL: s.baseURL + "/paid?ref=" + ref,
		Metadata: map[string]any{
			"order_no": o.OrderNo, "customer_name": in.CustomerName,
			"custom_fields": []map[string]string{{"display_name": "Order", "variable_name": "order_no", "value": o.OrderNo}},
		},
	})
	if err != nil {
		_ = s.db.Cancel(ctx, o.ID, "init_failed")
		_ = store.AddEvent(ctx, s.db.Pool, o.ID, "init_failed", "system", map[string]any{"err": err.Error()})
		s.publish(ctx, redisx.Event{Type: "order.updated", OrderNo: o.OrderNo, Status: store.StatusCancelled})
		var rej *paystack.RejectedError
		if errors.As(err, &rej) {
			return CreateOrderOutput{}, ErrPaystackRejected(rej.Message)
		}
		return CreateOrderOutput{}, ErrPaystackUnavailable("initialize")
	}
	if err := s.db.SetReference(ctx, o.ID, ref, init.AuthorizationURL); err != nil {
		s.log.Error("order.set_reference", "err", err, "order_no", o.OrderNo)
		return CreateOrderOutput{}, ErrStoreUnavailable("create")
	}
	_ = store.AddEvent(ctx, s.db.Pool, o.ID, "link_issued", in.Source, map[string]any{"reference": ref})
	o.Reference, o.AuthorizationURL = &ref, &init.AuthorizationURL
	s.publish(ctx, redisx.Event{Type: "order.created", OrderNo: o.OrderNo, Status: o.Status})
	return s.createOutput(o, snap, zoneName, false), nil
}

func (s *Service) createOutput(o store.Order, snap Snapshot, zoneName string, reused bool) CreateOrderOutput {
	if zoneName == "" && o.ZoneID != nil {
		for _, z := range snap.Zones {
			if z.ID == *o.ZoneID {
				zoneName = z.Name
			}
		}
	}
	out := CreateOrderOutput{
		OrderNo: o.OrderNo, Reused: reused, Fulfilment: o.Fulfilment, Zone: zoneName, Window: o.Window,
		SubtotalKobo: o.SubtotalKobo, SubtotalDisplay: o.SubtotalKobo.Format(),
		DeliveryFeeKobo: o.DeliveryFeeKobo, DeliveryFeeDisplay: o.DeliveryFeeKobo.Format(),
		TotalKobo: o.TotalKobo, TotalDisplay: o.TotalKobo.Format(),
		LinkExpiresAtWAT: clock.WAT(o.ExpiresAt), Lines: []OrderLineOut{},
	}
	if o.Reference != nil {
		out.Reference = *o.Reference
	}
	if o.AuthorizationURL != nil {
		out.AuthorizationURL = *o.AuthorizationURL
	}
	if o.Address != nil {
		out.Address = *o.Address
	}
	for _, l := range o.Lines {
		out.Lines = append(out.Lines, OrderLineOut{ItemID: l.ItemID, Name: l.NameSnapshot, Qty: l.Qty, UnitPriceKobo: l.UnitPriceKobo,
			UnitDisplay: l.UnitPriceKobo.Format(), LineKobo: l.LineKobo, LineDisplay: l.LineKobo.Format()})
	}
	out.NextStep = "Repeat the lines and total_display exactly, send authorization_url, and say: tell me 'paid' when done. Then call verify_payment with the reference."
	if reused {
		out.NextStep = "This is the same open order as before; send the same link again. " + out.NextStep
	}
	return out
}

// ---------- verify ----------

func (s *Service) check(ctx context.Context, o store.Order, source string) (paystack.VerifyRes, string, error) {
	ref := *o.Reference
	start := time.Now()
	v, err := s.pay.Verify(ctx, ref)
	lat := int(time.Since(start).Milliseconds())
	pc := store.PaymentCheck{OrderID: &o.ID, Reference: ref, Source: source, LatencyMs: &lat}
	if err != nil {
		outcome := "unavailable"
		if errors.Is(err, paystack.ErrNotFound) {
			outcome = "unknown_reference"
		}
		pc.Outcome = outcome
		_ = s.db.InsertPaymentCheck(ctx, pc)
		s.log.Warn("paystack.verify", "reference", ref, "err", err, "latency_ms", lat)
		return v, outcome, err
	}
	outcome := paystack.Outcome(v, int64(o.TotalKobo))
	pc.Outcome = outcome
	pc.PaystackStatus = &v.Status
	pc.AmountKobo = &v.AmountKobo
	pc.Currency = &v.Currency
	pc.GatewayResponse = &v.GatewayResponse
	pc.Raw = v.Raw
	_ = s.db.InsertPaymentCheck(ctx, pc)
	lvl := slog.LevelInfo
	if outcome != paystack.OutcomeSuccess {
		lvl = slog.LevelWarn
	}
	s.log.Log(ctx, lvl, "paystack.verify", "reference", ref, "status", v.Status, "outcome", outcome, "latency_ms", lat)
	return v, outcome, nil
}

// orderByRef accepts a Paystack reference (IBK-0231-a1b2) or a bare order
// number (IBK-0231); models send either, in any case.
func (s *Service) orderByRef(ctx context.Context, ref string) (store.Order, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return store.Order{}, store.ErrNotFound
	}
	// Normalise case: prefix and number upper, 4-hex suffix lower.
	if i := strings.LastIndex(ref, "-"); i > 0 && len(ref)-i-1 == 4 && strings.Count(ref, "-") >= 2 {
		ref = strings.ToUpper(ref[:i]) + "-" + strings.ToLower(ref[i+1:])
	} else {
		ref = strings.ToUpper(ref)
	}
	o, err := s.db.OrderByReference(ctx, ref)
	if errors.Is(err, store.ErrNotFound) {
		o, err = s.db.OrderByNo(ctx, ref)
		if err == nil && o.Reference == nil {
			return o, store.ErrNotFound
		}
	}
	return o, err
}

func (s *Service) VerifyPayment(ctx context.Context, reference, source string) (VerifyOutput, error) {
	o, err := s.orderByRef(ctx, reference)
	if errors.Is(err, store.ErrNotFound) {
		return VerifyOutput{}, ErrUnknownReference(reference)
	}
	if err != nil {
		return VerifyOutput{}, ErrStoreUnavailable("read")
	}
	reference = *o.Reference
	v, outcome, err := s.check(ctx, o, source)
	if err != nil {
		if errors.Is(err, paystack.ErrNotFound) {
			return VerifyOutput{}, ErrUnknownReference(reference)
		}
		return VerifyOutput{}, ErrPaystackUnavailable("verify")
	}
	_ = store.AddEvent(ctx, s.db.Pool, o.ID, "verified", source, map[string]any{"outcome": outcome, "paystack_status": v.Status})

	out := VerifyOutput{
		Reference: reference, OrderNo: o.OrderNo, Outcome: outcome, PaystackStatus: v.Status,
		AmountKobo: money.Kobo(v.AmountKobo), AmountDisplay: money.Kobo(v.AmountKobo).Format(),
		GatewayResponse: v.GatewayResponse, OrderStatus: o.Status,
	}
	if v.PaidAt != nil {
		t := v.PaidAt.UTC().Format(time.RFC3339)
		out.PaidAt = &t
	}
	switch outcome {
	case paystack.OutcomeSuccess:
		if store.Terminal(o.Status) {
			out.NextStep = "This order is already confirmed. Call confirm_order to get the details to repeat to the customer."
		} else {
			out.NextStep = "Call confirm_order with this reference now."
		}
	case paystack.OutcomeNotConfirmed:
		out.Message = "Paystack has not recorded this payment yet. The link is open until " + clock.WAT(o.ExpiresAt) + "."
		out.NextStep = "Tell the customer it is not confirmed yet (never 'failed'). Ask them to finish the payment and say paid again."
	case paystack.OutcomeFailed:
		out.Message = "The bank declined the payment: " + humanGateway(v.GatewayResponse) + "."
		out.NextStep = "Say the bank's reason plainly and offer a new link by calling create_order again with the same cart."
	case paystack.OutcomeAmountMismatch:
		out.Message = fmt.Sprintf("Paystack recorded %s but the order total is %s. Flagged for Iya Bisi.", money.Kobo(v.AmountKobo).Format(), o.TotalKobo.Format())
		out.NextStep = "Do not confirm. Tell the customer the amount did not match and Iya Bisi will contact them."
	}
	return out, nil
}

// ---------- confirm ----------

// ConfirmOrder verifies again, reserves stock and marks paid, in one
// transaction. Idempotent on reference. Called by the model, the reconciler
// and (if ever added) a webhook.
func (s *Service) ConfirmOrder(ctx context.Context, reference, actor string) (ConfirmOutput, error) {
	o, err := s.orderByRef(ctx, reference)
	if errors.Is(err, store.ErrNotFound) {
		return ConfirmOutput{}, ErrUnknownReference(reference)
	}
	if err != nil {
		return ConfirmOutput{}, ErrStoreUnavailable("read")
	}
	reference = *o.Reference
	snap, err := s.Snapshot(ctx)
	if err != nil {
		return ConfirmOutput{}, ErrStoreUnavailable("read")
	}
	if store.Terminal(o.Status) {
		return s.confirmOutput(o, snap, true), nil
	}

	release, got := redisx.AcquireWait(ctx, s.locks, "confirm:"+reference, 20*time.Second, 3*time.Second)
	if got {
		defer release()
		// another caller may have finished while we waited
		if o2, err := s.db.OrderByReference(ctx, reference); err == nil && store.Terminal(o2.Status) {
			return s.confirmOutput(o2, snap, true), nil
		}
	}

	v, outcome, err := s.check(ctx, o, actor)
	if err != nil {
		if errors.Is(err, paystack.ErrNotFound) {
			return ConfirmOutput{}, ErrUnknownReference(reference)
		}
		return ConfirmOutput{}, ErrPaystackUnavailable("verify")
	}
	switch outcome {
	case paystack.OutcomeNotConfirmed:
		return ConfirmOutput{}, ErrPaymentNotConfirmed(clock.WAT(o.ExpiresAt))
	case paystack.OutcomeFailed:
		return ConfirmOutput{}, ErrPaymentFailed(v.GatewayResponse)
	case paystack.OutcomeAmountMismatch:
		_ = store.AddEvent(ctx, s.db.Pool, o.ID, "amount_mismatch", actor, map[string]any{"got": v.AmountKobo, "want": o.TotalKobo})
		return ConfirmOutput{}, ErrAmountMismatch(money.Kobo(v.AmountKobo), o.TotalKobo)
	}

	paidAt := time.Now()
	if v.PaidAt != nil {
		paidAt = *v.PaidAt
	}
	var final store.Order
	var finalStatus string
	err = s.db.Tx(ctx, func(tx pgx.Tx) error {
		locked, err := store.LockOrder(ctx, tx, o.ID)
		if err != nil {
			return err
		}
		if store.Terminal(locked.Status) {
			final = locked
			return nil
		}
		ids := make([]string, 0, len(locked.Lines))
		for _, l := range locked.Lines {
			ids = append(ids, l.ItemID)
		}
		portions, err := store.LockPortions(ctx, tx, ids)
		if err != nil {
			return err
		}
		var short []string
		for _, l := range locked.Lines {
			if portions[l.ItemID] < l.Qty {
				short = append(short, fmt.Sprintf("%s: wanted %d, only %d left", l.NameSnapshot, l.Qty, portions[l.ItemID]))
			}
		}
		if len(short) == 0 {
			for _, l := range locked.Lines {
				if err := store.DecrementPortions(ctx, tx, l.ItemID, l.Qty); err != nil {
					return err
				}
			}
			readyBy := clock.ReadyBy(paidAt, snap.Settings.PrepMinutes)
			if err := store.MarkPaid(ctx, tx, locked.ID, paidAt, readyBy, actor); err != nil {
				return err
			}
			finalStatus = store.StatusPaid
			return store.AddEvent(ctx, tx, locked.ID, "confirmed", actor, map[string]any{"ready_by": readyBy})
		}
		reason := strings.Join(short, "; ")
		if err := store.MarkNeedsAttention(ctx, tx, locked.ID, paidAt, actor, reason); err != nil {
			return err
		}
		finalStatus = store.StatusNeedsAttention
		return store.AddEvent(ctx, tx, locked.ID, "needs_attention", actor, map[string]any{"reason": reason})
	})
	if err != nil {
		s.log.Error("order.confirm", "err", err, "reference", reference)
		return ConfirmOutput{}, ErrStoreUnavailable("confirm")
	}
	if final.ID == "" {
		final, err = s.db.OrderByReference(ctx, reference)
		if err != nil {
			return ConfirmOutput{}, ErrStoreUnavailable("read")
		}
	}
	s.Invalidate(ctx, "")
	s.publish(ctx, redisx.Event{Type: "order.updated", OrderNo: final.OrderNo, Status: final.Status})
	s.publish(ctx, redisx.Event{Type: "menu.updated"})
	if s.hooks.OnConfirmed != nil && finalStatus != "" {
		s.hooks.OnConfirmed(actor, finalStatus)
	}
	return s.confirmOutput(final, snap, finalStatus == ""), nil
}

func (s *Service) confirmOutput(o store.Order, snap Snapshot, already bool) ConfirmOutput {
	out := ConfirmOutput{
		OrderNo: o.OrderNo, Status: o.Status, AlreadyConfirmed: already, CustomerName: o.CustomerName,
		TotalDisplay: o.TotalKobo.Format(), Fulfilment: o.Fulfilment, Lines: []ConfirmLineOut{},
	}
	if o.ConfirmedBy != nil {
		out.ConfirmedBy = *o.ConfirmedBy
	}
	if o.Address != nil {
		out.Address = *o.Address
	}
	if o.ZoneID != nil {
		for _, z := range snap.Zones {
			if z.ID == *o.ZoneID {
				out.Zone = z.Name
			}
		}
	}
	for _, l := range o.Lines {
		out.Lines = append(out.Lines, ConfirmLineOut{Name: l.NameSnapshot, Qty: l.Qty})
	}
	switch o.Status {
	case store.StatusPaid, store.StatusResolved:
		if o.ReadyBy != nil {
			out.ReadyByWAT = clock.WAT(*o.ReadyBy)
		}
		if o.Fulfilment == "delivery" {
			out.RiderPhone = snap.Settings.RiderPhone
			out.NextStep = fmt.Sprintf("Tell the customer: order %s confirmed, ready by about %s, the rider will call.", o.OrderNo, out.ReadyByWAT)
		} else {
			out.NextStep = fmt.Sprintf("Tell the customer: order %s confirmed, ready for pickup at %s by about %s.", o.OrderNo, snap.Settings.PickupAddress, out.ReadyByWAT)
		}
	case store.StatusNeedsAttention:
		if o.AttentionReason != nil {
			out.AttentionReason = *o.AttentionReason
		}
		out.NextStep = "Apologise: payment received but an item ran out before it was confirmed. Iya Bisi will contact them to swap or sort it out. Do not promise a refund."
	case store.StatusCancelled:
		out.NextStep = "This order was cancelled. Offer to start a new one with create_order."
	default:
		out.NextStep = "The order is not confirmed."
	}
	return out
}

// ---------- admin actions ----------

func (s *Service) ResolveOrder(ctx context.Context, orderNo, note, actor string) (store.Order, error) {
	o, err := s.db.OrderByNo(ctx, orderNo)
	if errors.Is(err, store.ErrNotFound) {
		return o, ErrNotFound("Order " + orderNo)
	}
	if err != nil {
		return o, err
	}
	if o.Status != store.StatusNeedsAttention {
		return o, ErrInvalidTransition("Only an order that needs attention can be resolved.")
	}
	if err := s.db.Resolve(ctx, o.ID, note); err != nil {
		return o, err
	}
	_ = store.AddEvent(ctx, s.db.Pool, o.ID, "resolved", actor, map[string]any{"note": note})
	s.publish(ctx, redisx.Event{Type: "order.updated", OrderNo: o.OrderNo, Status: store.StatusResolved})
	return s.db.OrderByNo(ctx, orderNo)
}

func (s *Service) CancelOrder(ctx context.Context, orderNo, reason, actor string) (store.Order, error) {
	o, err := s.db.OrderByNo(ctx, orderNo)
	if errors.Is(err, store.ErrNotFound) {
		return o, ErrNotFound("Order " + orderNo)
	}
	if err != nil {
		return o, err
	}
	if o.Status != store.StatusPending && o.Status != store.StatusExpired {
		return o, ErrInvalidTransition("Only an unpaid order can be cancelled.")
	}
	if err := s.db.Cancel(ctx, o.ID, reason); err != nil {
		return o, err
	}
	_ = store.AddEvent(ctx, s.db.Pool, o.ID, "cancelled", actor, map[string]any{"reason": reason})
	s.publish(ctx, redisx.Event{Type: "order.updated", OrderNo: o.OrderNo, Status: store.StatusCancelled})
	return s.db.OrderByNo(ctx, orderNo)
}

// Publish lets other packages announce events through the same bus.
func (s *Service) Publish(ctx context.Context, e redisx.Event) { s.publish(ctx, e) }
