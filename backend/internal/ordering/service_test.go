package ordering

// Integration tests against a real Postgres (TEST_DATABASE_URL) and the memory
// implementations of the Redis ports. Each test gets its own schema.

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/spiderocious/chat-to-order/backend/internal/clock"
	"github.com/spiderocious/chat-to-order/backend/internal/paystack"
	"github.com/spiderocious/chat-to-order/backend/internal/redisx"
	"github.com/spiderocious/chat-to-order/backend/internal/store"
)

type harness struct {
	svc  *Service
	db   *store.DB
	fake *paystack.Fake
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	schema := fmt.Sprintf("t_%d_%d", time.Now().UnixNano()%1_000_000, os.Getpid())
	admin, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = admin.Exec(context.Background(), "DROP SCHEMA "+schema+" CASCADE")
		admin.Close()
	})
	sep := "?"
	if strings.Contains(url, "?") {
		sep = "&"
	}
	db, err := store.Open(ctx, url+sep+"search_path="+schema)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(db.Close)
	if err := db.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	fake := paystack.NewFake("http://test")
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	svc := New(db, fake, redisx.NewMemCache(), redisx.NewMemLocker(), redisx.NewMemBus(), log, "http://test")
	// fixed clock inside lunch, orders open (seed: 07:00–16:00, orders close 15:45)
	clock.Now = func() time.Time { return time.Date(2026, 10, 27, 12, 40, 0, 0, clock.Lagos) }
	return &harness{svc: svc, db: db, fake: fake}
}

func cart(name string, lines ...CartLine) CreateOrderInput {
	return CreateOrderInput{CustomerName: name, Items: lines, Fulfilment: "pickup"}
}

func TestCreateVerifyConfirm(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	out, err := h.svc.CreateOrder(ctx, CreateOrderInput{CustomerName: "Tobi", Items: []CartLine{{"jollof", 2}, {"chapman", 1}},
		Fulfilment: "delivery", ZoneID: "surulere", Address: "22 Bode Thomas"})
	if err != nil {
		t.Fatal(err)
	}
	if out.TotalDisplay != "₦8,700" || out.DeliveryFeeDisplay != "₦700" {
		t.Fatalf("totals %+v", out)
	}
	if !strings.HasPrefix(out.Reference, out.OrderNo+"-") {
		t.Fatalf("reference %s", out.Reference)
	}
	// not paid yet
	v, err := h.svc.VerifyPayment(ctx, out.Reference, "mcp")
	if err != nil || v.Outcome != paystack.OutcomeNotConfirmed {
		t.Fatalf("verify before pay: %v %+v", err, v)
	}
	if _, err := h.svc.ConfirmOrder(ctx, out.Reference, "mcp"); err == nil || !strings.Contains(err.Error(), "payment_not_confirmed") {
		t.Fatalf("confirm before pay: %v", err)
	}
	h.fake.MarkPaid(out.Reference)
	v, err = h.svc.VerifyPayment(ctx, out.Reference, "mcp")
	if err != nil || v.Outcome != paystack.OutcomeSuccess {
		t.Fatalf("verify after pay: %v %+v", err, v)
	}
	c, err := h.svc.ConfirmOrder(ctx, out.Reference, "mcp")
	if err != nil || c.Status != store.StatusPaid || c.AlreadyConfirmed {
		t.Fatalf("confirm: %v %+v", err, c)
	}
	if c.ReadyByWAT == "" {
		t.Fatal("ready_by missing")
	}
	it, _ := h.db.Item(ctx, "jollof")
	if it.Portions != 18 {
		t.Fatalf("stock %d", it.Portions)
	}
	c2, err := h.svc.ConfirmOrder(ctx, out.Reference, "mcp")
	if err != nil || !c2.AlreadyConfirmed {
		t.Fatalf("second confirm: %v %+v", err, c2)
	}
	it, _ = h.db.Item(ctx, "jollof")
	if it.Portions != 18 {
		t.Fatalf("double decrement: %d", it.Portions)
	}
	o, _ := h.db.OrderByReference(ctx, out.Reference)
	ev, _ := h.db.Events(ctx, o.ID)
	var types []string
	for _, e := range ev {
		types = append(types, e.Type)
	}
	if strings.Join(types, ",") != "created,link_issued,verified,verified,confirmed" {
		t.Fatalf("events %v", types)
	}
}

func TestIdempotentCart(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	in := cart("Ada", CartLine{"beans", 1})
	a, err := h.svc.CreateOrder(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	b, err := h.svc.CreateOrder(ctx, in)
	if err != nil || !b.Reused || b.Reference != a.Reference {
		t.Fatalf("reuse: %v %+v", err, b)
	}
	// line order should not matter
	c, _ := h.svc.CreateOrder(ctx, cart("ada ", CartLine{"beans", 1}))
	if !c.Reused {
		t.Fatal("case/space insensitive reuse")
	}
	// different cart: new order
	d, _ := h.svc.CreateOrder(ctx, cart("Ada", CartLine{"beans", 2}))
	if d.Reused || d.OrderNo == a.OrderNo {
		t.Fatal("different cart must be a new order")
	}
}

func TestValidationErrors(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	check := func(in CreateOrderInput, code string) {
		t.Helper()
		_, err := h.svc.CreateOrder(ctx, in)
		var de *DomainError
		if !errors.As(err, &de) || de.Code != code {
			t.Fatalf("want %s got %v", code, err)
		}
	}
	check(cart("", CartLine{"beans", 1}), "validation")
	check(cart("A", CartLine{"nope", 1}), "item_unknown")
	check(cart("A", CartLine{"ofada", 3}), "item_unavailable")
	check(cart("A", CartLine{"egusi", 1}), "item_unavailable") // evening only
	check(cart("A", CartLine{"beans", 0}), "validation")
	check(CreateOrderInput{CustomerName: "A", Items: []CartLine{{"beans", 1}}, Fulfilment: "delivery"}, "zone_required")
	check(CreateOrderInput{CustomerName: "A", Items: []CartLine{{"beans", 1}}, Fulfilment: "delivery", ZoneID: "mars"}, "zone_unknown")
	check(CreateOrderInput{CustomerName: "A", Items: []CartLine{{"beans", 1}}, Fulfilment: "delivery", ZoneID: "yaba"}, "address_required")
	check(CreateOrderInput{CustomerName: "A", Items: []CartLine{{"beans", 1}}, Fulfilment: "drone"}, "validation")

	clock.Now = func() time.Time { return time.Date(2026, 10, 27, 15, 50, 0, 0, clock.Lagos) } // lunch orders closed
	_, err := h.svc.CreateOrder(ctx, cart("A", CartLine{"beans", 1}))
	var de *DomainError
	if !errors.As(err, &de) || de.Code != "window_closed" || de.Fields["next_window"] == nil {
		t.Fatalf("window: %v", err)
	}
}

func TestAmountMismatchAndFailed(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	// water ₦300 ×1 = 30000 kobo; the fake needs a total ending in 50 for mismatch and 99 for failed.
	// Add an item priced to land on those endings.
	_, _ = h.db.CreateItem(ctx, store.Item{ID: "mm", Name: "Mismatch", PriceKobo: 100050, Portions: 5, Available: true, Window: "all", Sort: 9})
	_, _ = h.db.CreateItem(ctx, store.Item{ID: "ff", Name: "Fails", PriceKobo: 100099, Portions: 5, Available: true, Window: "all", Sort: 9})
	h.svc.Invalidate(ctx, "")

	m, err := h.svc.CreateOrder(ctx, cart("M", CartLine{"mm", 1}))
	if err != nil {
		t.Fatal(err)
	}
	h.fake.MarkPaid(m.Reference)
	if _, err := h.svc.ConfirmOrder(ctx, m.Reference, "mcp"); err == nil || !strings.Contains(err.Error(), "amount_mismatch") {
		t.Fatalf("mismatch: %v", err)
	}
	o, _ := h.db.OrderByReference(ctx, m.Reference)
	if o.Status != store.StatusPending {
		t.Fatalf("mismatch must not confirm: %s", o.Status)
	}
	f, _ := h.svc.CreateOrder(ctx, cart("F", CartLine{"ff", 1}))
	h.fake.MarkPaid(f.Reference)
	v, _ := h.svc.VerifyPayment(ctx, f.Reference, "mcp")
	if v.Outcome != paystack.OutcomeFailed || !strings.Contains(v.Message, "insufficient funds") {
		t.Fatalf("failed: %+v", v)
	}
}

func TestParallelConfirmLastPortions(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	// ofada has 2 portions; six customers each order 1 and pay.
	var refs []string
	for i := 0; i < 6; i++ {
		o, err := h.svc.CreateOrder(ctx, cart(fmt.Sprintf("C%d", i), CartLine{"ofada", 1}))
		if err != nil {
			t.Fatal(err)
		}
		h.fake.MarkPaid(o.Reference)
		refs = append(refs, o.Reference)
	}
	var wg sync.WaitGroup
	results := make([]string, len(refs))
	for i, r := range refs {
		wg.Add(1)
		go func(i int, r string) {
			defer wg.Done()
			out, err := h.svc.ConfirmOrder(ctx, r, "mcp")
			if err != nil {
				results[i] = "err:" + err.Error()
				return
			}
			results[i] = out.Status
		}(i, r)
	}
	wg.Wait()
	paid, attention := 0, 0
	for _, s := range results {
		switch s {
		case store.StatusPaid:
			paid++
		case store.StatusNeedsAttention:
			attention++
		default:
			t.Errorf("unexpected %s", s)
		}
	}
	if paid != 2 || attention != 4 {
		t.Fatalf("paid=%d attention=%d results=%v", paid, attention, results)
	}
	it, _ := h.db.Item(ctx, "ofada")
	if it.Portions != 0 {
		t.Fatalf("portions %d", it.Portions)
	}
}

func TestPaystackDown(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	h.fake.Down = true
	_, err := h.svc.CreateOrder(ctx, cart("D", CartLine{"beans", 1}))
	var de *DomainError
	if !errors.As(err, &de) || de.Code != "paystack_unavailable" || !de.Retryable {
		t.Fatalf("init down: %v", err)
	}
	// the order must be cancelled, not left pending
	page, _ := h.db.ListOrders(ctx, store.OrdersFilter{}, time.Time{})
	if len(page.Items) != 1 || page.Items[0].Status != store.StatusCancelled {
		t.Fatalf("orders after failed init: %+v", page.Items)
	}
}

func TestAdminTransitions(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	o, _ := h.svc.CreateOrder(ctx, cart("R", CartLine{"beans", 1}))
	if _, err := h.svc.ResolveOrder(ctx, o.OrderNo, "x", "admin:a"); err == nil {
		t.Fatal("resolve pending must fail")
	}
	if _, err := h.svc.CancelOrder(ctx, o.OrderNo, "changed mind", "admin:a"); err != nil {
		t.Fatal(err)
	}
	if _, err := h.svc.CancelOrder(ctx, o.OrderNo, "again", "admin:a"); err == nil {
		t.Fatal("cancel twice must fail")
	}
	if _, err := h.svc.ResolveOrder(ctx, "IBK-9999", "x", "admin:a"); err == nil || !strings.Contains(err.Error(), "not_found") {
		t.Fatalf("missing: %v", err)
	}
}
