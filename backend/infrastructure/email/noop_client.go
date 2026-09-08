package email

import (
	"context"
	"errors"

	emailDomain "github.com/adirkuhn/commerceos/domain/email"
)

// NoopClient is used when no SMTP relay is configured (SMTPHost unset) --
// the storefront auth routes still exist (oapi-codegen wires the whole
// ServerInterface uniformly, so individual operations can't be
// conditionally unmounted the way api/pos's routes are), but every send
// fails loudly with a clear error rather than the app silently pretending
// email works.
type NoopClient struct{}

var _ emailDomain.Client = NoopClient{}

func (NoopClient) Send(context.Context, emailDomain.Message) error {
	return errors.New("email sending is not configured (SMTP_HOST unset)")
}
