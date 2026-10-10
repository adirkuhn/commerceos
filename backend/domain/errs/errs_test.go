package errs_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/adirkuhn/commerceos/domain/errs"
)

var errSample = errs.New("SAMPLE_THING_MISSING", "sample thing is missing")

func TestNew_isASentinelWithACode(t *testing.T) {
	wrapped := fmt.Errorf("createThing: %w", errSample)

	assert.True(t, errors.Is(wrapped, errSample))
	assert.Equal(t, "sample thing is missing", errSample.Error())

	code, message, ok := errs.CodeOf(wrapped)
	assert.True(t, ok)
	assert.Equal(t, "SAMPLE_THING_MISSING", code)
	assert.Equal(t, "sample thing is missing", message, "the sentence, without call-site prefixes")
}

func TestCodeOf_plainErrorHasNoCode(t *testing.T) {
	_, _, ok := errs.CodeOf(errors.New("boom"))
	assert.False(t, ok)
}

func TestCodes_listsEveryRegisteredCode(t *testing.T) {
	assert.Contains(t, errs.Codes(), "SAMPLE_THING_MISSING")
}

func TestNew_rejectsBadOrDuplicateCodes(t *testing.T) {
	assert.Panics(t, func() { errs.New("lower_case", "x") })
	assert.Panics(t, func() { errs.New("SAMPLE_THING_MISSING", "again") })
}
