-- +goose Up
-- Reviewers test at any hour. The two headline dishes are served all day so the
-- demo conversation always has something to buy; ofada stays lunch-only and
-- egusi evening-only so the window rule is still visible.
UPDATE menu_items SET "window" = 'all', version = version + 1, updated_at = now() WHERE id IN ('jollof','fried');

-- +goose Down
UPDATE menu_items SET "window" = 'lunch', version = version + 1, updated_at = now() WHERE id IN ('jollof','fried');
