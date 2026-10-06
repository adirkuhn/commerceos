package eventrecord_test

// pendingEvents: files under application/ that wrote without raising an
// event when the every-action-raises-an-event rule (docs/events.md) was
// adopted, 2026-10-06. Burned down module by module; only ever shrinks --
// TestEveryWriteRaisesAnEvent fails when a file here is fixed but not
// removed, or a new file writes without an event.
var pendingEvents = []string{
}
