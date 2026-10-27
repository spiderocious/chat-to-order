package ordering

import (
	"fmt"

	"github.com/spiderocious/chat-to-order/backend/internal/clock"
	"github.com/spiderocious/chat-to-order/backend/internal/money"
)

// DomainError is a rejection the caller can act on. Code is the stable
// identity, Message is a complete sentence a customer or the vendor can read.
type DomainError struct {
	Code      string
	Message   string
	Retryable bool
	Fields    map[string]any
}

func (e *DomainError) Error() string { return e.Code + ": " + e.Message }

func newErr(code, msg string, retryable bool, fields map[string]any) *DomainError {
	if fields == nil {
		fields = map[string]any{}
	}
	return &DomainError{Code: code, Message: msg, Retryable: retryable, Fields: fields}
}

func ErrValidation(field, msg string) *DomainError {
	return newErr("validation", msg, false, map[string]any{"field": field})
}

func ErrItemUnknown(id string) *DomainError {
	return newErr("item_unknown", fmt.Sprintf("No item with id '%s' on today's menu. Call get_menu for current ids.", id), false, map[string]any{"item_id": id})
}

func ErrItemUnavailable(name, id string, max int) *DomainError {
	var msg string
	switch max {
	case 0:
		msg = fmt.Sprintf("%s is sold out.", name)
	case 1:
		msg = fmt.Sprintf("%s has only 1 portion left.", name)
	default:
		msg = fmt.Sprintf("%s has only %d portions left.", name, max)
	}
	return newErr("item_unavailable", msg, false, map[string]any{"item_id": id, "max_qty": max})
}

func ErrItemNotInWindow(name, id, window string) *DomainError {
	return newErr("item_unavailable", fmt.Sprintf("%s is only served in the %s window.", name, window), false, map[string]any{"item_id": id, "max_qty": 0})
}

func ErrWindowClosed(st clock.WindowState) *DomainError {
	msg := "The kitchen is closed right now."
	if st.Name != "closed" && !st.OrdersOpen {
		msg = fmt.Sprintf("%s orders closed at %s.", title(st.Name), st.OrdersCloseAt)
	}
	fields := map[string]any{}
	if st.Next != nil {
		msg += fmt.Sprintf(" %s orders open at %s.", title(st.Next.Name), st.Next.OpensAt)
		fields["next_window"] = st.Next
	}
	return newErr("window_closed", msg, false, fields)
}

func ErrZoneRequired(names []string) *DomainError {
	return newErr("zone_required", "Delivery needs a zone: "+joinOr(names)+".", false, nil)
}

func ErrZoneUnknown(id string, names []string) *DomainError {
	return newErr("zone_unknown", fmt.Sprintf("No delivery zone '%s'. Choose one of: %s.", id, joinOr(names)), false, map[string]any{"zone_id": id})
}

func ErrAddressRequired() *DomainError {
	return newErr("address_required", "Delivery needs a street address.", false, nil)
}

func ErrTotalTooLarge(limit money.Kobo) *DomainError {
	return newErr("validation", fmt.Sprintf("Order total is above the %s limit for one order. Split it or call TPlus.", limit.Format()), false, map[string]any{"field": "items"})
}

func ErrUnknownReference(ref string) *DomainError {
	return newErr("unknown_reference", fmt.Sprintf("No order with reference %s. Use the reference or order number that create_order returned in this conversation; if you do not have it, ask the customer for the order number on their payment page (it looks like TPK-0000) and call again with that.", ref), false, map[string]any{"reference": ref})
}

func ErrPaystackUnavailable(action string) *DomainError {
	msg := "Paystack did not respond within 8 seconds. Nothing was charged. Try again in a minute."
	if action == "verify" {
		msg = "Paystack did not respond. Try again in a minute; if the customer paid, the payment is safe and will be picked up."
	}
	return newErr("paystack_unavailable", msg, true, nil)
}

func ErrPaystackRejected(reason string) *DomainError {
	return newErr("paystack_rejected", "Paystack rejected the transaction: "+reason+".", false, nil)
}

func ErrStoreUnavailable(action string) *DomainError {
	msg := "The ordering system is not responding. Nothing was charged. Try again in a minute."
	if action == "read" {
		msg = "The ordering system is not responding. Try again in a minute."
	}
	return newErr("store_unavailable", msg, true, nil)
}

func ErrPaymentNotConfirmed(linkExpiresWAT string) *DomainError {
	msg := "Paystack has not recorded this payment yet."
	if linkExpiresWAT != "" {
		msg += " The link is open until " + linkExpiresWAT + "."
	}
	return newErr("payment_not_confirmed", msg, true, map[string]any{"outcome": "not_confirmed"})
}

func ErrPaymentFailed(reason string) *DomainError {
	return newErr("payment_failed", "The bank declined the payment: "+humanGateway(reason)+".", false, map[string]any{"outcome": "failed"})
}

func ErrAmountMismatch(got, want money.Kobo) *DomainError {
	return newErr("amount_mismatch", fmt.Sprintf("Paystack recorded %s but the order total is %s. Flagged for TPlus; not confirmed.", got.Format(), want.Format()), false, map[string]any{"outcome": "amount_mismatch"})
}

func ErrInvalidTransition(msg string) *DomainError {
	return newErr("invalid_state_transition", msg, false, nil)
}

func ErrNotFound(what string) *DomainError {
	return newErr("not_found", what+" not found.", false, nil)
}

func title(s string) string {
	if s == "" {
		return s
	}
	return string(s[0]-32) + s[1:]
}

func joinOr(names []string) string {
	switch len(names) {
	case 0:
		return "none configured"
	case 1:
		return names[0]
	}
	out := ""
	for i, n := range names {
		switch {
		case i == 0:
			out = n
		case i == len(names)-1:
			out += " or " + n
		default:
			out += ", " + n
		}
	}
	return out
}

func humanGateway(s string) string {
	if s == "" {
		return "no reason given"
	}
	// Paystack phrases like "Declined: insufficient funds" read fine; trim the prefix.
	for _, p := range []string{"Declined: ", "Declined - ", "declined: "} {
		if len(s) > len(p) && s[:len(p)] == p {
			return s[len(p):]
		}
	}
	return s
}
