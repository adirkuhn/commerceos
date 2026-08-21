-- +goose Up
-- Lets staff silence a failed sync entry that isn't worth retrying (the
-- Woo product was deleted, the mismatch is expected, etc.) from the "needs
-- attention" view without it lying about being fixed -- see
-- application/channels.ListSyncLogsNeedingAttentionUseCase's doc.
ALTER TABLE channel_sync_logs
    ADD COLUMN dismissed_at TIMESTAMPTZ NULL;

-- +goose Down
ALTER TABLE channel_sync_logs
    DROP COLUMN dismissed_at;
