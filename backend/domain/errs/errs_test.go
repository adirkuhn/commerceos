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

var errUpstream = errs.NewWithCause("SAMPLE_UPSTREAM_FAILED", "upstream request failed")

func TestNewWithCause_exposesTheWrappedDetail(t *testing.T) {
	err := fmt.Errorf("testConnection: %w", fmt.Errorf("%w: 401 unauthorized", errUpstream))

	detail, ok := errs.DetailOf(err)
	assert.True(t, ok)
	assert.Equal(t, "401 unauthorized", detail)

	_, ok = errs.DetailOf(fmt.Errorf("x: %w", errSample))
	assert.False(t, ok, "only errors created with NewWithCause expose a detail")

	_, ok = errs.DetailOf(errUpstream)
	assert.False(t, ok, "nothing wrapped, no detail")
}

var (
	errParent = errs.New("SAMPLE_PARENT", "thing has history")
	errChild  = errs.NewChild(errParent, "SAMPLE_CHILD", "thing has sales history")
)

func TestNewChild_isItsParentButKeepsItsOwnCode(t *testing.T) {
	wrapped := fmt.Errorf("deleteThing: %w", errChild)

	assert.True(t, errors.Is(wrapped, errParent))
	assert.True(t, errors.Is(wrapped, errChild))
	code, message, _ := errs.CodeOf(wrapped)
	assert.Equal(t, "SAMPLE_CHILD", code)
	assert.Equal(t, "thing has sales history", message)
}

var (
	errSampleUpstream = errs.NewWithCause("SAMPLE_REMOTE_FAILED", "upstream failed")
	errSampleBlocked  = errs.NewChildWithCause(errSampleUpstream, "SAMPLE_REMOTE_BLOCKED", "upstream blocked us")
)

// Callers wrap a generic parent around an adapter error that already
// carries a more specific child ("%w: %w", parent, err): the child is what
// the reader needs.
func TestCodeOf_aChildDeeperInTheChainRefinesItsParent(t *testing.T) {
	adapterErr := fmt.Errorf("Get \"https://shop\": %w: status 403", errSampleBlocked)
	wrapped := fmt.Errorf("importThing: %w: %w", errSampleUpstream, adapterErr)

	code, message, ok := errs.CodeOf(wrapped)
	assert.True(t, ok)
	assert.Equal(t, "SAMPLE_REMOTE_BLOCKED", code)
	assert.Equal(t, "upstream blocked us", message)
	assert.True(t, errors.Is(wrapped, errSampleUpstream))

	detail, ok := errs.DetailOf(wrapped)
	assert.True(t, ok)
	assert.Equal(t, "status 403", detail)
}

func TestCodeOf_anUnrelatedCodedErrorDeeperInTheChainDoesNotWin(t *testing.T) {
	wrapped := fmt.Errorf("x: %w: %w", errSampleUpstream, errSample)

	code, _, _ := errs.CodeOf(wrapped)
	assert.Equal(t, "SAMPLE_REMOTE_FAILED", code)
}
