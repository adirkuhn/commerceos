package pos

import "errors"

var (
	ErrConnectionNotFound = errors.New("pos connection not found")
	// ErrInvalidLicense covers every way license verification can fail:
	// bad signature, expired, wrong status, or the payload just doesn't
	// parse -- deliberately one error, not several, since the caller (an
	// unauthenticated bootstrap endpoint) should never learn WHY a license
	// was rejected beyond "it wasn't accepted".
	ErrInvalidLicense = errors.New("invalid or inactive license")
	// ErrConnectionNotPending guards ApprovePOSConnectionUseCase -- only a
	// still-pending request can be approved (already-active or revoked
	// connections can't be re-approved).
	ErrConnectionNotPending = errors.New("pos connection is not pending approval")
	// ErrConnectionNotActive guards RevokePOSConnectionUseCase -- a
	// pending or already-revoked connection can't be revoked again.
	ErrConnectionNotActive = errors.New("pos connection is not active")
)
