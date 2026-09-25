package woocommerce

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	wc "github.com/adirkuhn/go-woocommerce-api"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
)

type wooSettingDTO struct {
	ID    string `json:"id"`
	Value string `json:"value"`
}

// GetProductUnits reads the store's weight/dimension unit settings —
// there's no per-product unit on Woo's product resource, so this is the
// only way to know whether "0.5" means 0.5kg or 0.5lb.
func (c *wooClient) GetProductUnits(ctx context.Context) (string, string, error) {
	url := fmt.Sprintf("%s/wp-json/wc/v3/settings/products", c.storeURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Authorization", c.basicAuthHeader())
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", err
	}
	var settings []wooSettingDTO
	decodeErr := json.NewDecoder(resp.Body).Decode(&settings)
	resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", "", fmt.Errorf("get product units: %s: status %d", url, resp.StatusCode)
	}
	if decodeErr != nil {
		return "", "", decodeErr
	}

	var weightUnit, dimensionUnit string
	for _, s := range settings {
		switch s.ID {
		case "woocommerce_weight_unit":
			weightUnit = s.Value
		case "woocommerce_dimension_unit":
			dimensionUnit = s.Value
		}
	}
	return weightUnit, dimensionUnit, nil
}

// ListTaxRates fetches the store's whole tax rate table in one page --
// see WooClient.ListTaxRates's doc comment for why this isn't paged.
func (c *wooClient) ListTaxRates(ctx context.Context) ([]integrationsDomain.WooTaxRate, error) {
	rates, _, err := c.client.TaxRates.List(ctx, &wc.ListTaxRatesParams{PerPage: 100})
	if err != nil {
		return nil, err
	}
	out := make([]integrationsDomain.WooTaxRate, len(rates))
	for i, r := range rates {
		out[i] = integrationsDomain.WooTaxRate{
			ID: strconv.Itoa(r.ID), Country: r.Country, State: r.State, Rate: r.Rate, Class: r.Class,
		}
	}
	return out, nil
}
