package integrations_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	appintegrations "github.com/adirkuhn/commerceos/application/integrations"
	"github.com/adirkuhn/commerceos/domain/errs"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
)

func TestNextStatusAfterTest_connectedOnNilError(t *testing.T) {
	assert.Equal(t, integrationsDomain.StatusConnected, appintegrations.NextStatusAfterTest(nil))
}

func TestNextStatusAfterTest_errorOnAnyError(t *testing.T) {
	assert.Equal(t, integrationsDomain.StatusError, appintegrations.NextStatusAfterTest(errors.New("401 unauthorized")))
}

func TestConnectionTestFailure_keepsAClassifiedWooRefusalAsTheCode(t *testing.T) {
	cause := fmt.Errorf("Get \"https://shop\": %w: status 403", integrationsDomain.ErrWooBlocked)

	err := appintegrations.ConnectionTestFailure(cause)

	code, _, ok := errs.CodeOf(err)
	assert.True(t, ok)
	assert.Equal(t, "WOO_BLOCKED", code)
	assert.True(t, errors.Is(err, integrationsDomain.ErrConnectionTestFailed), "still a connection test failure for the handler")
}

func TestConnectionTestFailure_unclassifiedCauseKeepsTheGenericCode(t *testing.T) {
	err := appintegrations.ConnectionTestFailure(errors.New("dial tcp: no such host"))

	code, _, _ := errs.CodeOf(err)
	assert.Equal(t, "CONNECTION_TEST_FAILED", code)
	detail, _ := errs.DetailOf(err)
	assert.Equal(t, "dial tcp: no such host", detail)
}
