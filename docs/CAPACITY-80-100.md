# Panduan Kapasitas 80–100 Device

> Collector (Go) + PostgreSQL 16 di 1 VM Ubuntu. Target: laptop/PC Windows ×80–100, Agent interval detik, Asset inventory 24 jam.

## 1. Hitungan Beban

| Parameter | Nilai |
|---|---|
| Device | 80–100 |
| Payload metrics | ~400 byte JSON + header ~1 KB |
| Interval default | 5s (`agent/config.yaml: interval_seconds`) |
| Req/s metrics | `100 / 5 = 20 req/s` (80 → 16 req/s) |
| Req/s service_status | sama, 1–3 insert tambahan per req jika `services_to_check` diisi |
| Asset-info | 1 req / device / 24 jam ≈ 0.001 req/s → abaikan |
| Dashboard GET | polling ringan, <1 req/s |

**Collector Go mampu >500 req/s di 1 vCPU** — bukan bottleneck.

## 2. Estimasi Storage PostgreSQL

1 row `metrics` ≈ 80–120 byte + index ≈ 60 byte.

| Interval | Row/device/hari | Row/hari (100 dev) | Row/30 hari | Size tabel+index 30 hari* |
|---|---|---|---|---|
| 5s | 17.280 | 1.728.000 | 51,8jt | 8–12 GB |
| **15s (rekomendasi 80+ dev)** | 5.760 | 576.000 | 17,3jt | 2,5–4 GB |
| 30s | 2.880 | 288.000 | 8,6jt | 1,2–2 GB |

*belum termasuk `service_status` (~sama, tapi row lebih kecil) & `installed_applications` (snapshot, kecil).

**Kesimpulan:** 5s untuk 100 device = 52jt row/bulan → query `GET /api/servers/{id}/metrics?range=24h` scan berat & vacuum lambat. **Naikkan ke 15s** potong 3× write/storage tanpa kehilangan visibilitas operasional.

## 3. Spesifikasi Server Rekomendasi

| Skala | vCPU | RAM | Disk | Catatan |
|---|---|---|---|---|
| 80 device @15s | 2 vCPU | 4 GB | 40 GB SSD (20 GB free untuk DB) | Cukup, `shared_buffers=256MB` |
| 100 device @15s | 2–4 vCPU | 4–8 GB | 60 GB SSD | 4 vCPU jika dashboard banyak user |
| 100 device @5s | 4 vCPU | 8 GB | 80–120 GB SSD | Tidak direkomendasikan |

Postgres adalah bottleneck, bukan Collector.

## 4. Tuning Wajib Sebelum Rollout 80–100

### 4.1 Interval Agent

```yaml
# agent/config.example.yaml
interval_seconds: 15  # 5 untuk <30 device, 15 untuk 80–100, 30 jika disk tipis
```

Semua device baca dari file yang sama — deploy massal via GPO/script cukup ganti 1 nilai.

### 4.2 docker-compose.yml

Sudah ditambah di repo:

- `postgres.healthcheck` (`pg_isready`) + `collector.depends_on: condition: service_healthy` — cegah race collector start duluan (sebelumnya retry 10× di `collector/db.go`, sekarang + healthcheck).
- `collector` expose `${COLLECTOR_PORT:-8081}:8081`, `postgres` `5433:5432` (hindari konflik `5432` host yang dipakai `invito-postgres`).
- `DB_HOST=postgres` override (Docker network), host lain dari `.env`.

Tidak perlu `collector/.env` terpisah — root `.env` via `env_file: .env`.

### 4.3 Koneksi DB

`collector/db.go` sekarang baca `DB_MAX_CONNS` (default `25` untuk 80–100). Set di `.env`:

```
DB_MAX_CONNS=25
```

Rumus: `req/s × avg query 10ms × 2 = ~20`. 25 aman untuk 20 req/s. Jangan >50 tanpa naikkan `postgres max_connections`.

### 4.4 Retensi Data (Penting!)

Tanpa retensi, DB tumbuh selamanya. Jalankan cron harian di host Ubuntu:

```sql
-- hapus detail >30 hari (jalan tiap malam 02:00)
DELETE FROM metrics WHERE recorded_at < NOW() - INTERVAL '30 days';
DELETE FROM service_status WHERE checked_at < NOW() - INTERVAL '30 days';
VACUUM ANALYZE metrics;
VACUUM ANALYZE service_status;
```

Untuk 100 device @15s, ini jaga DB tetap <4 GB. Tahap lanjut: agregasi per-jam untuk histori 1 tahun (PRD 7), tidak blocking untuk MVP.

Tambahkan ke crontab:

```bash
0 2 * * * docker exec monitoring-postgres psql -U monitoring_user -d monitoring_db -c "DELETE FROM metrics WHERE recorded_at < NOW() - INTERVAL '30 days'; DELETE FROM service_status WHERE checked_at < NOW() - INTERVAL '30 days';"
```

### 4.5 Index Sudah Ada

`idx_metrics_server_recorded (server_id, recorded_at DESC)` dan `idx_service_status_server_checked` sudah dari `001` & `002` — cukup untuk `GET /metrics?range=`.

## 5. Variabel .env yang Boleh Diubah

| Var | Fungsi | Default | Kapan ubah |
|---|---|---|---|
| `DB_HOST` | host postgres | `localhost` (lokal), `postgres` (Docker) | Jangan ubah manual — compose override |
| `DB_PORT` | port postgres | `5432` | Jika bentrok host, compose map `5433:5432` |
| `DB_MAX_CONNS` | pool collector | `25` | Naikkan jika `too many connections` |
| `COLLECTOR_PORT` | port API | `8081` | Jika bentrok `8080` (adminer) |
| `COLLECTOR_API_SECRET` | token agent (Fase 6) | — | Isi sebelum rollout produksi |

Restart `docker compose up -d` tiap ubah `.env`.

## 6. Rollout 80–100 Device

1. Deploy 5 device dulu (campur pc/laptop), interval 15s, pantau `docker stats` & `SELECT count(*) FROM metrics` 24 jam.
2. Cek `GET /api/servers`, `GET /api/servers/{id}/asset-info`, `GET /api/servers/{id}/applications` dari docs.
3. Tambah 20 device/minggu, pantau `pg_stat_activity` & disk `df -h`.
4. Jika CPU postgres >70% persisten → naikkan interval ke 30s atau tambah vCPU.

## 7. Kapan Perlu Scale Lebih

- >150 device atau interval tetap 5s → pisah postgres ke VM dedicated, atau partisi `metrics` per bulan (PostgreSQL declarative partitioning).
- Pgbouncer hanya jika `DB_MAX_CONNS` >100.

## 8. Cek Kesehatan

```bash
docker compose ps
docker logs monitoring-collector --tail 50
docker exec monitoring-postgres psql -U monitoring_user -d monitoring_db -c "SELECT server_id, count(*) FROM metrics WHERE recorded_at > NOW() - INTERVAL '1 hour' GROUP BY server_id LIMIT 5;"
curl -s http://localhost:8081/api/servers | jq length
```
