package publishers

import "testing"

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

// Every subject a consumer can dead-letter from must have a DLQ subject,
// or the dead-letter publish fails ("no stream matches subject").
func TestDLQSubjectsMirrorEveryEventSubject(t *testing.T) {
	for _, s := range subjects {
		want := "dlq." + s
		found := false
		for _, d := range dlqSubjects {
			if d == want {
				found = true
			}
		}
		if !found {
			t.Errorf("dlqSubjects is missing %q", want)
		}
	}
}
