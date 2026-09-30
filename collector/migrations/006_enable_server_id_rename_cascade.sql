-- Rename server_id harus mentebar ke seluruh tabel anak tanpa kehilangan histori.
-- PostgreSQL butuh ON UPDATE CASCADE; FK bawaan (NO ACTION) akan menolak UPDATE.
-- ON DELETE CASCADE dipertahankan hanya di device_users (ikut terhapus saat device dihapus),
-- mengikuti definisi aslinya.

ALTER TABLE metrics DROP CONSTRAINT IF EXISTS metrics_server_id_fkey;
ALTER TABLE metrics ADD CONSTRAINT metrics_server_id_fkey
    FOREIGN KEY (server_id) REFERENCES servers(server_id) ON UPDATE CASCADE;

ALTER TABLE service_status DROP CONSTRAINT IF EXISTS service_status_server_id_fkey;
ALTER TABLE service_status ADD CONSTRAINT service_status_server_id_fkey
    FOREIGN KEY (server_id) REFERENCES servers(server_id) ON UPDATE CASCADE;

ALTER TABLE asset_info DROP CONSTRAINT IF EXISTS asset_info_server_id_fkey;
ALTER TABLE asset_info ADD CONSTRAINT asset_info_server_id_fkey
    FOREIGN KEY (server_id) REFERENCES servers(server_id) ON UPDATE CASCADE;

ALTER TABLE installed_applications DROP CONSTRAINT IF EXISTS installed_applications_server_id_fkey;
ALTER TABLE installed_applications ADD CONSTRAINT installed_applications_server_id_fkey
    FOREIGN KEY (server_id) REFERENCES servers(server_id) ON UPDATE CASCADE;

ALTER TABLE device_users DROP CONSTRAINT IF EXISTS device_users_server_id_fkey;
ALTER TABLE device_users ADD CONSTRAINT device_users_server_id_fkey
    FOREIGN KEY (server_id) REFERENCES servers(server_id) ON UPDATE CASCADE ON DELETE CASCADE;
