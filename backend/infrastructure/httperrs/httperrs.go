// Package httperrs are the coded errors HTTP handlers raise themselves for
// malformed requests (a missing field, a missing upload) -- coded like
// domain errors (domain/errs) so clients show them in the reader's
// language. Respond with middleware.FailErr.
package httperrs

import "github.com/adirkuhn/commerceos/domain/errs"

var (
	// ErrRequiredFieldsMissing: a required field was left empty.
	ErrRequiredFieldsMissing = errs.New("REQUIRED_FIELDS_MISSING", "required fields are missing")
	// ErrFileRequired: an upload endpoint got no file.
	ErrFileRequired = errs.New("FILE_REQUIRED", "file is required")
	// ErrFileRequiredMax10MB: an upload endpoint got no file, or one over
	// its 10 MB limit.
	ErrFileRequiredMax10MB = errs.New("FILE_REQUIRED_MAX_10MB", "file is required (max 10 MB)")
	// ErrUploadUnreadable: the upload could not be read.
	ErrUploadUnreadable = errs.New("UPLOAD_UNREADABLE", "could not read the upload")
	// ErrDateRangeIncomplete: from and to must be given together.
	ErrDateRangeIncomplete = errs.New("DATE_RANGE_INCOMPLETE", "from and to go together")
	// ErrRateLimited: too many attempts in a short time.
	ErrRateLimited = errs.New("RATE_LIMITED", "too many attempts, try again later")
)
