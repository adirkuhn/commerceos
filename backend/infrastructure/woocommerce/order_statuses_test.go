package woocommerce

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListOrderStatuses_batchesIDsThroughWooInclude(t *testing.T) {
	var includes []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		include := r.URL.Query().Get("include")
		includes = append(includes, include)
		assert.Equal(t, "100", r.URL.Query().Get("per_page"))
		var out []map[string]any
		for _, id := range strings.Split(include, ",") {
			n, _ := strconv.Atoi(id)
			if n == 3 {
				continue // deleted in Woo
			}
			out = append(out, map[string]any{"id": n, "status": "on-hold"})
		}
		_ = json.NewEncoder(w).Encode(out)
	}))
	t.Cleanup(server.Close)
	client, err := (&Factory{}).NewClient(server.URL, "ck", "cs")
	require.NoError(t, err)

	ids := make([]string, 0, 150)
	for i := 1; i <= 150; i++ {
		ids = append(ids, strconv.Itoa(i))
	}
	got, err := client.ListOrderStatuses(context.Background(), ids)

	require.NoError(t, err)
	require.Len(t, includes, 2, "150 ids -> two calls of at most 100")
	assert.Len(t, strings.Split(includes[0], ","), 100)
	assert.Len(t, got, 149)
	assert.Equal(t, "on-hold", got["1"])
	assert.NotContains(t, got, "3")
}

func TestListOrderStatuses_noIDsMakesNoCall(t *testing.T) {
	client, err := (&Factory{}).NewClient("http://127.0.0.1:1", "ck", "cs")
	require.NoError(t, err)

	got, err := client.ListOrderStatuses(context.Background(), nil)

	require.NoError(t, err)
	assert.Empty(t, got)
}
