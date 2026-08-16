-- +goose Up

-- iam_user_groups.user_id and scanner_pairing_codes.user_id were TEXT while
-- user_permissions.user_id (same concept -- the external lm2 user identity)
-- is UUID. The split forces a cross-type cast on any query joining group
-- membership with fine-grained permissions for a user, and invites a stray
-- case/whitespace difference to silently fail a TEXT match that a UUID
-- comparison would normalize away. database/sql scans both identically
-- into a Go string, so this is transparent to application code.
--
-- One pre-existing row fails the cast: a dev-session test pairing code
-- claimed by a non-lm2 test JWT (user_id = 'test-user'), not a real
-- claimed pairing. Removed rather than worked around -- it's throwaway
-- session data, not a real user's pairing.
DELETE FROM scanner_pairing_codes WHERE user_id IS NOT NULL AND user_id !~ '^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$';

ALTER TABLE iam_user_groups ALTER COLUMN user_id TYPE UUID USING user_id::uuid;
ALTER TABLE scanner_pairing_codes ALTER COLUMN user_id TYPE UUID USING user_id::uuid;

-- +goose Down

ALTER TABLE scanner_pairing_codes ALTER COLUMN user_id TYPE TEXT;
ALTER TABLE iam_user_groups ALTER COLUMN user_id TYPE TEXT;
