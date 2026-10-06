package events

// Scheduler actions (backend/application/scheduler). Job runs themselves
// are bookkeeping; what a job changes raises that change's own events.
var (
	SchedulerJobUpdated = defineAction("scheduler.job.updated",
		"A scheduled job's configuration changed (changes: enabled, interval_minutes). data: job_key.")
)
