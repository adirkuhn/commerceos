package theme

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/adirkuhn/commerceos/domain/locale"
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
	// Translations are the title (pages) and SEO texts in the store's other
	// languages; a blank one shows the default-language text.
	Translations map[locale.Locale]DocumentTexts
	Root         Node
}

// DocumentTexts is a document's merchant texts in one language.
type DocumentTexts struct {
	Title string
	SEO   SEO
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
	title, err := ValidPageTitle(title)
	if err != nil {
		return Document{}, err
	}
	return Document{Kind: KindPage, Key: key, Title: title, Root: EmptyRoot()}, nil
}

// ValidPageTitle trims a page title and checks it's 1-120 characters.
func ValidPageTitle(title string) (string, error) {
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

const (
	maxSEOTitleLength       = 70
	maxSEODescriptionLength = 320
)

// ValidDocumentTranslations trims the translated texts, drops languages
// with none, and checks the languages and lengths. Only pages have a title.
func ValidDocumentTranslations(kind DocumentKind, tr map[locale.Locale]DocumentTexts) (map[locale.Locale]DocumentTexts, error) {
	out := map[locale.Locale]DocumentTexts{}
	for l, texts := range tr {
		if !l.Valid() {
			return nil, fmt.Errorf("%w: language %q is not supported", ErrInvalidDocumentTranslation, l)
		}
		texts = DocumentTexts{Title: strings.TrimSpace(texts.Title), SEO: SEO{Title: strings.TrimSpace(texts.SEO.Title), Description: strings.TrimSpace(texts.SEO.Description)}}
		switch {
		case texts.Title != "" && kind != KindPage:
			return nil, fmt.Errorf("%w: only pages have a title", ErrInvalidDocumentTranslation)
		case utf8.RuneCountInString(texts.Title) > maxNameLength:
			return nil, fmt.Errorf("%w: title longer than %d characters", ErrInvalidDocumentTranslation, maxNameLength)
		case utf8.RuneCountInString(texts.SEO.Title) > maxSEOTitleLength:
			return nil, fmt.Errorf("%w: SEO title longer than %d characters", ErrInvalidDocumentTranslation, maxSEOTitleLength)
		case utf8.RuneCountInString(texts.SEO.Description) > maxSEODescriptionLength:
			return nil, fmt.Errorf("%w: SEO description longer than %d characters", ErrInvalidDocumentTranslation, maxSEODescriptionLength)
		}
		if texts != (DocumentTexts{}) {
			out[l] = texts
		}
	}
	return out, nil
}
