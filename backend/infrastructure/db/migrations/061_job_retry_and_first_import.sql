-- +goose Up
-- retry_count backs a bounded requeue-on-restart: a job left "running"
-- when the scheduler process died (crash, redeploy) gets reset to
-- pending and retried automatically instead of just being marked failed
-- and requiring a manual retrigger, up to a small cap so a job that
-- crashes the scheduler because of its own bug can't loop forever.
ALTER TABLE import_jobs ADD COLUMN retry_count INT NOT NULL DEFAULT 0;
ALTER TABLE document_parsing_jobs ADD COLUMN retry_count INT NOT NULL DEFAULT 0;
ALTER TABLE receiving_parsing_jobs ADD COLUMN retry_count INT NOT NULL DEFAULT 0;

-- is_first_import is decided once at job-creation time (before any attempt
-- runs) and stuck to across retries -- previously this was re-derived each
-- run from "does this connection have zero imported products yet", which
-- breaks the moment a crash-and-requeue happens mid-first-import: some
-- products already exist by the retry, so the dynamic check would wrongly
-- say "not first" and skip the orders bootstrap.
ALTER TABLE import_jobs ADD COLUMN is_first_import BOOLEAN NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE import_jobs DROP COLUMN is_first_import;
ALTER TABLE receiving_parsing_jobs DROP COLUMN retry_count;
ALTER TABLE document_parsing_jobs DROP COLUMN retry_count;
ALTER TABLE import_jobs DROP COLUMN retry_count;
