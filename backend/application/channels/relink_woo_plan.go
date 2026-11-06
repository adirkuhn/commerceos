package channels

import (
	"strings"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
)

// wooTypeVariable is the WooCommerce product type whose SKUs live on its
// variations -- the parent's own SKU, if any, identifies nothing here.
const wooTypeVariable = "variable"

// Why a relink left something for the user to resolve. Conflicts are
// reported, never guessed.
const (
	RelinkConflictDuplicateWooSKU        = "duplicate_woo_sku"
	RelinkConflictDuplicateSKU           = "duplicate_commerceos_sku"
	RelinkConflictExternalIDTaken        = "woo_id_linked_to_another_product"
	RelinkConflictProductLinkedElsewhere = "product_linked_to_another_woo_id"
	RelinkConflictParentSpansProducts    = "woo_parent_spans_products"
	RelinkConflictProductSpansShapes     = "product_spans_woo_shapes"
)

// relinkSKU is one sellable CommerceOS SKU. OnlySellableSKU means its
// product has this one SKU, so a standalone Woo product links at product
// level rather than to the variant.
type relinkSKU struct {
	ProductID       string
	ProductName     string
	VariantID       string
	SKUCode         string
	OnlySellableSKU bool
}

// relinkWooItem is one product or variation read from the store. ParentID
// is set on a variation only.
type relinkWooItem struct {
	ID       string
	ParentID string
	Name     string
	SKU      string
	Type     string
}

// RelinkLink is one listing row a relink would create. SKUCode is empty on
// a variable product's parent row.
type RelinkLink struct {
	ProductID   string
	ProductName string
	VariantID   *string
	ExternalID  string
	SKUCode     string
	IsVariation bool
}

// RelinkItem is one product matching nothing on the other side: a
// CommerceOS product id, or a Woo id.
type RelinkItem struct {
	ID      string
	Name    string
	SKUCode string
}

// RelinkConflict is one case left for the user: Reason is a
// RelinkConflict* constant, Detail the SKU, Woo id or product at issue.
type RelinkConflict struct {
	Reason string
	Detail string
}

// RelinkPlan is everything a relink found, before any write.
type RelinkPlan struct {
	ToLink              []RelinkLink
	AlreadyLinked       int
	MissingInWoo        []RelinkItem
	MissingInCommerceOS []RelinkItem
	Conflicts           []RelinkConflict
}

// planWooRelink matches a store's catalog to CommerceOS SKUs and works out
// which listing rows are missing. Pure: it reads nothing and writes nothing.
func planWooRelink(skus []relinkSKU, woo []relinkWooItem, existing []channelsDomain.ProductChannelListing) RelinkPlan {
	p := relinkPlanner{
		parentByID: map[string]relinkWooItem{}, matchedCodes: map[string]bool{},
		byExternal: map[string]channelsDomain.ProductChannelListing{},
		byRow:      map[string]channelsDomain.ProductChannelListing{},
		hasListing: map[string]bool{},
	}
	for _, l := range existing {
		p.byExternal[l.ExternalID] = l
		p.byRow[relinkRowKey(l.ProductID, l.VariantID)] = l
		p.hasListing[l.ProductID] = true
	}
	groups := p.matchItems(skus, woo)
	p.dropProductsInTwoGroups(groups)
	for _, g := range groups {
		p.applyGroup(g)
	}
	for _, s := range skus {
		if !p.matchedCodes[s.SKUCode] && !p.hasListing[s.ProductID] {
			p.plan.MissingInWoo = append(p.plan.MissingInWoo, RelinkItem{ID: s.ProductID, Name: s.ProductName, SKUCode: s.SKUCode})
		}
	}
	return p.plan
}

// relinkGroup is the rows that stand or fall together: a variable parent
// with its variations (parent set), or one standalone Woo product.
type relinkGroup struct {
	parent   *RelinkLink
	children []RelinkLink
	conflict *RelinkConflict
}

type relinkPlanner struct {
	plan         RelinkPlan
	parentByID   map[string]relinkWooItem
	matchedCodes map[string]bool
	byExternal   map[string]channelsDomain.ProductChannelListing
	byRow        map[string]channelsDomain.ProductChannelListing
	hasListing   map[string]bool
}

func relinkRowKey(productID string, variantID *string) string {
	if variantID == nil {
		return productID + "|"
	}
	return productID + "|" + *variantID
}

// matchItems pairs every Woo item with its CommerceOS SKU, recording
// duplicates and non-matches, and returns the matches grouped by shape in
// the order they were read.
func (p *relinkPlanner) matchItems(skus []relinkSKU, woo []relinkWooItem) []*relinkGroup {
	catalog := map[string][]relinkSKU{}
	for _, s := range skus {
		catalog[s.SKUCode] = append(catalog[s.SKUCode], s)
	}
	wooSKUCount := map[string]int{}
	for _, item := range woo {
		if item.Type == wooTypeVariable {
			p.parentByID[item.ID] = item
			continue
		}
		wooSKUCount[strings.TrimSpace(item.SKU)]++
	}

	var groups []*relinkGroup
	groupByKey := map[string]*relinkGroup{}
	reported := map[string]bool{}
	for _, item := range woo {
		if item.Type == wooTypeVariable {
			continue
		}
		code := strings.TrimSpace(item.SKU)
		if code == "" {
			p.addMissingInCommerceOS(item, code)
			continue
		}
		matches := catalog[code]
		reason := ""
		switch {
		case wooSKUCount[code] > 1:
			reason = RelinkConflictDuplicateWooSKU
		case len(matches) > 1:
			reason = RelinkConflictDuplicateSKU
		case len(matches) == 0:
			p.addMissingInCommerceOS(item, code)
			continue
		}
		p.matchedCodes[code] = true
		if reason != "" {
			if !reported[code] {
				reported[code] = true
				p.plan.Conflicts = append(p.plan.Conflicts, RelinkConflict{Reason: reason, Detail: code})
			}
			continue
		}
		key := relinkGroupKey(item)
		g := groupByKey[key]
		if g == nil {
			g = &relinkGroup{}
			groupByKey[key] = g
			groups = append(groups, g)
		}
		p.addToGroup(g, item, matches[0])
	}
	return groups
}

func relinkGroupKey(item relinkWooItem) string {
	if item.ParentID != "" {
		return "parent|" + item.ParentID
	}
	return "standalone|" + item.ID
}

func (p *relinkPlanner) addMissingInCommerceOS(item relinkWooItem, code string) {
	name := item.Name
	if item.ParentID != "" {
		name = p.parentByID[item.ParentID].Name
	}
	p.plan.MissingInCommerceOS = append(p.plan.MissingInCommerceOS, RelinkItem{ID: item.ID, Name: name, SKUCode: code})
}

func (p *relinkPlanner) addToGroup(g *relinkGroup, item relinkWooItem, sku relinkSKU) {
	if item.ParentID == "" {
		link := RelinkLink{ProductID: sku.ProductID, ProductName: sku.ProductName, ExternalID: item.ID, SKUCode: sku.SKUCode}
		if !sku.OnlySellableSKU {
			variantID := sku.VariantID
			link.VariantID = &variantID
		}
		g.children = append(g.children, link)
		return
	}
	if g.parent == nil {
		g.parent = &RelinkLink{ProductID: sku.ProductID, ProductName: sku.ProductName, ExternalID: item.ParentID}
	}
	if g.parent.ProductID != sku.ProductID && g.conflict == nil {
		g.conflict = &RelinkConflict{Reason: RelinkConflictParentSpansProducts, Detail: item.ParentID}
	}
	variantID := sku.VariantID
	g.children = append(g.children, RelinkLink{
		ProductID: sku.ProductID, ProductName: sku.ProductName, VariantID: &variantID,
		ExternalID: item.ID, SKUCode: sku.SKUCode, IsVariation: true,
	})
}

// dropProductsInTwoGroups rejects a product matched both as standalone
// products and as variations, or under two parents: a product's listings
// on one connection are one shape, never a mix (see ProductChannelListing).
func (p *relinkPlanner) dropProductsInTwoGroups(groups []*relinkGroup) {
	shapes := map[string]map[string]bool{}
	for _, g := range groups {
		for _, link := range g.links() {
			if shapes[link.ProductID] == nil {
				shapes[link.ProductID] = map[string]bool{}
			}
			shapes[link.ProductID][g.shape()] = true
		}
	}
	for _, g := range groups {
		if g.conflict != nil {
			continue
		}
		for _, link := range g.links() {
			if len(shapes[link.ProductID]) > 1 {
				g.conflict = &RelinkConflict{Reason: RelinkConflictProductSpansShapes, Detail: link.ProductName}
				break
			}
		}
	}
}

func (g *relinkGroup) shape() string {
	if g.parent != nil {
		return g.parent.ExternalID
	}
	return "standalone"
}

func (g *relinkGroup) links() []RelinkLink {
	if g.parent == nil {
		return g.children
	}
	return append([]RelinkLink{*g.parent}, g.children...)
}

// applyGroup adds a group's missing rows, or none: a variable product with
// a bad parent row must not gain orphan variation rows.
func (p *relinkPlanner) applyGroup(g *relinkGroup) {
	if g.conflict != nil {
		p.addConflictOnce(*g.conflict)
		return
	}
	var toLink []RelinkLink
	already := 0
	for _, link := range g.links() {
		if l, ok := p.byExternal[link.ExternalID]; ok {
			if relinkRowKey(l.ProductID, l.VariantID) != relinkRowKey(link.ProductID, link.VariantID) {
				p.plan.Conflicts = append(p.plan.Conflicts, RelinkConflict{Reason: RelinkConflictExternalIDTaken, Detail: link.ExternalID})
				return
			}
			already++
			continue
		}
		if _, ok := p.byRow[relinkRowKey(link.ProductID, link.VariantID)]; ok {
			p.plan.Conflicts = append(p.plan.Conflicts, RelinkConflict{Reason: RelinkConflictProductLinkedElsewhere, Detail: link.ProductName})
			return
		}
		toLink = append(toLink, link)
	}
	p.plan.ToLink = append(p.plan.ToLink, toLink...)
	p.plan.AlreadyLinked += already
}

func (p *relinkPlanner) addConflictOnce(c RelinkConflict) {
	for _, have := range p.plan.Conflicts {
		if have == c {
			return
		}
	}
	p.plan.Conflicts = append(p.plan.Conflicts, c)
}
