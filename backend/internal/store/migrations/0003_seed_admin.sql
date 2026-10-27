-- +goose Up
-- The password hash is set at boot from ADMIN_PASSWORD; it never lives in the repo.
INSERT INTO admins (id, email, name) VALUES ('adm_01', 'devferanmi@gmail.com', 'Iya Bisi')
ON CONFLICT (email) DO NOTHING;

-- +goose Down
DELETE FROM admins WHERE id = 'adm_01';
