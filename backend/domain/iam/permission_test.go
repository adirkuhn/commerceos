package iam_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/adirkuhn/commerceos/domain/iam"
)

func TestRule_Validate_valid(t *testing.T) {
	cases := []string{
		"pms.product.create",
		"oms.order.read",
		"wms.warehouse.update",
		"iam.group.delete",
	}
	for _, c := range cases {
		assert.NoError(t, iam.Rule(c).Validate(), "rule: %s", c)
	}
}

func TestRule_Validate_invalid(t *testing.T) {
	cases := []string{
		"",
		"pms",
		"pms.product",
		"pms.product.create.extra",
		"pms..create",
		".product.create",
		"pms.product.",
		"   .product.create",
	}
	for _, c := range cases {
		assert.ErrorIs(t, iam.Rule(c).Validate(), iam.ErrPermissionInvalid, "rule: %q", c)
	}
}

func TestRule_String(t *testing.T) {
	r := iam.Rule("pms.product.create")
	assert.Equal(t, "pms.product.create", r.String())
}
