-- Label perangkat yang bisa diedit dari dashboard (pemilik, divisi, catatan)
ALTER TABLE servers ADD COLUMN IF NOT EXISTS owner_name VARCHAR(100);
ALTER TABLE servers ADD COLUMN IF NOT EXISTS department VARCHAR(100);
ALTER TABLE servers ADD COLUMN IF NOT EXISTS notes VARCHAR(255);

-- Akun user lokal pada perangkat beserta hak aksesnya
CREATE TABLE IF NOT EXISTS device_users (
    id            SERIAL PRIMARY KEY,
    server_id     VARCHAR(50) NOT NULL REFERENCES servers(server_id) ON DELETE CASCADE,
    username      VARCHAR(100) NOT NULL,
    full_name     VARCHAR(150),
    privilege     VARCHAR(30) NOT NULL DEFAULT 'standard_user',
    is_disabled   BOOLEAN NOT NULL DEFAULT FALSE,
    is_logged_in  BOOLEAN NOT NULL DEFAULT FALSE,
    reported_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (server_id, username)
);

CREATE INDEX IF NOT EXISTS idx_device_users_server ON device_users (server_id);

-- Cari perangkat mana saja yang punya user tertentu / hak akses admin
CREATE INDEX IF NOT EXISTS idx_device_users_username ON device_users (username);
CREATE INDEX IF NOT EXISTS idx_device_users_privilege ON device_users (privilege);
