package assistant

import "github.com/adirkuhn/commerceos/domain/errs"

var (
	ErrRequestNotFound   = errs.New("ASSISTANT_REQUEST_NOT_FOUND", "assistant request not found")
	ErrTextRequired      = errs.New("ASSISTANT_TEXT_REQUIRED", "write or paste what you need")
	ErrTextTooLong       = errs.New("ASSISTANT_TEXT_TOO_LONG", "the text can be at most 20000 characters")
	ErrInvalidTransition = errs.New("ASSISTANT_INVALID_TRANSITION", "the request can't do that in its current state")
	ErrNotCancellable    = errs.New("ASSISTANT_NOT_CANCELLABLE", "only a request that hasn't finished can be cancelled")

	// ErrModelBusy marks a transient model failure (rate limit, quota,
	// overload, timeout): the request waits in the queue and retries.
	ErrModelBusy = errs.New("ASSISTANT_MODEL_BUSY", "the AI model is busy or out of quota")
	// ErrProviderUnsupported: the tenant's AI provider can't run the
	// assistant (no tool calling adapter for it yet).
	ErrProviderUnsupported = errs.New("ASSISTANT_PROVIDER_UNSUPPORTED", "the connected AI provider can't run the assistant yet")
)
