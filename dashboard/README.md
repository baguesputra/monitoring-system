# Seneschal Dashboard

Estate Steward untuk monitoring infrastruktur & inventaris aset PT Tata Optima Property.

## Teknologi

- **React 19** + **Vite** (dev server + build)
- **Tailwind CSS v4** (styling via `@tailwindcss/vite`)
- **Recharts** (grafik time-series)
- **Lucide React** (ikona)
- **Oxlint** (lint cepat)

## Arsitektur Visual (Seneschal Design)

Desain mengadopsi estetika *European Estate Ledger*:
- Parchment background (`#F4F1EA`) dengan tekstur kertas halus
- Ink black (`#2C2621`) tipografi
- Gold leaf (`#C5A059`) aksen
- Wax-seal badge untuk status server (NOMINAL/ATTENTION/CRITICAL/OFFLINE)

## Menjalankan

```bash
# Install dependencies
npm install

# Development (port 3000, proxy /api ke collector :8081)
npm run dev

# Production build
npm run build

# Preview build
npm run preview
```

## API Endpoint (via proxy Vite)

| Path | Deskripsi |
|------|-----------|
| `/api/servers` | Daftar perangkat |
| `/api/servers/{id}/status` | Metrik terkini + status service |
| `/api/servers/{id}/metrics?range=1h` | Histori metrik |
| `/api/servers/{id}/asset-info` | Inventaris hardware |
| `/api/servers/{id}/users` | Akun user lokal & privilege |
| `/api/servers/{id}/applications` | Daftar aplikasi terinstall |
| `PUT /api/servers/{id}/metadata` | Update owner, department, location, notes |

## Desain Token (CSS)

Semua token di `src/index.css`:
- `--color-parchment`, `--color-ink`, `--color-gold-leaf`, `--color-forest-moss`, `--color-cardinal`
- Font: `Fraunces` (display) / `IBM Plex Mono` (data)
- Wax seal status: `.wax-seal-normal`, `.wax-seal-warning`, `.wax-seal-critical`, `.wax-seal-offline`
- Ledger card: `.ledger-card`, `.ledger-card-elevated`