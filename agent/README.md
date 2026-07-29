# Monitoring Agent

Agent monitoring yang berjalan di tiap server untuk membaca metrics sistem
(CPU, RAM, Disk) secara berkala dan mengirimkannya ke Collector.

## Status

🚧 Development — saat ini agent baru bisa membaca metrics dan menampilkan
ke console. Pengiriman data ke Collector API akan ditambahkan di fitur berikutnya.

## Requirements

- Go 1.21 atau lebih baru
- Windows Server (target deployment)

## Menjalankan secara lokal

```bash
cd agent
go run main.go
```

Agent akan membaca dan menampilkan metrics setiap 5 detik:

[15:20:00] CPU Usage: 10.20%
[15:20:01] RAM Usage: 45.30% (Used: 7.20 GB / Total: 16.00 GB)
[15:20:01] Disk Usage (C:): 62.10% (Used: 124.50 GB / Total: 200.00 GB)


## Metrics yang dikumpulkan

| Metric | Deskripsi | Interval |
|---|---|---|
| CPU Usage | Persentase penggunaan CPU (agregat semua core) | 5 detik |
| RAM Usage | Persentase & jumlah penggunaan memory | 5 detik |
| Disk Usage | Persentase & jumlah penggunaan disk drive `C:` | 5 detik |

## Dependencies

- [gopsutil](https://github.com/shirou/gopsutil) — library untuk membaca
  metrics sistem secara cross-platform

## Roadmap

- [ ] Kirim data ke Collector API via HTTP
- [ ] Baca network usage
- [ ] Service/process uptime check
- [ ] Konfigurasi interval & target Collector via file config
- [ ] Jalankan sebagai Windows Service