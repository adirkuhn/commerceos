-- +goose Up
-- Global job: once a day, tell each tenant with dead-lettered events (a
-- consumer gave up after its last redelivery) -- escalated once the oldest
-- waited 24h. COM-14; see application/deadletters.CheckDeadLettersUseCase.
INSERT INTO scheduled_jobs (job_key, enabled, interval_minutes) VALUES
    ('check_dead_letters', true, 1440);

-- +goose Down
DELETE FROM scheduled_jobs WHERE job_key = 'check_dead_letters';
