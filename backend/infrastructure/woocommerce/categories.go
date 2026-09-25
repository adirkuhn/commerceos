package woocommerce

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
)

type wooCategoryDTO struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Slug   string `json:"slug"`
	Parent int    `json:"parent"`
}

const categoriesPerPage = 100

func (c *wooClient) ListCategories(ctx context.Context) ([]integrationsDomain.WooCategory, error) {
	var out []integrationsDomain.WooCategory
	for page := 1; page <= maxCategoryPages; page++ {
		url := fmt.Sprintf("%s/wp-json/wc/v3/products/categories?page=%d&per_page=%d", c.storeURL, page, categoriesPerPage)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", c.basicAuthHeader())
		req.Header.Set("Accept", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		var dtos []wooCategoryDTO
		decodeErr := json.NewDecoder(resp.Body).Decode(&dtos)
		resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("list categories: %s: status %d", url, resp.StatusCode)
		}
		if decodeErr != nil {
			return nil, decodeErr
		}
		if len(dtos) == 0 {
			break
		}
		for _, d := range dtos {
			parent := ""
			if d.Parent != 0 {
				parent = strconv.Itoa(d.Parent)
			}
			out = append(out, integrationsDomain.WooCategory{
				ExternalID: strconv.Itoa(d.ID), Name: d.Name, Slug: d.Slug, ParentExternalID: parent,
			})
		}
	}
	return out, nil
}

// maxCategoryPages mirrors import_woo_products.go's maxImportPages — a
// safety cap against a runaway pagination loop, not an expected real limit
// (100 pages * 100 per page = 10,000 categories).
const maxCategoryPages = 100
