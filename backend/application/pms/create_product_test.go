package pms_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apppms "github.com/adirkuhn/commerceos/application/pms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

// TestCreateProduct_raisesProductChangedEvent is the actual fix for "new
// products don't sync to WooCommerce" -- CreateProductUseCase used to be
// the only product-mutating use case that never raised pms.product.changed,
// so EnrollProductInChannelsUseCase (which reacts to that event) never ran
// for a brand-new product. See application/channels' consumer wiring.
func TestCreateProduct_raisesProductChangedEvent(t *testing.T) {
	productRepo := &stubProductRepo{}
	outboxRepo := &stubOutboxRepo{}
	uc := apppms.NewCreateProductUseCase(newMockGormDB(t), productRepo, outboxRepo)

	_, err := uc.Execute(context.Background(), apppms.CreateProductInput{Name: "T-Shirt", Status: pmsDomain.ProductStatusDraft})

	require.NoError(t, err)
	assert.Equal(t, []string{"pms.product.changed"}, outboxRepo.calls)
}

func TestCreateProduct_setsTags(t *testing.T) {
	productRepo := &stubProductRepo{}
	uc := apppms.NewCreateProductUseCase(newMockGormDB(t), productRepo, &stubOutboxRepo{})

	product, err := uc.Execute(context.Background(), apppms.CreateProductInput{
		Name: "T-Shirt", Status: pmsDomain.ProductStatusDraft, Tags: []string{"summer", "sale"},
	})

	require.NoError(t, err)
	assert.Equal(t, []string{"summer", "sale"}, product.Tags)
}
