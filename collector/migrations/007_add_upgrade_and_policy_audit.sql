-- Tambahan kolom inventori hardware upgrade, lisensi, keamanan endpoint, dan audit policy

ALTER TABLE asset_info
    ADD COLUMN IF NOT EXISTS device_manufacturer VARCHAR(100),
    ADD COLUMN IF NOT EXISTS device_model VARCHAR(150),
    ADD COLUMN IF NOT EXISTS bios_release_date VARCHAR(20),
    ADD COLUMN IF NOT EXISTS gpu_model VARCHAR(200),
    ADD COLUMN IF NOT EXISTS ram_speed_mhz INT,
    ADD COLUMN IF NOT EXISTS ram_max_gb NUMERIC(10,2),
    ADD COLUMN IF NOT EXISTS ram_form_factor VARCHAR(30),
    ADD COLUMN IF NOT EXISTS disk_volumes JSONB NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS licenses JSONB NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS security_status JSONB NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS running_processes JSONB NOT NULL DEFAULT '[]'::jsonb;

ALTER TABLE installed_applications
    ADD COLUMN IF NOT EXISTS install_scope VARCHAR(20) NOT NULL DEFAULT 'system';

-- Aturan kebijakan software perusahaan (forbidden / required / licensed)
CREATE TABLE IF NOT EXISTS software_policies (
    id              SERIAL PRIMARY KEY,
    pattern         VARCHAR(150) NOT NULL UNIQUE,
    policy_type     VARCHAR(20) NOT NULL, -- 'forbidden' | 'required' | 'licensed'
    category        VARCHAR(50) NOT NULL DEFAULT '',
    licensed_seats  INT NOT NULL DEFAULT 0,
    notes           VARCHAR(255) NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Seed aturan dasar bawaan (aman dijalankan berulang)
INSERT INTO software_policies (pattern, policy_type, category, licensed_seats, notes)
VALUES
    ('KMSpico', 'forbidden', 'Crack / Activator', 0, 'Activator bajakan Windows/Office'),
    ('AutoKMS', 'forbidden', 'Crack / Activator', 0, 'Activator bajakan Windows/Office'),
    ('KMSAuto', 'forbidden', 'Crack / Activator', 0, 'Activator bajakan Windows/Office'),
    ('uTorrent', 'forbidden', 'P2P / Torrent', 0, 'Aplikasi torrent berisiko malware & boros bandwidth'),
    ('BitTorrent', 'forbidden', 'P2P / Torrent', 0, 'Aplikasi torrent tidak diizinkan policy kantor'),
    ('qBittorrent', 'forbidden', 'P2P / Torrent', 0, 'Aplikasi torrent tidak diizinkan policy kantor'),
    ('TLauncher', 'forbidden', 'Game / Non-Work', 0, 'Game launcher tidak resmi'),
    ('Cheat Engine', 'forbidden', 'Crack / Utility', 0, 'Tool modifikasi memori / game')
ON CONFLICT (pattern) DO NOTHING;
