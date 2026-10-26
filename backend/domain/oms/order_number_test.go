package oms

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatOrderNumber(t *testing.T) {
	assert.Equal(t, "CO-1001", FormatOrderNumber(1001))
}
