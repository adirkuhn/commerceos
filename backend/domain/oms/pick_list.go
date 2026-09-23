package oms

import "sort"

// PickList is a read-only "what to fetch off the shelves" view across one or
// more sales orders -- quantities summed per SKU so a picker walks once,
// with a per-order breakdown so they can split items between orders
// afterwards. Nothing is persisted: this is the lightweight alternative to a
// PickingSession, for when the actual item-by-item scanning happens during
// packing instead.
type PickList struct {
	Items []PickListItem
}

// PickListItem is one thing to fetch. SKUID nil means the order line was
// never linked to a catalog SKU -- such lines are grouped by Name instead.
type PickListItem struct {
	SKUID         *string
	SKUCode       string
	Barcode       string
	Name          string
	TotalQuantity int64
	Orders        []PickListOrderLine
}

// PickListOrderLine is one order's share of a PickListItem.
type PickListOrderLine struct {
	SalesOrderID string
	OrderNumber  string
	LineItemID   string
	Quantity     int64
}

// BuildPickList aggregates lineItems (belonging to orders) into a PickList.
// Each line contributes its still-owed quantity (Quantity minus
// RefundedQuantity); fully refunded lines are left out. Items are sorted by
// SKU code, then name, with lines lacking a SKU code last.
func BuildPickList(orders []SalesOrder, lineItems []SalesOrderLineItem) PickList {
	numbers := make(map[string]string, len(orders))
	for _, o := range orders {
		numbers[o.ID] = o.DisplayNumber()
	}

	byKey := make(map[string]*PickListItem)
	var keys []string
	for _, li := range lineItems {
		qty := li.Quantity - li.RefundedQuantity
		if qty <= 0 {
			continue
		}
		key := pickListKey(li)
		item, ok := byKey[key]
		if !ok {
			item = &PickListItem{SKUID: li.SKUID, SKUCode: li.SKUCode, Barcode: li.Barcode, Name: li.Name}
			byKey[key] = item
			keys = append(keys, key)
		}
		item.TotalQuantity += qty
		item.Orders = append(item.Orders, PickListOrderLine{
			SalesOrderID: li.SalesOrderID, OrderNumber: numbers[li.SalesOrderID], LineItemID: li.ID, Quantity: qty,
		})
	}

	items := make([]PickListItem, 0, len(keys))
	for _, k := range keys {
		items = append(items, *byKey[k])
	}
	sort.SliceStable(items, func(i, j int) bool {
		a, b := items[i], items[j]
		if (a.SKUCode == "") != (b.SKUCode == "") {
			return b.SKUCode == ""
		}
		if a.SKUCode != b.SKUCode {
			return a.SKUCode < b.SKUCode
		}
		return a.Name < b.Name
	})
	return PickList{Items: items}
}

func pickListKey(li SalesOrderLineItem) string {
	if li.SKUID != nil {
		return "sku:" + *li.SKUID
	}
	return "name:" + li.Name
}

// MaxPickListOrders caps one pick list's selection -- matches the packing
// queue's page size, the largest batch the UI offers.
const MaxPickListOrders = 100
