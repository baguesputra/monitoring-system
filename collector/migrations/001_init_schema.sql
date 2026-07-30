-- Tabel daftar server yang dipantau
CREATE TABLE servers (
    id              SERIAL PRIMARY KEY,
    server_id       VARCHAR(50) NOT NULL UNIQUE,   -- identitas unik dari config.yaml agent
    hostname        VARCHAR(100) NOT NULL,
    ip_address      VARCHAR(45),
    location        VARCHAR(100),                   -- lokasi fisik/gedung, opsional
    auth_token      VARCHAR(255),                    -- untuk autentikasi agent (fase lanjut)
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_active       BOOLEAN NOT NULL DEFAULT TRUE
);

-- Tabel data metrics (CPU, RAM, Disk) - time-series
CREATE TABLE metrics (
    id              BIGSERIAL PRIMARY KEY,
    server_id       VARCHAR(50) NOT NULL REFERENCES servers(server_id),
    cpu_percent     NUMERIC(5,2) NOT NULL,
    ram_percent     NUMERIC(5,2) NOT NULL,
    disk_percent    NUMERIC(5,2) NOT NULL,
    recorded_at     TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index untuk query yang sering dipakai: filter per server + urutan waktu
CREATE INDEX idx_metrics_server_recorded
    ON metrics (server_id, recorded_at DESC);