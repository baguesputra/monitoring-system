-- Tambah kolom network usage ke tabel metrics
ALTER TABLE metrics
    ADD COLUMN network_sent_bps NUMERIC(12,2) DEFAULT 0,
    ADD COLUMN network_recv_bps NUMERIC(12,2) DEFAULT 0;

-- Tabel baru khusus untuk status service (karena sifatnya beda dari metrics angka biasa,
-- 1 baris = 1 service per waktu pengecekan, bukan 1 baris per semua metrics)
CREATE TABLE service_status (
    id              BIGSERIAL PRIMARY KEY,
    server_id       VARCHAR(50) NOT NULL REFERENCES servers(server_id),
    service_name    VARCHAR(100) NOT NULL,
    is_running      BOOLEAN NOT NULL,
    checked_at      TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_service_status_server_checked
    ON service_status (server_id, checked_at DESC);