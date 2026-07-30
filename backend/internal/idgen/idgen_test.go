package idgen_test

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/adirkuhn/commerceos/internal/idgen"
)

var uuidV4Pattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func TestNewV4_matchesUUIDv4Format(t *testing.T) {
	id := idgen.NewV4()
	assert.Regexp(t, uuidV4Pattern, id)
}

func TestNewV4_producesUniqueValues(t *testing.T) {
	a := idgen.NewV4()
	b := idgen.NewV4()
	assert.NotEqual(t, a, b)
}
