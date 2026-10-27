// Package paystack is the HTTP port to Paystack's transaction API. Two calls,
// typed errors, 8 second timeouts. Nothing here decides anything about an
// order; it reports what Paystack said.
package paystack

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

var (
	ErrUnavailable = errors.New("paystack: unavailable")
	ErrNotFound    = errors.New("paystack: not found")
)

// RejectedError is a 4xx with a message from Paystack.
type RejectedError struct {
	Status  int
	Message string
}

func (e *RejectedError) Error() string {
	return fmt.Sprintf("paystack: rejected (%d): %s", e.Status, e.Message)
}

type InitializeReq struct {
	Email         string         `json:"email"`
	AmountKobo    int64          `json:"amount"`           // what the customer was charged, fees included when the customer bears them
	RequestedKobo int64          `json:"requested_amount"` // what we asked for; the figure to compare against the order
	FeesKobo      int64          `json:"fees"`
	Currency      string         `json:"currency"`
	Reference     string         `json:"reference"`
	CallbackURL   string         `json:"callback_url"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

type InitializeRes struct {
	AuthorizationURL string `json:"authorization_url"`
	AccessCode       string `json:"access_code"`
	Reference        string `json:"reference"`
}

type VerifyRes struct {
	Status          string          `json:"status"` // success | abandoned | failed | pending | ongoing | reversed …
	Reference       string          `json:"reference"`
	AmountKobo      int64           `json:"amount"`           // charged to the customer; includes the fee when the customer bears it
	RequestedKobo   int64           `json:"requested_amount"` // what we asked for; compare the order against this
	FeesKobo        int64           `json:"fees"`
	Currency        string          `json:"currency"`
	GatewayResponse string          `json:"gateway_response"`
	PaidAt          *time.Time      `json:"paid_at"`
	Channel         string          `json:"channel"`
	Raw             json.RawMessage `json:"-"`
}

type Client interface {
	Initialize(ctx context.Context, req InitializeReq) (InitializeRes, error)
	Verify(ctx context.Context, reference string) (VerifyRes, error)
}

type HTTPClient struct {
	base   string
	secret string
	http   *http.Client
}

func NewHTTP(secret string) *HTTPClient {
	return &HTTPClient{
		base:   "https://api.paystack.co",
		secret: secret,
		http:   &http.Client{Timeout: 8 * time.Second},
	}
}

// WithBase is for tests against httptest servers.
func (c *HTTPClient) WithBase(base string) *HTTPClient { c.base = base; return c }

type envelope struct {
	Status  bool            `json:"status"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func (c *HTTPClient) do(ctx context.Context, method, path string, body any) (envelope, int, error) {
	var buf io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return envelope{}, 0, err
		}
		buf = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, buf)
	if err != nil {
		return envelope{}, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+c.secret)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	res, err := c.http.Do(req)
	if err != nil {
		var ne net.Error
		if errors.As(err, &ne) || errors.Is(err, context.DeadlineExceeded) {
			return envelope{}, 0, ErrUnavailable
		}
		return envelope{}, 0, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return envelope{}, res.StatusCode, ErrUnavailable
	}
	if res.StatusCode >= 500 {
		return envelope{}, res.StatusCode, ErrUnavailable
	}
	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		if res.StatusCode == 404 {
			return envelope{}, 404, ErrNotFound
		}
		return envelope{}, res.StatusCode, fmt.Errorf("%w: malformed response", ErrUnavailable)
	}
	if res.StatusCode == 404 {
		return env, 404, ErrNotFound
	}
	if res.StatusCode >= 400 || !env.Status {
		msg := env.Message
		if msg == "" {
			msg = http.StatusText(res.StatusCode)
		}
		return env, res.StatusCode, &RejectedError{Status: res.StatusCode, Message: msg}
	}
	return env, res.StatusCode, nil
}

func (c *HTTPClient) Initialize(ctx context.Context, req InitializeReq) (InitializeRes, error) {
	if req.Currency == "" {
		req.Currency = "NGN"
	}
	env, _, err := c.do(ctx, http.MethodPost, "/transaction/initialize", req)
	if err != nil {
		return InitializeRes{}, err
	}
	var out InitializeRes
	if err := json.Unmarshal(env.Data, &out); err != nil {
		return InitializeRes{}, fmt.Errorf("%w: bad initialize payload", ErrUnavailable)
	}
	return out, nil
}

func (c *HTTPClient) Verify(ctx context.Context, reference string) (VerifyRes, error) {
	env, _, err := c.do(ctx, http.MethodGet, "/transaction/verify/"+reference, nil)
	if err != nil {
		// Paystack answers 404 with status:false, "Transaction reference not found".
		var rej *RejectedError
		if errors.As(err, &rej) && rej.Status == 404 {
			return VerifyRes{}, ErrNotFound
		}
		return VerifyRes{}, err
	}
	var out VerifyRes
	if err := json.Unmarshal(env.Data, &out); err != nil {
		return VerifyRes{}, fmt.Errorf("%w: bad verify payload", ErrUnavailable)
	}
	out.Raw = env.Data
	return out, nil
}

// Outcome maps a Paystack status (and amount/currency check) to our vocabulary.
const (
	OutcomeSuccess        = "success"
	OutcomeNotConfirmed   = "not_confirmed"
	OutcomeFailed         = "failed"
	OutcomeAmountMismatch = "amount_mismatch"
)

// Settled is the amount to compare with the order total. When the merchant
// passes Paystack's fee to the customer, data.amount includes the fee and
// data.requested_amount is the price we asked for.
func (v VerifyRes) Settled() int64 {
	if v.RequestedKobo > 0 {
		return v.RequestedKobo
	}
	return v.AmountKobo
}

func Outcome(v VerifyRes, expectedKobo int64) string {
	switch v.Status {
	case "success":
		if v.Settled() != expectedKobo || (v.Currency != "" && v.Currency != "NGN") {
			return OutcomeAmountMismatch
		}
		return OutcomeSuccess
	case "failed", "reversed":
		return OutcomeFailed
	default:
		return OutcomeNotConfirmed
	}
}
