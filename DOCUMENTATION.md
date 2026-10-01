# Seneschal Infrastructure Monitoring System

> **Estate Steward for Infrastructure & Hardware Asset Monitoring**  
> PT Tata Optima Property — IT Programme Coordinator

---

## Table of Contents

1. [Overview](#overview)
2. [Architecture](#architecture)
3. [Components](#components)
4. [Installation & Deployment](#installation--deployment)
5. [Agent Hardening & Security](#agent-hardening--security)
6. [Dashboard](#dashboard)
7. [API Reference](#api-reference)
8. [Database Schema](#database-schema)
9. [Configuration](#configuration)
10. [Operations & Maintenance](#operations--maintenance)
11. [Troubleshooting](#troubleshooting)
12. [Development](#development)

---

## Overview

**Seneschal** is an infrastructure monitoring system designed for PT Tata Optima Property to provide centralized visibility into the health, performance, and hardware inventory of servers, PCs, and laptops across the organization.

### Key Capabilities

- **Real-time Telemetry**: CPU, RAM, Disk, Network throughput (5–15s interval)
- **Hardware Asset Inventory**: Processor, RAM (slots/DDR type), Disk (SMART health %), Battery, Serial Number, OS, IP/MAC
- **Software Inventory**: Installed applications from Windows Registry
- **Local User Accounts**: Enumeration with privilege detection (Administrator/Standard/Guest)
- **Service Monitoring**: Configurable process/service status checks
- **Device Assignment**: Editable owner, department, location, notes
- **Device Rename**: Full server_id migration with cascade to historical data
- **Alerting Ready**: Threshold-based status (Nominal/Attention/Critical/Offline)

### Design Philosophy

Seneschal adopts a **European Estate Ledger** aesthetic — parchment background, ink typography, gold accents, wax-seal status badges — reflecting the steward's role in managing the estate's infrastructure.

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           SENESCHAL ARCHITECTURE                             │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌──────────────┐    HTTPS/HTTP POST      ┌──────────────────────────────┐  │
│  │  PC/Laptop    │────────────────────────▶│  Collector (Go API)          │  │
│  │  (Windows)    │  /api/metrics (5-15s)   │  :8081                       │  │
│  │  ┌────────┐  │  /api/asset-info (24h)  │  ┌────────────────────────┐   │  │
│  │  │ Agent  │  │                         │  │ Nginx Reverse Proxy    │   │  │
│  │  │ (Go)   │  │                         │  │ :80/443 → :8081        │   │  │
│  │  └────────┘  │                         │  └───────────┬────────────┘   │  │
│  └──────┬───────┘                         │              │                │  │
│         × N devices                        │              ▼                │  │
│                                          │  ┌────────────────────────┐   │  │
│  ┌──────────────┐                         │  │ PostgreSQL 16          │   │  │
│  │  Server      │────────────────────────▶│  │ metrics, asset_info,   │   │  │
│  │  (Windows)   │                         │  │ service_status,        │   │  │
│  │  ┌────────┐  │                         │  │ device_users, apps     │   │  │
│  │  │ Agent  │  │                         │  └────────────────────────┘   │  │
│  │  └────────┘  │                         └──────────────────────────────┘  │  │
│  └──────────────┘                                                            │
│                                                                              │
│  ┌──────────────────────────────────────────────────────────────────────┐   │
│  │                    Dashboard (React + Vite)                           │   │
│  │  Daily Ledger | Estate Inventory | Provisions Register | Device Detail│   │
│  └──────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Data Flow

1. **Agent** (Go, Windows) collects metrics every 5–15s → POST `/api/metrics`
2. **Agent** collects asset inventory every 24h → POST `/api/asset-info`
3. **Collector** validates, stores in PostgreSQL, evaluates thresholds
4. **Dashboard** polls Collector REST API → visualizes in ledger-style UI
5. **Alerting** (future) evaluates thresholds → Telegram/Email notifications

---

## Components

### 1. Agent (`agent/`)

**Technology**: Go 1.21+, WMI (`github.com/yusufpapurcu/wmi`), gopsutil, Windows Registry

**Capabilities**:
- CPU/RAM/Disk/Network metrics (gopsutil)
- Hardware inventory via WMI (Win32_Processor, Win32_DiskDrive, Win32_PhysicalMemory, Win32_BIOS, Battery, Win32_UserAccount)
- SMART disk health prediction (MSStorageDriver_FailurePredictStatus)
- Disk wear percentage from vendor-specific SMART data
- Installed applications (Registry: HKLM\Software\Microsoft\Windows\CurrentVersion\Uninstall + WOW6432Node)
- Local user accounts with privilege detection (Administrators group membership)
- Network identity (IP/MAC of active non-loopback interface)
- Configurable service/process checks

**Configuration** (`config.yaml`):
```yaml
server_id: "PC-HRD-01"        # Unique device identifier
device_type: "pc"              # server | pc | laptop
collector_url: "http://192.168.1.50:8081"
interval_seconds: 15           # Telemetry interval
services_to_check:             # Processes to monitor
  - "nginx"
  - "postgres"
```

**Embedded Config** (build-time via `-ldflags`):
```bash
go build -ldflags \
  "-X main.embeddedCollectorURL=http://192.168.1.50:8081 \
   -X main.embeddedIntervalSeconds=15 \
   -X main.embeddedDeviceType=pc \
   -X main.embeddedServicesToCheck=nginx,postgres"
```

Priority: **Embedded defaults < config.yaml** (config.yaml overrides)

---

### 2. Collector (`collector/`)

**Technology**: Go 1.21+, pgx (PostgreSQL driver), godotenv

**Endpoints**:

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/metrics` | Receive telemetry from agent |
| POST | `/api/asset-info` | Receive hardware inventory + apps + users |
| GET | `/api/servers` | List all registered devices |
| GET | `/api/servers/{id}/metrics?range=1h` | Historical metrics |
| GET | `/api/servers/{id}/status` | Latest metrics + service status |
| GET | `/api/servers/{id}/asset-info` | Current hardware inventory |
| GET | `/api/servers/{id}/applications` | Installed applications |
| GET | `/api/servers/{id}/users` | Local user accounts + privileges |
| PUT | `/api/servers/{id}/metadata` | Update owner, department, location, notes |
| PUT | `/api/servers/{id}/rename` | Rename server_id (cascades to all tables) |

**Features**:
- CORS enabled for dashboard access
- Connection pooling (configurable `DB_MAX_CONNS`, default 25)
- Health check with retry logic
- UPSERT for asset_info (insert or update)
- REPLACE for installed_applications (full refresh per report)
- Cascade rename via `ON UPDATE CASCADE` foreign keys

---

### 3. Dashboard (`dashboard/`)

**Technology**: React 19, Vite, Tailwind CSS v4, Recharts, Lucide React

**Pages**:
- **Daily Ledger** (`/`) — Overview cards, server grid with wax-seal status, real-time metrics bars
- **Estate Inventory** (`/assets`) — Hardware register table with disk health %, SMART status, battery, serial, IP/MAC, CSV export
- **Provisions Register** (`/apps`) — Software catalog per device with search
- **Device Detail** (`/detail/{id}`) — Time-series charts (CPU/RAM, Network), service status, user accounts, device assignment editor, rename flow

**Design System** (Seneschal Estate Ledger):
- Colors: Parchment `#F4F1EA`, Ink `#2C2621`, Gold Leaf `#C5A059`, Forest Moss `#4A5D4E`, Cardinal `#8B1E1E`
- Typography: Fraunces (display), IBM Plex Mono (data)
- Wax-seal status badges: Normal (green), Attention (gold), Critical (red), Offline (gray)

---

## Installation & Deployment

### Prerequisites

- **Server**: Ubuntu 22.04+ with Docker & Docker Compose
- **Clients**: Windows 10/11 / Server 2019+ (Agent)
- **Network**: Clients must reach Collector on port 8081 (HTTP) or 443 (HTTPS via Nginx)

### Server Deployment (Docker Compose)

```yaml
# docker-compose.yml
services:
  postgres:
    image: postgres:16-alpine
    container_name: monitoring-postgres
    restart: unless-stopped
    environment:
      POSTGRES_DB: monitoring_db
      POSTGRES_USER: monitoring_user
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    ports:
      - "5433:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./collector/migrations:/docker-entrypoint-initdb.d
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U monitoring_user -d monitoring_db"]
      interval: 5s
      timeout: 3s
      retries: 10

  collector:
    build: ./collector
    container_name: monitoring-collector
    restart: unless-stopped
    env_file: .env
    environment:
      DB_HOST: postgres
      DB_PORT: 5432
    ports:
      - "${COLLECTOR_PORT:-8081}:8081"
    depends_on:
      postgres:
        condition: service_healthy

volumes:
  postgres_data:
```

```bash
# .env
DB_HOST=localhost
DB_PORT=5433
DB_NAME=monitoring_db
DB_USER=monitoring_user
DB_PASSWORD=change_this_password
DB_MAX_CONNS=25
COLLECTOR_PORT=8081
```

```bash
# Start
docker compose up -d --build

# Logs
docker compose logs -f collector
```

### Client Agent Deployment

#### Option A: Hardened Install (Recommended)

Run **PowerShell as Administrator** on target machine:

```powershell
# Copy install.ps1, agent.exe, config.yaml to target machine
.\install.ps1 -CollectorURL "http://192.168.1.50:8081" `
              -ServerID "PC-HRD-01" `
              -DeviceType pc `
              -IntervalSeconds 15 `
              -ServicesToCheck "nginx,postgres"
```

**What it does**:
1. Creates `C:\ProgramData\Seneschal\`
2. Copies `agent.exe` and generates `config.yaml`
3. **Hardens NTFS ACLs**: SYSTEM/Admins=Full, Users=ReadOnly (no edit/delete)
4. Registers **Windows Service** (`SeneschalAgent`) running as `NT AUTHORITY\SYSTEM`
5. Configures **auto-restart** on failure (5s → 10s → 60s)
6. Starts service immediately

#### Option B: Embedded Config (No config.yaml)

Build binary with embedded defaults:

```powershell
.\build.ps1 -CollectorURL "http://192.168.1.50:8081" `
            -IntervalSeconds 15 `
            -DeviceType pc `
            -ServicesToCheck "nginx,postgres" `
            -OutputPath ".\agent.exe"
```

Deploy `agent.exe` alone — it auto-generates `server_id` from BIOS Serial and uses embedded defaults.

#### Uninstall

```powershell
.\uninstall.ps1          # Remove service + folder
.\uninstall.ps1 -KeepData  # Keep folder, remove service only
```

---

## Agent Hardening & Security

### Threat Model

| Threat | Mitigation |
|--------|------------|
| User edits `config.yaml` to spoof `server_id` | NTFS ACL: Users=ReadOnly; Embedded config fallback; Token auth (planned) |
| User kills agent process | Windows Service (SYSTEM) — user cannot stop; Auto-restart on crash |
| User deletes agent folder | NTFS ACL: Users=ReadOnly; Folder owned by SYSTEM |
| Spoofed metrics from network | Token authentication (planned Phase 6) |
| Data tampering in transit | HTTPS via Nginx reverse proxy (production) |

### NTFS ACL Hardening

```powershell
# Applied by install.ps1 to C:\ProgramData\Seneschal\
SYSTEM          → FullControl (ContainerInherit, ObjectInherit)
Administrators  → FullControl (ContainerInherit, ObjectInherit)
Users           → ReadAndExecute ONLY (no Write, Modify, Delete)
```

### Windows Service Configuration

```ini
Service Name:     SeneschalAgent
Display Name:     Seneschal Monitoring Agent
Account:          NT AUTHORITY\SYSTEM
Startup Type:     Automatic
Recovery Actions: Restart after 5s → 10s → 60s (reset after 24h)
```

### Embedded Config Security

Binary can run **without `config.yaml`**:
- `server_id` auto-generated from BIOS Serial Number (`Win32_BIOS`)
- Collector URL, interval, device type, services embedded at build time
- `config.yaml` still supported for overrides (highest priority)

---

## Dashboard

### Access

```
http://<server-ip>:3000    # Development (Vite dev server)
http://<server-ip>       # Production (Nginx serves built assets)
```

### Navigation

| Tab | Route | Description |
|-----|-------|-------------|
| **Daily Ledger** | `/` | Server overview with status cards |
| **Estate Inventory** | `/assets` | Hardware register table |
| **Provisions Register** | `/apps` | Software catalog |

### Key Features

#### Daily Ledger
- Summary registers: Total, Nominal, Attention, Critical
- Search by server_id, hostname, IP, owner, department
- Filter by device type (server/pc/laptop/all)
- Server cards: hostname, device type icon, metrics bars (CPU/RAM/Disk), wax-seal status

#### Device Detail
- Time-series charts: CPU/RAM (%), Network (KB/s) — 30m/1h/6h/24h ranges
- Service status register (Operational/Halted)
- Local User Accounts table: Username, Full Name, Privilege badge, State (Enabled/Disabled), Session (Active/—)
- Device Assignment editor: Owner, Department, Location, Device Type, Notes
- **Rename Device**: Inline edit `server_id` with validation (cascades to all tables)

#### Estate Inventory
- Columns: Device, Processor & RAM, Disk Health (%), SMART, Battery, Serial/OS, Network
- Disk Health % color-coded: Pristine (green ≥80), Wearing (gold 50-79), Degraded (red <50)
- SMART badge: OK (green) / WARNING (red)
- CSV export

#### Provisions Register
- Searchable table: Holding, Provision, Version, Purveyor, Acquired

---

## API Reference

### Authentication

Currently: **None** (trusted network).  
Planned: Per-agent token (`Authorization: Bearer <token>`) + Dashboard session auth.

### Endpoints

#### POST `/api/metrics`
```json
{
  "server_id": "PC-HRD-01",
  "cpu_percent": 45.2,
  "ram_percent": 67.8,
  "disk_percent": 42.1,
  "network_sent_bps": 102400,
  "network_recv_bps": 204800,
  "service_status": { "nginx": true, "postgres": true },
  "timestamp": "2026-10-01T10:30:00Z"
}
```

#### POST `/api/asset-info`
```json
{
  "server_id": "PC-HRD-01",
  "processor_model": "Intel Core i7-10700",
  "processor_cores": 8,
  "processor_clock_ghz": 2.9,
  "disk_type": "SSD",
  "disk_size_gb": 512,
  "disk_health_status": "OK",
  "disk_health_percent": 98,
  "ram_total_gb": 32,
  "ram_slots_used": 2,
  "ram_slots_total": 4,
  "ram_ddr_type": "DDR4",
  "battery_health_percent": 95.5,
  "serial_number": "SGH731S84K",
  "ip_address": "192.168.1.100",
  "mac_address": "00:1A:2B:3C:4D:5E",
  "os_version": "Windows 10 Pro 10.0.19045",
  "applications": [
    { "app_name": "Google Chrome", "app_version": "118.0.5993.88", "publisher": "Google LLC", "install_date": "2023-10-15" }
  ],
  "device_users": [
    { "username": "IT 10", "full_name": "", "privilege": "administrator", "is_disabled": false, "is_logged_in": true }
  ]
}
```

#### PUT `/api/servers/{id}/metadata`
```json
{
  "device_type": "pc",
  "location": "Ruang IT",
  "owner_name": "Bagues IT",
  "department": "IT",
  "notes": "Primary workstation"
}
```

#### PUT `/api/servers/{id}/rename`
```json
{ "new_server_id": "PC-HRD-02" }
```
Response:
```json
{ "status": "renamed", "old_server_id": "PC-HRD-01", "new_server_id": "PC-HRD-02" }
```

---

## Database Schema

```sql
-- Servers
CREATE TABLE servers (
    id              SERIAL PRIMARY KEY,
    server_id       VARCHAR(50) NOT NULL UNIQUE,
    hostname        VARCHAR(100) NOT NULL,
    ip_address      VARCHAR(45),
    location        VARCHAR(100),
    device_type     VARCHAR(20) DEFAULT 'server',
    owner_name      VARCHAR(100),
    department      VARCHAR(100),
    notes           VARCHAR(255),
    auth_token      VARCHAR(255),
    is_active       BOOLEAN DEFAULT TRUE,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

-- Metrics (time-series)
CREATE TABLE metrics (
    id                  BIGSERIAL PRIMARY KEY,
    server_id           VARCHAR(50) NOT NULL REFERENCES servers(server_id) ON UPDATE CASCADE,
    cpu_percent         NUMERIC(5,2) NOT NULL,
    ram_percent         NUMERIC(5,2) NOT NULL,
    disk_percent        NUMERIC(5,2) NOT NULL,
    network_sent_bps    NUMERIC(12,2) NOT NULL,
    network_recv_bps    NUMERIC(12,2) NOT NULL,
    recorded_at         TIMESTAMPTZ NOT NULL,
    created_at          TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_metrics_server_recorded ON metrics (server_id, recorded_at DESC);

-- Service Status
CREATE TABLE service_status (
    id              BIGSERIAL PRIMARY KEY,
    server_id       VARCHAR(50) NOT NULL REFERENCES servers(server_id) ON UPDATE CASCADE,
    service_name    VARCHAR(100) NOT NULL,
    is_running      BOOLEAN NOT NULL,
    checked_at      TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_service_status_server_checked ON service_status (server_id, checked_at DESC);

-- Asset Info (1 row per device, UPSERT)
CREATE TABLE asset_info (
    id                      SERIAL PRIMARY KEY,
    server_id               VARCHAR(50) NOT NULL UNIQUE REFERENCES servers(server_id) ON UPDATE CASCADE,
    processor_model         VARCHAR(200),
    processor_cores         INT,
    processor_clock_ghz     NUMERIC(5,2),
    disk_type               VARCHAR(10),
    disk_size_gb            NUMERIC(10,2),
    disk_health_status      VARCHAR(20),
    disk_health_percent     INT,
    ram_total_gb            NUMERIC(10,2),
    ram_slots_used          INT,
    ram_slots_total         INT,
    ram_ddr_type            VARCHAR(10),
    battery_health_percent  NUMERIC(5,2),
    serial_number           VARCHAR(100),
    ip_address              VARCHAR(45),
    mac_address             VARCHAR(20),
    os_version              VARCHAR(100),
    updated_at              TIMESTAMPTZ DEFAULT NOW()
);

-- Installed Applications (REPLACE per report)
CREATE TABLE installed_applications (
    id              BIGSERIAL PRIMARY KEY,
    server_id       VARCHAR(50) NOT NULL REFERENCES servers(server_id) ON UPDATE CASCADE,
    app_name        VARCHAR(255) NOT NULL,
    app_version     VARCHAR(100),
    publisher       VARCHAR(255),
    install_date    VARCHAR(20),
    reported_at     TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_installed_apps_server ON installed_applications (server_id);

-- Device Users (REPLACE per report)
CREATE TABLE device_users (
    id              SERIAL PRIMARY KEY,
    server_id       VARCHAR(50) NOT NULL REFERENCES servers(server_id) ON UPDATE CASCADE,
    username        VARCHAR(100) NOT NULL,
    full_name       VARCHAR(150),
    privilege       VARCHAR(30) NOT NULL DEFAULT 'standard_user',
    is_disabled     BOOLEAN DEFAULT FALSE,
    is_logged_in    BOOLEAN DEFAULT FALSE,
    reported_at     TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (server_id, username)
);
CREATE INDEX idx_device_users_server ON device_users (server_id);
```

### Retention Policy

```sql
-- Run daily via cron (02:00)
DELETE FROM metrics WHERE recorded_at < NOW() - INTERVAL '30 days';
DELETE FROM service_status WHERE checked_at < NOW() - INTERVAL '30 days';
VACUUM ANALYZE metrics;
VACUUM ANALYZE service_status;
```

---

## Configuration

### Collector Environment Variables (`.env`)

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_HOST` | `localhost` | PostgreSQL host |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_NAME` | `monitoring_db` | Database name |
| `DB_USER` | `monitoring_user` | Database user |
| `DB_PASSWORD` | required | Database password |
| `DB_MAX_CONNS` | `25` | Connection pool size |
| `COLLECTOR_PORT` | `8081` | API listen port |

### Agent Configuration (`config.yaml`)

| Key | Required | Default | Description |
|-----|----------|---------|-------------|
| `server_id` | Yes* | — | Unique device ID (auto from BIOS if empty) |
| `device_type` | No | `pc` | `server`, `pc`, or `laptop` |
| `collector_url` | Yes* | — | Collector base URL (e.g., `http://192.168.1.50:8081`) |
| `interval_seconds` | No | `15` | Metrics collection interval |
| `services_to_check` | No | `["nginx","postgres"]` | Process names to monitor |

*Required unless provided via embedded defaults at build time.

### Embedded Build Flags

| Flag | Type | Description |
|------|------|-------------|
| `main.embeddedCollectorURL` | string | Collector base URL |
| `main.embeddedIntervalSeconds` | string | Interval in seconds |
| `main.embeddedDeviceType` | string | `server`, `pc`, or `laptop` |
| `main.embeddedServicesToCheck` | string | Comma-separated process names |
| `main.embeddedServerID` | string | Fallback server_id |

---

## Operations & Maintenance

### Daily Operations

- **Dashboard**: Monitor Daily Ledger for Critical/Attention devices
- **Alerts**: Check for Offline devices (no metrics > 15 min)
- **Disk Health**: Review Estate Inventory for Degraded disks (<50%)

### Weekly

- Review disk health trends (SMART %)
- Verify all target devices reporting
- Check service status for critical processes

### Monthly

- Run retention cleanup (cron at 02:00)
- `VACUUM ANALYZE` on metrics tables
- Review agent versions, plan updates

### Backup

```bash
# PostgreSQL dump
docker exec monitoring-postgres pg_dump -U monitoring_user monitoring_db > backup_$(date +%F).sql

# Restore
docker exec -i monitoring-postgres psql -U monitoring_user -d monitoring_db < backup_2026-10-01.sql
```

### Logs

```bash
# Collector logs
docker compose logs -f collector

# Agent (Windows Event Viewer → Applications and Services Logs)
# Or redirect output: agent.exe > C:\ProgramData\Seneschal\agent.log 2>&1
```

---

## Troubleshooting

### Collector Won't Start

| Symptom | Cause | Fix |
|---------|-------|-----|
| `connection refused` to DB | PostgreSQL not ready | Check `docker compose ps`; wait for healthcheck |
| `port 8081 in use` | Another process | `netstat -ano | findstr :8081` → kill or change port |
| `DB_PORT=5433` in container | Wrong port | Container uses `5432`; host maps to `5433` |

### Agent Not Reporting

| Symptom | Cause | Fix |
|---------|-------|-----|
| No metrics in Dashboard | Agent not running | Check service: `sc query SeneschalAgent` |
| `config.yaml` not found | Wrong path | Agent expects config in same folder as exe |
| `collector_url` unreachable | Network/Firewall | Test `curl http://collector:8081/api/servers` from client |
| `server_id` mismatch | Duplicate ID | Each device needs unique `server_id` |

### Device Rename Issues

| Issue | Resolution |
|-------|------------|
| `400 Bad Request` | New ID contains invalid chars (only a-z, A-Z, 0-9, ., _, -) |
| `409 Conflict` | New ID already exists |
| `404 Not Found` | Old server_id doesn't exist |
| Orphan data after rename | Check FK `ON UPDATE CASCADE` on all 5 child tables |

### Disk Health % Not Showing

- Only works on Windows with SMART vendor data available
- Some SSDs don't expose wear level via WMI
- Check agent logs: `Gagal membaca disk health percent`

---

## Development

### Project Structure

```
monitoring-system/
├── agent/                 # Go agent (Windows)
│   ├── main.go           # Entry point, schedulers
│   ├── config.go         # Config loading (file + embedded)
│   ├── asset.go          # Hardware inventory, users, apps
│   ├── metrics.go        # CPU, RAM, Disk, Network, Services
│   ├── config.yaml       # Runtime config
│   ├── config.example.yaml
│   ├── build.ps1         # Build with embedded defaults
│   ├── install.ps1       # Hardened Windows Service installer
│   └── uninstall.ps1     # Clean removal
│
├── collector/            # Go API server
│   ├── main.go           # HTTP server, routes, CORS
│   ├── handler.go        # HTTP handlers
│   ├── db.go             # PostgreSQL operations
│   ├── migrations/       # SQL migrations (001-006)
│   ├── Dockerfile
│   └── go.mod
│
├── dashboard/            # React frontend
│   ├── src/
│   │   ├── components/   # Overview, DeviceDetail, AssetInventory, Applications
│   │   ├── index.css     # Seneschal design system (Tailwind v4)
│   │   ├── App.jsx       # Routing, layout
│   │   └── main.jsx
│   ├── vite.config.js    # Proxy /api → :8081
│   ├── package.json
│   └── index.html
│
├── collector/migrations/ # SQL migrations
│   ├── 001_init_schema.sql
│   ├── 002_add_network_service_metrics.sql
│   ├── 003_add_asset_info.sql
│   ├── 004_add_disk_health_percent.sql
│   ├── 005_add_device_ownership_and_users.sql
│   └── 006_enable_server_id_rename_cascade.sql
│
├── docker-compose.yml
├── .env
└── README.md
```

### Build Commands

```bash
# Agent (Windows)
cd agent
go build -o agent.exe .                    # Standard
.\build.ps1 -CollectorURL "http://..."     # With embedded defaults

# Collector
cd collector
go build -o collector .

# Dashboard
cd dashboard
npm install
npm run dev      # Dev server (port 3000, proxies /api to :8081)
npm run build    # Production build → dist/
```

### Database Migrations

```bash
# Apply manually
docker exec -i monitoring-postgres psql -U monitoring_user -d monitoring_db < collector/migrations/006_*.sql

# Auto-applied on container start (postgres entrypoint)
```

---

## Version History

| Version | Date | Changes |
|---------|------|---------|
| **v2.0** | 2026-10-01 | Agent hardening, embedded config, Windows Service installer, device rename cascade, user privilege detection, disk wear %, Seneschal UI redesign |
| **v1.x** | 2026-09 | Initial agent, collector, dashboard, basic metrics, asset inventory |

---

## License & Credits

**Seneschal** — Internal monitoring system for PT Tata Optima Property.  
Developed by IT Programme Coordinator Bagues Putra Tawaqqal.

**Third-party libraries**:
- Go: pgx, gopsutil, wmi, yaml.v3
- React: Vite, Tailwind CSS v4, Recharts, Lucide React
- Infrastructure: PostgreSQL 16, Nginx, Docker

---

*Document version: 2.0 | Last updated: 2026-10-01*