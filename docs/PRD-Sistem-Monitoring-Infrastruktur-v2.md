# PRD — Sistem Monitoring Infrastruktur & Asset IT
### PT Tata Optima Property

| | |
|---|---|
| **Dokumen** | Product Requirements Document |
| **Versi** | 2.0 |
| **Tanggal** | 27 September 2026 |
| **Penyusun** | Bagues Putra Tawaqqal — IT Programmer |
| **Status** | Draft |
| **Riwayat** | v1.0 disusun untuk RS Amanah Medical Centre (fokus server); v2.0 direvisi untuk PT Tata Optima Property dengan scope diperluas ke PC/laptop client + asset inventory |

---

## 1. Latar Belakang

PT Tata Optima Property mengelola sejumlah server, PC, dan laptop kerja karyawan tanpa sistem monitoring dan inventarisasi aset terpusat. Kondisi ini menyebabkan:

- Masalah teknis (disk penuh, service down, resource habis) baru diketahui setelah dilaporkan pengguna
- Tidak ada data inventaris hardware yang akurat dan real-time (spesifikasi RAM, disk, serial number) — data asset biasanya usang begitu dicatat manual
- Tidak ada data historis untuk analisis tren pemakaian resource atau evaluasi kebutuhan upgrade hardware
- Tidak ada bukti kuantitatif untuk mendukung laporan kondisi infrastruktur IT ke manajemen

## 2. Tujuan

1. Menyediakan visibilitas real-time terhadap kondisi seluruh perangkat (server, PC, laptop) dalam satu dashboard terpusat
2. Mendeteksi masalah performa secara proaktif sebelum berdampak ke operasional
3. Membangun basis data inventaris hardware (asset management) yang selalu ter-update otomatis, tanpa pendataan manual
4. Menyimpan data historis untuk analisis tren dan perencanaan kapasitas/upgrade
5. Menghasilkan data kuantitatif yang mendukung pelaporan kinerja infrastruktur IT ke manajemen

## 3. Target Pengguna

| Role | Kebutuhan |
|---|---|
| IT Programmer/Coordinator (Bagues) | Full access — lihat semua perangkat, kelola alert, kelola user, kelola data asset |
| Staf IT lain (jika ada) | Lihat dashboard, terima notifikasi alert |
| Manajemen (opsional, fase lanjut) | Ringkasan kondisi infrastruktur secara read-only |

## 4. Lingkup (Scope)

### 4.1 Termasuk dalam scope

**Monitoring (real-time, interval detik):**
- CPU, RAM, Disk usage
- Network throughput (bytes sent/received per detik)
- Status service/proses kritis (dapat dikonfigurasi per device)

**Asset Management (berkala, interval jam/hari):**
- Model, jumlah core, dan clock speed processor
- Tipe & ukuran disk (SSD/HDD), status kesehatan disk (SMART — prediksi kegagalan)
- Total RAM, jumlah slot terpakai/total, tipe DDR
- Kesehatan baterai (khusus perangkat laptop — persentase kapasitas saat ini vs kapasitas desain pabrik)
- Serial number perangkat
- Versi & build OS (Windows)
- IP address dan MAC address perangkat (dilaporkan otomatis oleh agent, bukan input manual)
- Daftar aplikasi terinstall (nama, versi, publisher, tanggal instalasi)

**Platform:**
- Dashboard web (React) — overview seluruh device + detail per device + halaman asset inventory
- REST API (Go) untuk data real-time dan historis
- Notifikasi/alert otomatis (Telegram/email) saat threshold terlampaui
- Autentikasi login untuk akses dashboard
- Riwayat data historis (retensi minimal 30 hari detail, teragregasi untuk retensi jangka panjang)

### 4.2 Tidak termasuk scope saat ini (kemungkinan fase lanjut)

- Remote management (restart service, Wake on LAN, shutdown/restart perangkat jarak jauh)
- Mobile app
- Integrasi dengan sistem tiket/incident management
- Auto-remediation
- Monitoring khusus aplikasi (SIMRS, database internal, dll — spesifik jika dibutuhkan di kemudian hari)

## 5. Arsitektur Sistem

```
┌──────────────────┐                          ┌──────────────────────────────┐
│  PC / Laptop      │                          │   Server Ubuntu               │
│  Karyawan (Win)   │   HTTPS/HTTP POST        │                                │
│  ┌─────────────┐  │──────────────────────────▶  ┌──────────────┐             │
│  │ Agent (Go)  │  │   /api/metrics (5-15s)   │  │  Nginx        │             │
│  │             │  │   /api/asset-info (24h)  │  │  ├─ Reverse   │             │
│  └─────────────┘  │                          │  │  │  Proxy    │             │
└──────────────────┘                          │  │  └─ Serve     │             │
        × N device                             │  │     React    │             │
┌──────────────────┐                          │  └──────┬───────┘             │
│  Server Windows   │                          │         │                     │
│  ┌─────────────┐  │──────────────────────────▶  ┌──────▼───────┐             │
│  │ Agent (Go)  │  │                          │  │  Collector    │             │
│  └─────────────┘  │                          │  │  (Go API)     │             │
└──────────────────┘                          │  └──────┬───────┘             │
                                                │         │                     │
                                                │  ┌──────▼───────┐             │
                                                │  │  PostgreSQL   │             │
                                                │  └──────────────┘             │
                                                └──────────────────────────────┘
```

**Komponen:**

| Komponen | Teknologi | Keterangan |
|---|---|---|
| Agent | Go (binary `.exe`, Windows Service) | Dijalankan di tiap PC/laptop/server Windows; membaca metrics & asset info host asli |
| Collector (API) | Go | REST API — terima data dari agent, simpan ke database, expose data untuk dashboard |
| Database | PostgreSQL (di server Ubuntu) | Simpan data time-series metrics dan data asset inventory |
| Dashboard | React | Visualisasi data, konsumsi REST API dari Collector |
| Reverse Proxy | Nginx | Serve file statis React + proxy request `/api` ke Collector; titik masuk HTTPS |
| Alerting | Go (built-in di Collector) + Telegram Bot API | Evaluasi threshold, kirim notifikasi |
| Deployment pusat | Docker Compose | Collector + PostgreSQL + Nginx dalam satu orkestrasi di server Ubuntu |

## 6. Kebutuhan Fungsional

### 6.1 Agent — Usage Metrics
- FR-1: Membaca metrics CPU, RAM, Disk tiap interval yang dapat dikonfigurasi (default 5 detik)
- FR-2: Membaca throughput network (bytes sent/received per detik)
- FR-3: Mengecek status proses/service tertentu sesuai daftar di `config.yaml`
- FR-4: Mengirim data ke Collector via HTTP POST `/api/metrics`
- FR-5: Berjalan otomatis saat perangkat menyala (Windows Service), tanpa perlu login user
- FR-6: Melakukan buffer lokal & retry bila Collector tidak dapat dihubungi sementara

### 6.2 Agent — Asset Info
- FR-7: Membaca model processor, jumlah core, dan clock speed via WMI (`Win32_Processor`)
- FR-8: Membaca tipe & ukuran disk (SSD/HDD) via WMI (`Win32_DiskDrive`)
- FR-9: Membaca status prediksi kegagalan disk (SMART) via WMI (`MSStorageDriver_FailurePredictStatus`, namespace `root\wmi`)
- FR-10: Membaca total RAM, jumlah keping/slot terpakai, tipe DDR via WMI (`Win32_PhysicalMemory`, `Win32_PhysicalMemoryArray`)
- FR-11: Membaca total slot RAM di motherboard (termasuk yang kosong)
- FR-12: Membaca kesehatan baterai (persentase `FullChargedCapacity` terhadap `DesignedCapacity`) via WMI namespace `root\wmi`, khusus untuk perangkat bertipe laptop; dikosongkan untuk PC/server tanpa baterai
- FR-13: Membaca serial number perangkat via WMI (`Win32_BIOS`)
- FR-14: Membaca versi & build OS
- FR-15: Membaca IP address dan MAC address dari network interface aktif perangkat
- FR-16: Membaca daftar aplikasi terinstall (nama, versi, publisher, tanggal instalasi) melalui Windows Registry (bukan WMI `Win32_Product`, untuk menghindari proses re-validasi MSI yang membebani sistem)
- FR-17: Mengirim asset info sekali saat startup, lalu diperbarui berkala (default tiap 24 jam) ke endpoint `/api/asset-info`

### 6.3 Agent — Konfigurasi
- FR-18: Seluruh parameter (server ID, URL Collector, interval, daftar service yang dicek) dibaca dari file `config.yaml` eksternal, tidak di-hardcode
- FR-19: Identitas tiap perangkat (`server_id`) bersifat unik dan dapat ditelusuri ke hostname aslinya

### 6.4 Collector (API)
- FR-20: Menerima dan memvalidasi data metrics dari agent, menyimpan ke tabel `metrics`
- FR-21: Menerima data asset info dari agent (termasuk processor, disk health, battery health, IP, dan MAC address); melakukan **upsert** (insert jika baru, update jika sudah ada) ke tabel `asset_info`
- FR-22: Menerima daftar aplikasi terinstall dari agent; mengganti (replace) seluruh daftar lama dengan yang baru pada tiap laporan (bukan menumpuk histori)
- FR-23: Menyediakan endpoint `GET /api/servers` — daftar seluruh perangkat terdaftar
- FR-24: Menyediakan endpoint `GET /api/servers/{id}/metrics?range=` — data historis metrics dengan filter rentang waktu
- FR-25: Menyediakan endpoint `GET /api/servers/{id}/status` — kondisi terkini (metrics + service status terbaru)
- FR-26: Menyediakan endpoint `GET /api/servers/{id}/asset-info` — data inventaris hardware terkini, termasuk kesehatan disk & baterai
- FR-27: Menyediakan endpoint `GET /api/servers/{id}/applications` — daftar aplikasi terinstall di perangkat
- FR-28: Mengevaluasi threshold alert setiap data metrics masuk, memicu notifikasi bila terlampaui
- FR-29: Autentikasi request dari agent menggunakan token unik per perangkat (lihat NFR keamanan)

### 6.5 Dashboard
- FR-30: Halaman login dengan autentikasi
- FR-31: Halaman overview — status seluruh perangkat (kartu/list dengan indikator warna: normal/warning/critical), dapat difilter per lokasi/tipe perangkat
- FR-32: Halaman detail per perangkat — grafik historis CPU/RAM/Disk/Network
- FR-33: Halaman **Asset Inventory** — tabel seluruh perangkat dengan spesifikasi hardware (processor, disk, kesehatan disk, RAM, DDR, kesehatan baterai, serial number, IP, MAC address, versi OS), dapat di-export (CSV/Excel)
- FR-34: Halaman **daftar aplikasi terinstall** per perangkat, dengan kemampuan pencarian (misal cari perangkat mana saja yang menginstall aplikasi tertentu)
- FR-35: Indikator peringatan visual di dashboard saat disk diprediksi gagal atau kesehatan baterai di bawah ambang tertentu (misal <50%)
- FR-36: Halaman konfigurasi threshold alert per perangkat/metric
- FR-37: Halaman riwayat alert yang pernah terjadi

### 6.6 Alerting
- FR-38: Kirim notifikasi Telegram/email saat metric melebihi threshold selama durasi tertentu (menghindari false alert dari lonjakan sesaat)
- FR-39: Kirim notifikasi saat perangkat berhenti mengirim data (dianggap offline) lebih dari durasi tertentu
- FR-40: Kirim notifikasi recovery saat kondisi kembali normal
- FR-41: Kirim notifikasi saat disk diprediksi gagal (SMART warning) atau kesehatan baterai menurun drastis

## 7. Kebutuhan Non-Fungsional

| Kategori | Kebutuhan |
|---|---|
| **Keamanan** | Setiap agent mengirim data disertai token autentikasi unik; Collector menolak request tanpa token valid. HTTPS digunakan untuk seluruh komunikasi begitu rollout ke produksi (bukan hanya HTTP polos seperti tahap development). Dashboard wajib login |
| **Reliability** | Kegagalan 1 agent tidak boleh mengganggu agent lain (fault isolation). Agent tidak boleh membebani resource perangkat yang dipantau (target: <1% CPU, <50MB RAM saat idle) |
| **Skalabilitas** | Arsitektur awal menargetkan puluhan hingga ratusan perangkat (PC/laptop karyawan + beberapa server), tanpa perubahan arsitektur besar |
| **Retention data** | Data metrics detail (per beberapa detik) disimpan 30 hari; setelah itu diagregasi per jam untuk retensi jangka panjang (1 tahun). Data asset info disimpan sebagai snapshot terkini (tidak historis) |
| **Auditability** | Setiap perubahan konfigurasi (threshold, user, daftar perangkat) tercatat log — siapa, kapan, perubahan apa |
| **Availability** | Downtime Collector tidak boleh menyebabkan hilang data — agent melakukan buffer lokal saat Collector tidak dapat dihubungi |
| **Deployment** | Agent dapat di-deploy secara massal ke banyak perangkat (skrip deployment atau Group Policy jika tersedia Active Directory), tanpa perlu instalasi manual satu per satu |
| **Privasi** | Monitoring dibatasi pada data resource, hardware, dan daftar aplikasi terinstall; tidak mencakup pengawasan aktivitas pengguna (tidak ada screen capture, keylogging, riwayat browsing, atau pemantauan konten kerja karyawan) |

## 8. Model Data

```
servers
├── id
├── server_id        (unik, dari config.yaml agent)
├── hostname
├── ip_address        (disinkronkan otomatis dari laporan asset_info agent, bukan input manual)
├── device_type       (server | pc | laptop)
├── location           (opsional: nama ruangan/departemen)
├── auth_token
├── is_active
└── created_at

metrics
├── id
├── server_id (FK)
├── cpu_percent
├── ram_percent
├── disk_percent
├── network_sent_bps
├── network_recv_bps
├── recorded_at
└── created_at

service_status
├── id
├── server_id (FK)
├── service_name
├── is_running
├── checked_at
└── created_at

asset_info                      (1 baris per perangkat, di-UPSERT)
├── id
├── server_id (FK, unique)
├── processor_model
├── processor_cores
├── processor_clock_ghz
├── disk_type            (SSD | HDD)
├── disk_size_gb
├── disk_health_status    (OK | WARNING — dari SMART PredictFailure)
├── ram_total_gb
├── ram_slots_used
├── ram_slots_total
├── ram_ddr_type          (DDR3 | DDR4 | DDR5)
├── battery_health_percent (nullable — hanya untuk device_type = laptop)
├── serial_number
├── ip_address            (dilaporkan otomatis oleh agent, bukan input manual)
├── mac_address
├── os_version
└── updated_at

installed_applications           (banyak baris per perangkat, di-replace tiap laporan)
├── id
├── server_id (FK)
├── app_name
├── app_version
├── publisher
├── install_date
└── reported_at

alert_rules
├── id
├── server_id (FK)
├── metric_type
├── operator             (greater_than | less_than)
├── threshold_value
└── duration_seconds

alerts
├── id
├── server_id (FK)
├── metric_type
├── threshold_value
├── triggered_at
├── resolved_at
└── notified_via          (telegram | email)
```

## 9. Spesifikasi API (ringkasan)

| Method | Endpoint | Fungsi |
|---|---|---|
| POST | `/api/metrics` | Agent mengirim data usage metrics |
| POST | `/api/asset-info` | Agent mengirim/update data inventaris hardware |
| GET | `/api/servers` | Daftar semua perangkat terdaftar |
| GET | `/api/servers/{id}/metrics?range=1h` | Data metrics historis 1 perangkat |
| GET | `/api/servers/{id}/status` | Kondisi terkini 1 perangkat |
| GET | `/api/servers/{id}/asset-info` | Data inventaris hardware 1 perangkat |
| GET | `/api/servers/{id}/applications` | Daftar aplikasi terinstall 1 perangkat |
| GET | `/api/alerts` | Riwayat alert (fase lanjut) |

Seluruh endpoint POST dari agent memerlukan header autentikasi token; seluruh endpoint GET untuk dashboard memerlukan sesi login yang valid (fase autentikasi).

## 10. Fase Pengembangan (Roadmap)

| Fase | Cakupan | Status |
|---|---|---|
| **Fase 0** | Setup project (struktur repo, Git workflow, dual remote GitLab+GitHub) | ✅ Selesai |
| **Fase 1** | Agent: baca CPU/RAM/Disk + config file eksternal | ✅ Selesai |
| **Fase 2** | Collector: terima data via HTTP, simpan ke PostgreSQL | ✅ Selesai |
| **Fase 3** | Agent: tambah network usage & service uptime check | ✅ Selesai |
| **Fase 4** | Collector: REST API GET (servers, metrics history, status) | ✅ Selesai |
| **Fase 5** | Agent & Collector: Asset Info (disk type, RAM/DDR, serial number, OS version) | 🚧 Sedang berjalan |
| **Fase 6** | Keamanan: autentikasi token per-agent, HTTPS | Belum dimulai |
| **Fase 7** | Dashboard React: overview, detail perangkat, asset inventory | Belum dimulai |
| **Fase 8** | Autentikasi login dashboard + audit log | Belum dimulai |
| **Fase 9** | Alerting: threshold rule + notifikasi Telegram/email | Belum dimulai |
| **Fase 10** | Deploy Agent sebagai Windows Service + strategi deployment massal | Belum dimulai |
| **Fase 11** | Rollout bertahap: mulai dari beberapa PC/laptop non-kritis → evaluasi → rollout penuh | Belum dimulai |

> Catatan: rollout ke server produksi (jika ada) tetap wajib dilakukan paling akhir dan bertahap, setelah stabil di PC/laptop non-kritis.

## 11. Risiko & Mitigasi

| Risiko | Mitigasi |
|---|---|
| Agent belum teruji di banyak variasi hardware/driver (WMI query bisa gagal di sebagian device) | Setiap query WMI dibungkus error handling individual — kegagalan 1 data tidak menggagalkan seluruh laporan |
| Rollout ke banyak PC/laptop sekaligus berisiko menimbulkan masalah yang sulit dilacak | Rollout bertahap: mulai dari beberapa device dulu, pantau stabilitas, baru perluas |
| Development di luar jam kerja penuh (dikerjakan di sela pekerjaan utama) berpotensi molor | Fase dipecah kecil-kecil per fitur, tiap fase menghasilkan sesuatu yang bisa diuji |
| Tanpa autentikasi token, siapapun di jaringan bisa mengirim data palsu ke Collector | Autentikasi token per-agent diprioritaskan sebelum rollout ke luar lingkungan development |
| Data asset info dapat berubah signifikan setelah upgrade hardware manual | Interval refresh berkala (24 jam) memastikan data tidak basi lebih dari 1 hari |

## 12. Metrik Keberhasilan (Success Criteria)

- Seluruh perangkat target (PC/laptop/server) terpantau dalam satu dashboard terpusat
- Data asset inventory tersedia otomatis untuk seluruh perangkat tanpa pendataan manual
- Alert terkirim dalam waktu kurang dari 1 menit sejak threshold terlampaui
- Overhead resource agent di perangkat target tetap di bawah 1% CPU dan 50MB RAM
- Data historis 30 hari tersedia untuk analisis tren
- Tidak ada gangguan tambahan pada perangkat produksi akibat pemasangan agent

---

*Dokumen ini akan terus disempurnakan seiring proses development berjalan.*
