package mcpserver

import (
	"encoding/json"
	"math"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
)

// Models guess argument names. The schemas below are explicit and lenient
// (additional properties allowed, no hard "required" list) so a near miss
// reaches our handler, which normalises aliases and returns a readable error
// instead of a validator dump.

func prop(typ, desc string) *jsonschema.Schema {
	return &jsonschema.Schema{Type: typ, Description: desc}
}

var createOrderSchema = &jsonschema.Schema{
	Type:        "object",
	Description: `Example: {"customer_name":"Tobi","items":[{"item_id":"jollof","qty":2},{"item_id":"chapman","qty":1}],"fulfilment":"delivery","zone_id":"okefia","address":"12 Gbongan Road, Oke-Fia"}`,
	Properties: map[string]*jsonschema.Schema{
		"customer_name": prop("string", "Required. The customer's name, 1 to 80 characters."),
		"items": {
			Type:        "array",
			Description: "Required. One object per menu item. Each object: item_id (the id from get_menu, for example \"jollof\") and qty (1 to 20).",
			Items: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"item_id": prop("string", "Menu item id exactly as returned by get_menu (for example \"jollof\")."),
					"qty":     prop("integer", "Quantity, 1 to 20."),
				},
			},
		},
		"fulfilment": prop("string", "Required. \"delivery\" or \"pickup\"."),
		"zone_id":    prop("string", "Delivery zone id from get_menu (for example \"okefia\"). Required for delivery."),
		"address":    prop("string", "Street address, 5 to 200 characters. Required for delivery."),
		"phone":      prop("string", "Customer phone number, optional."),
		"note":       prop("string", "Special instruction, up to 140 characters, optional."),
	},
}

var referenceSchema = &jsonschema.Schema{
	Type:        "object",
	Description: `Pass the reference exactly as create_order returned it in this conversation (shape TPK-0000-xxxx), or the order number the customer reads from their payment page (shape TPK-0000). Never a reference from memory or from an example.`,
	Properties: map[string]*jsonschema.Schema{
		"reference": prop("string", "Required. The reference from create_order in this conversation, or the customer's order number. Never guessed."),
	},
}

var emptySchema = &jsonschema.Schema{Type: "object"}

// ---- lenient decoding ----

func pick(m map[string]any, keys ...string) (any, bool) {
	for _, k := range keys {
		if v, ok := m[k]; ok && v != nil {
			return v, true
		}
	}
	// case-insensitive fallback
	for mk, v := range m {
		for _, k := range keys {
			if strings.EqualFold(mk, k) && v != nil {
				return v, true
			}
		}
	}
	return nil, false
}

func pickStr(m map[string]any, keys ...string) string {
	v, ok := pick(m, keys...)
	if !ok {
		return ""
	}
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case float64:
		return strings.TrimRight(strings.TrimRight(json.Number(formatFloat(t)).String(), "0"), ".")
	default:
		b, _ := json.Marshal(t)
		return strings.Trim(string(b), `"`)
	}
}

func formatFloat(f float64) string {
	if f == math.Trunc(f) {
		return json.Number(strings.TrimSuffix(strings.TrimSuffix(jsonNum(f), ".0"), ".")).String()
	}
	return jsonNum(f)
}

func jsonNum(f float64) string {
	b, _ := json.Marshal(f)
	return string(b)
}

func pickInt(m map[string]any, keys ...string) int {
	v, ok := pick(m, keys...)
	if !ok {
		return 0
	}
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case string:
		var n int
		for _, c := range strings.TrimSpace(t) {
			if c < '0' || c > '9' {
				break
			}
			n = n*10 + int(c-'0')
		}
		return n
	}
	return 0
}

// decodeCreateOrder normalises the many shapes a model may send.
func decodeCreateOrder(m map[string]any) CreateOrderInput {
	in := CreateOrderInput{
		CustomerName: pickStr(m, "customer_name", "customer", "name", "customerName"),
		Fulfilment:   strings.ToLower(pickStr(m, "fulfilment", "fulfillment", "delivery_type", "type", "method")),
		ZoneID:       strings.ToLower(pickStr(m, "zone_id", "zone", "delivery_zone", "zoneId", "area")),
		Address:      pickStr(m, "address", "delivery_address", "street"),
		Phone:        pickStr(m, "phone", "phone_number", "customer_phone"),
		Note:         pickStr(m, "note", "notes", "instructions"),
	}
	switch in.Fulfilment {
	case "deliver", "home", "home_delivery":
		in.Fulfilment = "delivery"
	case "pick up", "pick-up", "collect", "collection", "self":
		in.Fulfilment = "pickup"
	case "":
		if in.Address != "" || in.ZoneID != "" {
			in.Fulfilment = "delivery"
		}
	}
	rawItems, _ := pick(m, "items", "lines", "order", "order_items", "cart")
	switch items := rawItems.(type) {
	case []any:
		for _, it := range items {
			im, ok := it.(map[string]any)
			if !ok {
				if s, ok := it.(string); ok { // ["jollof", "chapman"]
					in.Items = append(in.Items, CartLineInput{ItemID: s, Qty: 1})
				}
				continue
			}
			line := CartLineInput{
				ItemID: strings.ToLower(pickStr(im, "item_id", "id", "item", "menu_item_id", "itemId", "menu_item", "name", "dish", "product")),
				Qty:    pickInt(im, "qty", "quantity", "count", "number", "amount", "portions"),
			}
			if line.Qty == 0 {
				line.Qty = 1
			}
			in.Items = append(in.Items, line)
		}
	case map[string]any: // {"jollof": 2}
		for k, v := range items {
			q, _ := v.(float64)
			if q <= 0 {
				q = 1
			}
			in.Items = append(in.Items, CartLineInput{ItemID: strings.ToLower(k), Qty: int(q)})
		}
	}
	return in
}

func decodeReference(m map[string]any) string {
	return strings.TrimSpace(pickStr(m, "reference", "ref", "paystack_reference", "order_no", "order_number", "order", "order_id"))
}
