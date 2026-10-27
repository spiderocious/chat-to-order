package mcpserver

import (
	"encoding/json"
	"testing"
)

func decode(t *testing.T, s string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestDecodeCreateOrderAliases(t *testing.T) {
	cases := []string{
		`{"customer_name":"Tobi","items":[{"item_id":"jollof","qty":2}],"fulfilment":"delivery","zone_id":"okefia","address":"12 Gbongan Road"}`,
		`{"customer":"Tobi","items":[{"id":"jollof","quantity":2}],"fulfillment":"Delivery","zone":"okefia","delivery_address":"12 Gbongan Road"}`,
		`{"name":"Tobi","order":[{"item":"jollof","count":2}],"type":"deliver","area":"okefia","address":"12 Gbongan Road"}`,
		`{"customer_name":"Tobi","items":{"jollof":2},"address":"12 Gbongan Road","zone_id":"okefia"}`,
	}
	for _, c := range cases {
		in := decodeCreateOrder(decode(t, c))
		if in.CustomerName != "Tobi" || in.Fulfilment != "delivery" || in.ZoneID != "okefia" || in.Address != "12 Gbongan Road" {
			t.Errorf("%s → %+v", c, in)
		}
		if len(in.Items) != 1 || in.Items[0].ItemID != "jollof" || in.Items[0].Qty != 2 {
			t.Errorf("%s → items %+v", c, in.Items)
		}
	}
	in := decodeCreateOrder(decode(t, `{"customer_name":"Ada","items":["water"],"fulfilment":"pick up"}`))
	if in.Fulfilment != "pickup" || in.Items[0].Qty != 1 {
		t.Errorf("pickup/strings: %+v", in)
	}
}

func TestDecodeReference(t *testing.T) {
	for _, c := range []string{`{"reference":"TPK-0231-a1b2"}`, `{"ref":"TPK-0231-a1b2"}`, `{"order_no":"TPK-0231-a1b2"}`} {
		if got := decodeReference(decode(t, c)); got != "TPK-0231-a1b2" {
			t.Errorf("%s → %q", c, got)
		}
	}
}
