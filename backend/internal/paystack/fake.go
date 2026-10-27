package paystack

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Fake stands in for Paystack locally and in tests. A reference is
// "abandoned" until MarkPaid is called (the /paid page does this when
// visited with fake=1). Totals ending in 99 kobo fail; totals ending in 50
// kobo report a different amount, to exercise amount_mismatch.
type Fake struct {
	mu      sync.Mutex
	base    string
	paid    map[string]time.Time
	amounts map[string]int64
	Down    bool // simulate ErrUnavailable
}

func NewFake(publicBase string) *Fake {
	return &Fake{base: strings.TrimRight(publicBase, "/"), paid: map[string]time.Time{}, amounts: map[string]int64{}}
}

func (f *Fake) Initialize(_ context.Context, req InitializeReq) (InitializeRes, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.Down {
		return InitializeRes{}, ErrUnavailable
	}
	if !strings.Contains(req.Email, "@") {
		return InitializeRes{}, &RejectedError{Status: 400, Message: "Invalid email address"}
	}
	f.amounts[req.Reference] = req.AmountKobo
	return InitializeRes{
		AuthorizationURL: fmt.Sprintf("%s/paid?ref=%s&fake=1", f.base, req.Reference),
		AccessCode:       "fake_" + req.Reference,
		Reference:        req.Reference,
	}, nil
}

// MarkPaid simulates the customer completing checkout.
func (f *Fake) MarkPaid(reference string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, known := f.amounts[reference]; !known {
		return false
	}
	if _, done := f.paid[reference]; !done {
		f.paid[reference] = time.Now()
	}
	return true
}

func (f *Fake) Verify(_ context.Context, reference string) (VerifyRes, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.Down {
		return VerifyRes{}, ErrUnavailable
	}
	amt, known := f.amounts[reference]
	if !known {
		return VerifyRes{}, ErrNotFound
	}
	v := VerifyRes{Reference: reference, AmountKobo: amt, Currency: "NGN", Channel: "card"}
	paidAt, paid := f.paid[reference]
	switch {
	case !paid:
		v.Status, v.GatewayResponse = "abandoned", "The transaction was not completed"
	case amt%100 == 99:
		v.Status, v.GatewayResponse = "failed", "Declined: insufficient funds"
	case amt%100 == 50:
		v.Status, v.GatewayResponse = "success", "Approved"
		v.AmountKobo = amt - 50 // mismatch on purpose
		v.PaidAt = &paidAt
	default:
		v.Status, v.GatewayResponse = "success", "Approved"
		v.PaidAt = &paidAt
	}
	v.Raw, _ = json.Marshal(map[string]any{"status": v.Status, "reference": reference, "amount": v.AmountKobo, "currency": "NGN", "gateway_response": v.GatewayResponse, "fake": true})
	return v, nil
}
