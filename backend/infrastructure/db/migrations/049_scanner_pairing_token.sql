-- +goose Up
-- High-entropy opaque token alongside the existing 6-digit code. The digit
-- code is guessable within its keyspace and is now rate-limited at the
-- claim endpoint; the token (128-bit random, base64url) is unguessable
-- within the pairing TTL and is what QR codes / barcodes encode -- no rate
-- limit dependency on that path.
ALTER TABLE scanner_pairing_codes ADD COLUMN token TEXT NOT NULL DEFAULT '';

CREATE UNIQUE INDEX scanner_pairing_codes_pending_token_idx ON scanner_pairing_codes (token)
    WHERE status = 'pending' AND token <> '';

-- +goose Down
DROP INDEX scanner_pairing_codes_pending_token_idx;
ALTER TABLE scanner_pairing_codes DROP COLUMN token;
