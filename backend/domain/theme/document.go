package theme

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

type DocumentKind string

const (
	// KindTemplate renders a dynamic storefront route (a product page, the
	// cart...). One per key, every theme has all of them.
	KindTemplate DocumentKind = "template"
	// KindPage is a merchant-created page served at /pages/{key}.
	KindPage DocumentKind = "page"
	// KindSectionGroup wraps every template/page (header, footer).
	KindSectionGroup DocumentKind = "section_group"
)

var templateKeys = []string{
	"home", "product", "collection", "search", "cart", "checkout",
	"order-confirmation", "login", "account", "wishlist", "not-found",
}

var sectionGroupKeys = []string{"header", "footer"}

// TemplateKeys lists every template a theme version holds.
func TemplateKeys() []string { return append([]string(nil), templateKeys...) }

// SectionGroupKeys lists every section group a theme version holds.
func SectionGroupKeys() []string { return append([]string(nil), sectionGroupKeys...) }

var pageKeyPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

const maxPageKeyLength = 64

// Document is one editable surface of a theme version.
type Document struct {
	ID        string
	TenantID  string
	VersionID string
	Kind      DocumentKind
	Key       string
	Title     string
	SEO       SEO
	Root      Node
}

type SEO struct {
	Title       string
	Description string
}

func ValidateDocumentKey(kind DocumentKind, key string) error {
	switch kind {
	case KindTemplate:
		return oneOf(key, templateKeys)
	case KindSectionGroup:
		return oneOf(key, sectionGroupKeys)
	case KindPage:
		if len(key) > maxPageKeyLength || !pageKeyPattern.MatchString(key) {
			return ErrInvalidDocumentKey
		}
		return nil
	default:
		return ErrInvalidDocumentKind
	}
}

func oneOf(key string, allowed []string) error {
	for _, k := range allowed {
		if k == key {
			return nil
		}
	}
	return ErrInvalidDocumentKey
}

// NewPage starts an empty merchant page.
func NewPage(key, title string) (Document, error) {
	if err := ValidateDocumentKey(KindPage, key); err != nil {
		return Document{}, err
	}
	title, err := validTitle(title)
	if err != nil {
		return Document{}, err
	}
	return Document{Kind: KindPage, Key: key, Title: title, Root: EmptyRoot()}, nil
}

func validTitle(title string) (string, error) {
	title = strings.TrimSpace(title)
	if title == "" || utf8.RuneCountInString(title) > maxNameLength {
		return "", ErrInvalidDocumentTitle
	}
	return title, nil
}

// DataContext is the storefront data a component reads from where it's
// placed -- e.g. Add To Cart needs "the current product".
type DataContext string

const (
	ContextNone       DataContext = ""
	ContextProduct    DataContext = "product"
	ContextCollection DataContext = "collection"
	ContextOrder      DataContext = "order"
	// ContextCart and ContextCustomer are per-shopper session data, so
	// they're available in every document.
	ContextCart     DataContext = "cart"
	ContextCustomer DataContext = "customer"
)

var routeContexts = map[string]DataContext{
	"product":            ContextProduct,
	"collection":         ContextCollection,
	"search":             ContextCollection,
	"order-confirmation": ContextOrder,
}

// ContextAvailable reports whether a component needing ctx may be placed in
// the given document.
func ContextAvailable(kind DocumentKind, key string, ctx DataContext) bool {
	switch ctx {
	case ContextNone, ContextCart, ContextCustomer:
		return true
	}
	return kind == KindTemplate && routeContexts[key] == ctx
}
