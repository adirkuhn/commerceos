package publishers

import (
	"errors"
	"testing"

	"github.com/adirkuhn/commerceos/domain/webhooks"
)

func TestCheckPayloadSize_withinLimit_ok(t *testing.T) {
	if err := checkPayloadSize(100, 1024); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestCheckPayloadSize_exactlyAtLimit_ok(t *testing.T) {
	if err := checkPayloadSize(1024, 1024); err != nil {
		t.Fatalf("expected no error at exactly the limit, got %v", err)
	}
}

func TestCheckPayloadSize_overLimit_returnsErrPayloadTooLarge(t *testing.T) {
	err := checkPayloadSize(1025, 1024)
	if !errors.Is(err, webhooks.ErrPayloadTooLarge) {
		t.Fatalf("expected ErrPayloadTooLarge, got %v", err)
	}
}

// TestCheckPayloadSize_unknownMaxPayload_neverBlocks pins maxPayload <= 0
// (not yet negotiated with the server) as "don't judge" -- the real
// js.Publish call is the authority in that case, not this guard.
func TestCheckPayloadSize_unknownMaxPayload_neverBlocks(t *testing.T) {
	if err := checkPayloadSize(10_000_000, 0); err != nil {
		t.Fatalf("expected no error when maxPayload is unknown, got %v", err)
	}
}
