-- +goose Up
-- Frontend-relative deep link a notification points at (e.g.
-- "/products/{id}") -- clicking the notification navigates there. Empty
-- means "no link, mark-as-read only" (every existing notification stays
-- exactly as it behaves today). First consumer: the Bol not-for-sale-
-- reasons check links back to the product that's not for sale.
ALTER TABLE notifications ADD COLUMN link TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE notifications DROP COLUMN link;
