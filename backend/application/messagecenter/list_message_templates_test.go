package messagecenter_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	appmc "github.com/adirkuhn/commerceos/application/messagecenter"
	mcDomain "github.com/adirkuhn/commerceos/domain/messagecenter"
)

func TestListMessageTemplates_returnsEveryTemplate(t *testing.T) {
	repo := &stubMessageTemplateRepo{all: []mcDomain.MessageTemplate{{ID: "template-1"}, {ID: "template-2"}}}
	uc := appmc.NewListMessageTemplatesUseCase(newMockGormDB(t, 1), repo)

	templates, err := uc.Execute(context.Background())

	require.NoError(t, err)
	require.Len(t, templates, 2)
}
