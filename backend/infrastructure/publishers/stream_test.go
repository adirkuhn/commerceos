package publishers

import "testing"

// TestSubjectsCoverEveryPublishedDomain guards against the exact class of
// bug that took down order-status sync, stock-to-channel sync, and the
// Woo/Bol import scheduler simultaneously on 2026-09-10: a new
// domain/events.* type started publishing under a subject prefix that was
// never added to the stream's Subjects list. js.Subscribe/js.Publish then
// fail with "no stream matches subject" for every message on that prefix —
// and in cmd/scheduler's case, a single such failure at startup calls
// log.Fatalf, crash-looping the entire process and silently starving every
// *other* consumer and the periodic sync-schedule ticker too, not just the
// one with the missing subject.
//
// This test has no way to discover new domain prefixes on its own — it can
// only check the ones listed here. Whenever domain/events gets a type whose
// EventName() starts with a new prefix, add that prefix below in the same
// change.
func TestSubjectsCoverEveryPublishedDomain(t *testing.T) {
	wantPrefixes := []string{
		"integrations.>",
		"pms.>",
		"inventory.>",
		"oms.>",
		"shipping.>",
	}

	for _, want := range wantPrefixes {
		found := false
		for _, s := range subjects {
			if s == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("subjects is missing %q — any event published under this prefix will fail with 'no stream matches subject'", want)
		}
	}
}

func TestHasAllSubjects(t *testing.T) {
	tests := []struct {
		name     string
		existing []string
		want     []string
		expect   bool
	}{
		{"all present", []string{"a.>", "b.>"}, []string{"a.>", "b.>"}, true},
		{"superset present", []string{"a.>", "b.>", "c.>"}, []string{"a.>"}, true},
		{"missing one", []string{"a.>"}, []string{"a.>", "b.>"}, false},
		{"empty existing", nil, []string{"a.>"}, false},
		{"empty want", []string{"a.>"}, nil, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := hasAllSubjects(tc.existing, tc.want)
			if got != tc.expect {
				t.Errorf("hasAllSubjects(%v, %v) = %v, want %v", tc.existing, tc.want, got, tc.expect)
			}
		})
	}
}
