# PRD — Sistem Monitoring Infrastruktur Server
### Amanah Medical Centre

| | |
|---|---|
| **Dokumen** | Product Requirements Document |
| **Versi** | 1.0 |
| **Tanggal** | 29 Juli 2026 |
| **Penyusun** | Bagues — IT Coordinator |
| **Status** | Draft |

---

## 1. Latar Belakang

Amanah Medical Centre saat ini mengelola sekitar 10 server yang mencakup infrastruktur kritis (SIMRS, database SQL Server, CCTV, telephony, dan aplikasi internal lainnya) tanpa sistem monitoring terpusat. Kondisi server saat ini dipantau secara manual, sehingga:

- Masalah (disk penuh, service down, resource habis) baru diketahui setelah berdampak ke pengguna/pasien
- Tidak ada data historis untuk analisis kapasitas atau evaluasi performa infrastruktur
- Tidak ada bukti kuantitatif untuk mendukung laporan evaluasi infrastruktur ke direktur

## 2. Tujuan

1. Menyediakan visibilitas real-time terhadap kondisi seluruh server (±10 unit) dalam satu dashboard terpusat
2. Mendeteksi masalah secara proaktif sebelum berdampak ke layanan (SIMRS, billing, rekam medis)
3. Menyimpan data historis untuk analisis tren dan perencanaan kapasitas
4. Menghasilkan data kuantitatif yang mendukung pelaporan kinerja infrastruktur IT ke manajemen

## 3. Target Pengguna

| Role | Kebutuhan |
|---|---|
| IT Coordinator (Bagues) | Full access — lihat semua server, kelola alert, kelola user |
| Staf IT (4 personel) | Lihat dashboard, terima notifikasi alert sesuai server yang jadi tanggung jawabnya |
| Direktur/Manajemen (opsional, fase lanjut) | Lihat ringkasan uptime/kesehatan infrastruktur secara read-only |

## 4. Lingkup (Scope)

### 4.1 Termasuk dalam scope (Fase 1)
- Monitoring resource dasar: CPU, RAM, Disk usage
- Monitoring network: bandwidth in/out
- Monitoring service/process uptime (misal: apakah service Django SIMRS, Nginx, SQL Server masih berjalan)
- Dashboard web menampilkan status seluruh server real-time
- Notifikasi/alert otomatis (Telegram/email) saat threshold terlampaui
- Riwayat data historis (minimal 30 hari)
- Autentikasi login untuk akses dashboard

### 4.2 Tidak termasuk scope Fase 1 (kemungkinan fase lanjut)
- Custom metrics aplikatif (query performance SQL Server, jumlah antrian SIMRS, dll)
- Role-based access control granular per server
- Mobile app
- Integrasi otomatis dengan sistem tiket/incident management
- Auto-remediation (restart service otomatis, dll)

## 5. Arsitektur Sistem

```
┌──────────────┐   HTTPS POST   ┌──────────────┐   SQL    ┌──────────────┐
│  Agent (Go)  │───────────────▶│  Collector   │─────────▶│  PostgreSQL  │
│  di tiap     │   tiap 15-30s  │  API (Go)    │          │              │
│  server (x10)│                │              │◀─────────│              │
└──────────────┘                └──────┬───────┘   query  └──────────────┘
                                        │ REST API
                                        ▼
                                 ┌──────────────┐
                                 │  Dashboard   │
                                 │  (React)     │
                                 └──────────────┘
                                        │
                                 ┌──────▼───────┐
                                 │ Alertmanager │
                                 │ (Telegram/   │
                                 │  Email)      │
                                 └──────────────┘
```

**Komponen:**

| Komponen | Teknologi | Keterangan |
|---|---|---|
| Agent | Go (binary, systemd service) | Jalan di tiap server, baca metrics host asli — **tidak** dijalankan via Docker agar bisa membaca kondisi host langsung |
| Collector (API) | Go | Terima data dari agent, simpan ke database, expose REST API untuk dashboard |
| Database | PostgreSQL | Simpan data time-series metrics |
| Dashboard | React | Visualisasi data, konsumsi REST API dari Collector |
| Alerting | Go (built-in di Collector) + Telegram Bot API | Cek threshold, kirim notifikasi |
| Deployment pusat | Docker Compose | Collector + PostgreSQL + Dashboard di-container-kan di 1 server pusat |

## 6. Kebutuhan Fungsional

### 6.1 Agent
- FR-1: Agent membaca metrics CPU, RAM, Disk, Network tiap interval yang dapat dikonfigurasi (default 15 detik)
- FR-2: Agent mengecek status service tertentu yang dikonfigurasi (misal: proses `nginx`, `gunicorn`, port SQL Server)
- FR-3: Agent mengirim data ke Collector via HTTPS dengan autentikasi token per-server
- FR-4: Agent tetap berjalan otomatis saat server reboot (systemd)
- FR-5: Agent melakukan retry/buffer lokal bila Collector tidak dapat dihubungi sementara (agar data tidak hilang)

### 6.2 Collector (API)
- FR-6: Menerima dan memvalidasi data dari agent (autentikasi token, validasi format)
- FR-7: Menyimpan data ke PostgreSQL
- FR-8: Menyediakan endpoint REST API untuk dashboard (data realtime & historis)
- FR-9: Mengevaluasi threshold alert setiap data masuk, dan memicu notifikasi bila terlampaui
- FR-10: Endpoint untuk manajemen server terdaftar (tambah/hapus server yang dipantau)

### 6.3 Dashboard
- FR-11: Halaman login dengan autentikasi
- FR-12: Halaman overview — status seluruh server (kartu/list dengan indikator warna: normal/warning/critical)
- FR-13: Halaman detail per server — grafik historis CPU/RAM/Disk/Network
- FR-14: Halaman konfigurasi threshold alert per server/metric
- FR-15: Halaman riwayat alert yang pernah terjadi

### 6.4 Alerting
- FR-16: Kirim notifikasi ke Telegram/email saat metric melebihi threshold selama durasi tertentu (menghindari false alert dari spike sesaat)
- FR-17: Kirim notifikasi saat server berhenti mengirim data (dianggap down) lebih dari X menit
- FR-18: Notifikasi recovery saat kondisi kembali normal

## 7. Kebutuhan Non-Fungsional

| Kategori | Kebutuhan |
|---|---|
| **Keamanan** | Komunikasi agent↔collector wajib HTTPS + token autentikasi per-server. Dashboard wajib login. Karena berjalan di jaringan RS, isolasi network (VLAN/firewall) untuk server Collector sangat direkomendasikan |
| **Reliability** | Collector harus tetap menerima data meski salah satu agent bermasalah (fault isolation). Agent tidak boleh membebani resource server yang dipantau (target: <1% CPU, <50MB RAM) |
| **Skalabilitas** | Desain awal untuk ±10 server, namun arsitektur harus mampu menampung hingga 30-50 server tanpa perubahan besar |
| **Retention data** | Data detail (per 15 detik) disimpan 30 hari, kemudian diagregasi (per jam) untuk retensi jangka panjang (1 tahun) |
| **Auditability** | Setiap perubahan konfigurasi (threshold, user, server) tercatat log (siapa, kapan, apa yang diubah) — selaras dengan pola audit log yang sudah diterapkan di Data Integrity Center |
| **Availability** | Downtime Collector tidak boleh menyebabkan hilang data — agent buffer lokal saat Collector unreachable |

## 8. Model Data (garis besar)

```
servers
├── id
├── hostname
├── ip_address
├── location (lokasi fisik/gedung, opsional)
├── auth_token
└── created_at

metrics
├── id
├── server_id (FK)
├── metric_type (cpu | ram | disk | network_in | network_out)
├── value
└── recorded_at

service_status
├── id
├── server_id (FK)
├── service_name
├── status (up | down)
└── checked_at

alerts
├── id
├── server_id (FK)
├── metric_type
├── threshold_value
├── triggered_at
├── resolved_at
└── notified_via (telegram | email)

alert_rules
├── id
├── server_id (FK)
├── metric_type
├── operator (greater_than | less_than)
├── threshold_value
└── duration_seconds
```

## 9. Fase Pengembangan (Roadmap)

| Fase | Cakupan | Estimasi |
|---|---|---|
| **Fase 0** | Setup project structure, repo, Docker Compose skeleton (PostgreSQL + Collector kosong) | 2-3 hari |
| **Fase 1** | Agent Go: baca CPU/RAM/Disk, kirim ke Collector dummy | 3-5 hari |
| **Fase 2** | Collector: terima data, simpan ke DB, REST API dasar | 3-5 hari |
| **Fase 3** | Dashboard React: overview + detail server dengan grafik | 5-7 hari |
| **Fase 4** | Network monitoring + service uptime check di Agent | 3-4 hari |
| **Fase 5** | Alerting (threshold rule + Telegram/email notifikasi) | 4-5 hari |
| **Fase 6** | Autentikasi dashboard + audit log | 2-3 hari |
| **Fase 7** | Deploy bertahap ke 2-3 server non-kritis dulu → evaluasi → rollout ke semua server | fleksibel |

> Catatan: karena target langsung ke server RS, **rollout wajib bertahap** — mulai dari server non-kritis untuk validasi stabilitas agent sebelum dipasang di server yang menjalankan SIMRS/database produksi.

## 10. Risiko & Mitigasi

| Risiko | Mitigasi |
|---|---|
| Agent custom belum teruji, berpotensi mengganggu server produksi | Rollout bertahap, mulai dari server non-kritis; batasi resource agent |
| Development di luar jam kerja penuh (freelance/personal + tugas RS) berpotensi molor | Fase dipecah kecil-kecil, prioritaskan Fase 1-3 sebagai MVP fungsional dulu |
| Kurangnya pengalaman Go untuk kasus production-grade | Mulai dengan gopsutil (library matang) daripada baca `/proc` manual, kurangi risiko bug |
| Kehilangan data saat Collector down | Buffer lokal di agent + retry mechanism |

## 11. Metrik Keberhasilan (Success Criteria)

- Seluruh ±10 server terpantau dalam 1 dashboard terpusat
- Alert terkirim dalam <1 menit sejak threshold terlampaui
- Overhead resource agent di server target <1% CPU dan <50MB RAM
- Data historis 30 hari tersedia untuk analisis tren
- Zero downtime tambahan pada server produksi akibat pemasangan agent

---

*Dokumen ini adalah draft awal dan akan disempurnakan seiring proses development.*
