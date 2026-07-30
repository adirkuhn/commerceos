package integrations_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	appintegrations "github.com/adirkuhn/commerceos/application/integrations"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
)

func TestNextStatusAfterTest_connectedOnNilError(t *testing.T) {
	assert.Equal(t, integrationsDomain.StatusConnected, appintegrations.NextStatusAfterTest(nil))
}

func TestNextStatusAfterTest_errorOnAnyError(t *testing.T) {
	assert.Equal(t, integrationsDomain.StatusError, appintegrations.NextStatusAfterTest(errors.New("401 unauthorized")))
}
