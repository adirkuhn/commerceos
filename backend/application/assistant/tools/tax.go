package tools

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/adirkuhn/commerceos/domain/assistant"
	taxDomain "github.com/adirkuhn/commerceos/domain/tax"
)

type taxClassLister interface {
	Execute(ctx context.Context) ([]taxDomain.TaxClass, error)
}

type taxRateLister interface {
	Execute(ctx context.Context) ([]taxDomain.TaxRate, error)
}

// ListTaxRatesTool lists the tax classes with their per-country rates --
// one tool for both, since a rate means nothing without its class.
type ListTaxRatesTool struct {
	classes taxClassLister
	rates   taxRateLister
}

func NewListTaxRatesTool(classes taxClassLister, rates taxRateLister) *ListTaxRatesTool {
	return &ListTaxRatesTool{classes: classes, rates: rates}
}

func (t *ListTaxRatesTool) Definition() assistant.ToolDefinition {
	return assistant.ToolDefinition{
		Name: "list_tax_rates",
		Description: "List the tax (VAT) classes -- which one is the default -- with their rate per country. " +
			"Whether prices include tax is in get_tenant_settings.",
		Parameters: object(map[string]any{"country": str("Optional ISO country code, e.g. NL, to show only that country's rates.")}),
	}
}

type taxRateView struct {
	Country     string  `json:"country"`
	RatePercent float64 `json:"rate_percent"`
}

type taxClassView struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	IsDefault bool          `json:"is_default"`
	Rates     []taxRateView `json:"rates"`
}

func (t *ListTaxRatesTool) Call(ctx context.Context, raw json.RawMessage) (string, error) {
	var args struct {
		Country string `json:"country"`
	}
	if err := decode(raw, &args); err != nil {
		return "", err
	}
	classes, err := t.classes.Execute(ctx)
	if err != nil {
		return "", err
	}
	rates, err := t.rates.Execute(ctx)
	if err != nil {
		return "", err
	}
	country := strings.ToUpper(strings.TrimSpace(args.Country))
	byClass := map[string][]taxRateView{}
	for _, r := range rates {
		if country != "" && !strings.EqualFold(r.Country, country) {
			continue
		}
		byClass[r.TaxClassID] = append(byClass[r.TaxClassID], taxRateView{Country: r.Country, RatePercent: float64(r.RateBasisPoints) / 100})
	}
	out := struct {
		TaxClasses []taxClassView `json:"tax_classes"`
	}{TaxClasses: make([]taxClassView, 0, len(classes))}
	for _, c := range classes {
		rs := byClass[c.ID]
		if rs == nil {
			rs = []taxRateView{}
		}
		out.TaxClasses = append(out.TaxClasses, taxClassView{ID: c.ID, Name: c.Name, IsDefault: c.IsDefault, Rates: rs})
	}
	return toJSON(out)
}
