ALTER TABLE servers ADD COLUMN IF NOT EXISTS device_type VARCHAR(20) DEFAULT 'server';

CREATE TABLE IF NOT EXISTS asset_info (
    id                      SERIAL PRIMARY KEY,
    server_id               VARCHAR(50) NOT NULL UNIQUE REFERENCES servers(server_id),
    processor_model         VARCHAR(200),
    processor_cores         INT,
    processor_clock_ghz     NUMERIC(5,2),
    disk_type               VARCHAR(10),
    disk_size_gb            NUMERIC(10,2),
    disk_health_status      VARCHAR(20),
    ram_total_gb            NUMERIC(10,2),
    ram_slots_used          INT,
    ram_slots_total         INT,
    ram_ddr_type            VARCHAR(10),
    battery_health_percent  NUMERIC(5,2),
    serial_number           VARCHAR(100),
    ip_address              VARCHAR(45),
    mac_address             VARCHAR(20),
    os_version              VARCHAR(100),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS installed_applications (
    id              BIGSERIAL PRIMARY KEY,
    server_id       VARCHAR(50) NOT NULL REFERENCES servers(server_id),
    app_name        VARCHAR(255) NOT NULL,
    app_version     VARCHAR(100),
    publisher       VARCHAR(255),
    install_date    VARCHAR(20),
    reported_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_installed_apps_server ON installed_applications (server_id);
