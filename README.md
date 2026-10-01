# Seneschal Infrastructure Monitoring System

> **Estate Steward for Infrastructure & Hardware Asset Monitoring**  
> PT Tata Optima Property — IT Programme Coordinator

---

## Quick Start

### Server (Docker Compose)

```bash
# 1. Configure environment
cp .env.example .env
# Edit .env with secure passwords

# 2. Start stack
docker compose up -d --build

# 3. Verify
docker compose ps
curl http://localhost:8081/api/servers
```

### Client Agent (Windows)

```powershell
# Run as Administrator
.\install.ps1 -CollectorURL "http://192.168.1.50:8081" `
              -ServerID "PC-HRD-01" `
              -DeviceType pc
```

### Dashboard

```bash
cd dashboard
npm install
npm run dev        # http://localhost:3000
npm run build      # Production build
```

---

## Features

- **Real-time telemetry**: CPU, RAM, Disk, Network (5-15s interval)
- **Hardware inventory**: CPU, RAM (slots/DDR), Disk (SMART %), Battery, Serial, OS, IP/MAC
- **Software inventory**: Installed apps from Windows Registry
- **User accounts**: Local accounts with privilege detection (Admin/Standard/Guest)
- **Service monitoring**: Configurable process checks
- **Device assignment**: Owner, department, location, notes
- **Device rename**: Full server_id migration with cascade
- **Seneschal UI**: Estate ledger aesthetic with wax-seal status badges

---

## Architecture

```
Windows Clients          Collector (Go)          PostgreSQL         Dashboard (React)
┌─────────────┐    /api/metrics,      ┌─────────────┐    ┌──────────────────┐
│   Agent     │  /api/asset-info ────▶│  Collector  │───▶│   Database      │
│  (Go/WMI)   │    (5-15s / 24h)     │  :8081      │    │  metrics,       │
└─────────────┘                       └──────┬──────┘    │  asset_info,    │
                                             │           │  device_users,  │
                                    ┌────────▼──────┐   │  apps, users    │
                                    │    Nginx      │   └────────┬────────┘
                                    │  Reverse Proxy│            │
                                    │  :80/443      │            │
                                    └───────────────┘            │
                                                                 │
                                            ┌────────────────────▼──────┐
                                            │    Dashboard (React)       │
                                            │  Daily Ledger / Inventory │
                                            │  Provisions / Device Detail│
                                            └───────────────────────────┘
```

---

## Documentation

Full documentation: [DOCUMENTATION.md](DOCUMENTATION.md)

---

## Components

| Component | Language | Purpose |
|-----------|----------|---------|
| `agent/` | Go | Windows telemetry & inventory collector |
| `collector/` | Go | REST API, data ingestion, PostgreSQL |
| `dashboard/` | React/Vite | Web UI (Seneschal Estate Ledger design) |
| `collector/migrations/` | SQL | Database schema & migrations |

---

## Agent Hardening

- **NTFS ACL**: `C:\ProgramData\Seneschal\` — Users=ReadOnly
- **Windows Service**: Runs as SYSTEM, auto-restart
- **Embedded Config**: Runs without `config.yaml` (BIOS Serial fallback)
- **Config Priority**: Embedded defaults < config.yaml

---

## Requirements

- **Server**: Ubuntu 22.04+, Docker & Docker Compose
- **Clients**: Windows 10/11, Server 2019+
- **Network**: Client → Collector port 8081 (HTTP) or 443 (HTTPS)

---

## License

Internal use only — PT Tata Optima Property