package main

import schedulerapp "github.com/adirkuhn/commerceos/application/scheduler"

// missingJobRunners lists the known job keys with no runner -- the
// scheduler refuses to start with any (see main), so a job can never sit
// on the Scheduled Jobs page with nothing able to run it.
func missingJobRunners(runners map[string]schedulerapp.JobRunner, keys []string) []string {
	var missing []string
	for _, k := range keys {
		if _, ok := runners[k]; !ok {
			missing = append(missing, k)
		}
	}
	return missing
}
