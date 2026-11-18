# Planning Deployment — Seneschal Infrastructure Monitoring System

Dokumen ini mencatat rencana arsitektur dan langkah deployment **Seneschal Monitoring System** menggunakan **aaPanel + Docker + Subdomain HTTPS** agar PC, laptop, dan server kantor dapat melapor dari jaringan mana saja.

---

## 1. Ringkasan Sistem

Proyek ini terdiri dari 3 komponen utama:
1. **`agent/` (Go, Windows Service):** Dipasang di setiap PC/laptop/server Windows (`SeneschalAgent`). Mengirim telemetri CPU, RAM, Disk, Network, dan status service tiap ~15 detik ke `/api/metrics`, serta inventori hardware, software terinstal, dan user lokal tiap start & per 24 jam ke `/api/asset-info`.
2. **`collector/` & PostgreSQL (Go + PostgreSQL 16):** Backend REST API (port `8081`) yang menerima laporan agent, menyimpan ke PostgreSQL, dan melayani request dari Dashboard.
3. **`dashboard/` (React + Vite + Tailwind):** Tampilan web (*Daily Ledger*, *Device Detail*, *Estate Inventory*, *Provisions*, *API Reference*). Semua pemanggilan API menggunakan path relatif `/api/...`.

---

## 2. Arsitektur Deployment yang Dipilih (Hybrid: aaPanel + Docker)

Menggunakan **1 subdomain** (contoh: `monitor.kantorkamu.com`) untuk melayani Dashboard sekaligus Collector API melalui Nginx aaPanel:

```text
[Agent Windows di mana saja]
          │
          │ HTTPS POST (/api/metrics, /api/asset-info)
          ▼
[aaPanel Nginx + SSL: https://monitor.kantorkamu.com]
     ├── /       ──▶ Static files hasil build Dashboard (dashboard/dist/)
     └── /api/   ──▶ Reverse Proxy ke http://127.0.0.1:8081
                          │
                          ▼
               [Docker: monitoring-collector]
                          │
                          ▼
               [Docker: monitoring-postgres] (migrations SQL otomatis)
```

### Alasan Memilih Setup Ini
- **Backend (`postgres` + `collector`) tetap di Docker (`docker-compose.yml`):**
  - Tidak perlu install Go versi `1.26.5` (sesuai `collector/go.mod`) di server aaPanel karena `collector/Dockerfile` sudah otomatis build memakai `golang:1.26-alpine`.
  - Migrasi SQL (`collector/migrations/`) otomatis dijalankan saat inisialisasi database pertama kali.
  - Urutan startup aman (`collector` menunggu `postgres` healthy) dan auto-restart (`restart: unless-stopped`).
- **Frontend (`dashboard/`) cukup sebagai Static Website di aaPanel:**
  - Hasil `npm run build` (`dashboard/dist/`) ringan, mudah diberi SSL Let's Encrypt dan proteksi Basic Auth langsung dari panel Nginx aaPanel.
  - Karena Dashboard memanggil `/api/...` secara relatif, cukup 1 subdomain saja tanpa perlu ubah kode frontend atau masalah CORS lintas subdomain.

---

## 3. Rencana Langkah Implementasi

### Tahap 1 — Menjalankan Backend Docker di Server aaPanel
- [ ] Salin `.env.example` menjadi `.env`, lalu isi `DB_USER`, `DB_PASSWORD`, dan `COLLECTOR_API_SECRET` dengan nilai yang kuat.
- [ ] Jalankan stack Docker dari root folder project:
  ```bash
  docker compose up -d --build
  ```
- [ ] Pastikan kedua container (`monitoring-postgres` dan `monitoring-collector`) berstatus `Up` / `healthy`:
  ```bash
  docker compose ps
  ```
- [ ] **Pengamanan port host:** Tutup port `8081` dan `5433` di Firewall / menu Security aaPanel (atau ubah binding di `docker-compose.yml` menjadi `127.0.0.1:8081:8081` dan `127.0.0.1:5433:5432`) agar akses dari luar hanya bisa masuk melalui Nginx Reverse Proxy.

### Tahap 2 — Build Dashboard & Setup Subdomain di aaPanel
- [ ] Arahkan DNS record (`A` / `CNAME`) subdomain (misal `monitor.kantorkamu.com`) ke IP server aaPanel.
- [ ] Build Dashboard untuk production di folder `dashboard/`:
  ```bash
  npm install
  npm run build
  ```
- [ ] Di aaPanel → **Website → Add Site**:
  - Domain: `monitor.kantorkamu.com`
  - PHP Version: **Static**
  - Root directory: arahkan ke / salin isi folder `dashboard/dist/`.
- [ ] Aktifkan **SSL (Let's Encrypt)** pada site tersebut dan nyalakan **Force HTTPS**.
- [ ] Tambahkan konfigurasi Reverse Proxy di tab **Config** Nginx site tersebut agar `/api/` diteruskan ke container Collector:
  ```nginx
  location /api/ {
      proxy_pass http://127.0.0.1:8081;
      proxy_set_header Host $host;
      proxy_set_header X-Real-IP $remote_addr;
      proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
      proxy_set_header X-Forwarded-Proto $scheme;
  }
  ```

### Tahap 3 — Pengamanan Sebelum Go-Live ke Internet Publik (Penting)
> **Kondisi kode saat ini:** `collector/main.go` belum memiliki middleware autentikasi token (variabel `COLLECTOR_API_SECRET` di `.env.example` belum digunakan di kode, dan fitur token per-agent masih berstatus rencana Fase 6).

Karena subdomain akan terbuka ke internet, lakukan pengamanan berikut sebelum dipakai secara luas:
- [ ] **Kunci akses Dashboard & endpoint manajemen** (`/`, `GET /api/servers*`, `PUT /api/servers/*/metadata`, `PUT /api/servers/*/rename`):
  - Pasang **Basic Auth** di Nginx/aaPanel atau **Cloudflare Access** (OTP email kantor) — pastikan mengecualikan `POST /api/metrics` dan `POST /api/asset-info` agar agent tetap bisa mengirim laporan secara otomatis tanpa login interaktif.
- [ ] **Kunci endpoint penerima data agent (`POST /api/metrics` & `POST /api/asset-info`):**
  - Tambahkan pengecekan secret header / Bearer Token di `collector` dan `agent` (atau validasi custom header rahasia di Nginx) agar tidak sembarang pihak dari internet bisa mengirim data palsu.

### Tahap 4 — Build & Pasang Agent di Laptop/PC Kantor
- [ ] Build ulang agent dengan URL subdomain HTTPS (opsional jika tidak ingin mengandalkan `config.yaml` saja), dari folder `agent/`:
  ```powershell
  .\build.ps1 -CollectorURL "https://monitor.kantorkamu.com" -IntervalSeconds 15 -DeviceType "laptop"
  ```
- [ ] Instal di setiap perangkat Windows target (PowerShell sebagai Administrator):
  ```powershell
  .\install.ps1 -CollectorURL "https://monitor.kantorkamu.com" -DeviceType "laptop"
  ```
  *(Jika `-ServerID` dikosongkan, `server_id` otomatis mengambil Serial Number BIOS / hostname).*
- [ ] **Catatan perilaku saat offline:** Pada `agent/main.go`, pengiriman memakai `http.Post` langsung tanpa antrean buffer lokal. Saat laptop tidak terhubung ke internet, metrik pada siklus tersebut akan dilewati dan dicatat di log; begitu internet tersambung kembali, pelaporan otomatis berjalan lagi.

---

## 4. Alternatif Jika Ingin Tetap Tertutup (Tanpa Ekspos ke Publik)

Jika di suatu kondisi tidak ingin subdomain atau endpoint monitoring dibuka ke internet publik sama sekali:
- Gunakan **Mesh VPN** (seperti **Tailscale**, **ZeroTier**, atau **WireGuard**) di server Collector dan di setiap laptop/PC kantor.
- Arahkan `CollectorURL` ke IP VPN virtual server Collector (misal `http://100.x.y.z:8081`).

---

## 5. Checklist Verifikasi Akhir

- [ ] `docker compose ps` menunjukkan `monitoring-postgres` dan `monitoring-collector` sehat (`Up`).
- [ ] `curl https://monitor.kantorkamu.com/api/servers` mengembalikan JSON `[]` atau daftar perangkat (atau `401` jika Basic Auth sudah aktif).
- [ ] Membuka `https://monitor.kantorkamu.com` di browser menampilkan Dashboard Seneschal tanpa error API di Console/Network tab.
- [ ] Service `SeneschalAgent` berstatus `Running` di laptop uji coba (baik dari Wi-Fi kantor maupun jaringan luar/tethering HP) dan perangkat langsung muncul di tab **Daily Ledger**.
