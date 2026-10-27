package paystack

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func server(t *testing.T, status int, body string, delay time.Duration) *HTTPClient {
	t.Helper()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer sk_test_x" {
			t.Errorf("missing bearer")
		}
		time.Sleep(delay)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(s.Close)
	c := NewHTTP("sk_test_x").WithBase(s.URL)
	c.http.Timeout = 200 * time.Millisecond
	return c
}

func TestVerifyStatuses(t *testing.T) {
	for _, st := range []string{"success", "abandoned", "pending", "failed", "reversed", "ongoing"} {
		c := server(t, 200, `{"status":true,"data":{"status":"`+st+`","reference":"R","amount":870000,"currency":"NGN","gateway_response":"x"}}`, 0)
		v, err := c.Verify(context.Background(), "R")
		if err != nil || v.Status != st {
			t.Fatalf("%s: %v %+v", st, err, v)
		}
	}
}

func TestOutcomeMapping(t *testing.T) {
	cases := []struct {
		st   string
		amt  int64
		cur  string
		want string
	}{
		{"success", 870000, "NGN", OutcomeSuccess},
		{"success", 893402, "NGN", OutcomeAmountMismatch}, // without requested_amount, a fee-inclusive charge mismatches
		{"success", 800000, "NGN", OutcomeAmountMismatch},
		{"success", 870000, "USD", OutcomeAmountMismatch},
		{"abandoned", 870000, "NGN", OutcomeNotConfirmed},
		{"pending", 870000, "NGN", OutcomeNotConfirmed},
		{"queued", 870000, "NGN", OutcomeNotConfirmed},
		{"failed", 870000, "NGN", OutcomeFailed},
		{"reversed", 870000, "NGN", OutcomeFailed},
	}
	for _, c := range cases {
		if got := Outcome(VerifyRes{Status: c.st, AmountKobo: c.amt, Currency: c.cur}, 870000); got != c.want {
			t.Errorf("%+v: got %s", c, got)
		}
	}
}

func TestFeeBearerCustomer(t *testing.T) {
	// Customer bears the fee: amount includes it, requested_amount is ours.
	v := VerifyRes{Status: "success", AmountKobo: 893402, RequestedKobo: 870000, FeesKobo: 23402, Currency: "NGN"}
	if Outcome(v, 870000) != OutcomeSuccess {
		t.Fatal("requested_amount must win")
	}
}

func TestErrors(t *testing.T) {
	c := server(t, 404, `{"status":false,"message":"Transaction reference not found"}`, 0)
	if _, err := c.Verify(context.Background(), "R"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("404: %v", err)
	}
	c = server(t, 500, `boom`, 0)
	if _, err := c.Verify(context.Background(), "R"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("500: %v", err)
	}
	c = server(t, 200, `{"status":true,"data":{}`, 0)
	if _, err := c.Verify(context.Background(), "R"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("malformed: %v", err)
	}
	c = server(t, 200, `{}`, 500*time.Millisecond)
	if _, err := c.Verify(context.Background(), "R"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("timeout: %v", err)
	}
	c = server(t, 400, `{"status":false,"message":"Invalid email address"}`, 0)
	_, err := c.Initialize(context.Background(), InitializeReq{Email: "x", AmountKobo: 1, Reference: "R"})
	var rej *RejectedError
	if !errors.As(err, &rej) || rej.Message != "Invalid email address" {
		t.Fatalf("rejected: %v", err)
	}
}

func TestFake(t *testing.T) {
	f := NewFake("http://x")
	ctx := context.Background()
	if _, err := f.Verify(ctx, "nope"); !errors.Is(err, ErrNotFound) {
		t.Fatal("unknown ref")
	}
	_, _ = f.Initialize(ctx, InitializeReq{Email: "a@b", AmountKobo: 870000, Reference: "R1"})
	v, _ := f.Verify(ctx, "R1")
	if v.Status != "abandoned" {
		t.Fatal("before pay")
	}
	f.MarkPaid("R1")
	v, _ = f.Verify(ctx, "R1")
	if v.Status != "success" || v.AmountKobo != 870000 {
		t.Fatal("after pay")
	}
	_, _ = f.Initialize(ctx, InitializeReq{Email: "a@b", AmountKobo: 870099, Reference: "R2"})
	f.MarkPaid("R2")
	if v, _ := f.Verify(ctx, "R2"); v.Status != "failed" {
		t.Fatal("99 should fail")
	}
}
