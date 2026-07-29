# Monitoring System - Amanah Medical Centre

Sistem monitoring infrastruktur server internal untuk memantau kondisi
CPU, RAM, Disk, Network, dan status service di seluruh server RS secara real-time.

## Arsitektur

- **agent/** — Go, dijalankan di tiap server sebagai Windows Service, membaca metrics sistem
- **collector/** — Go, REST API yang menerima data dari agent dan menyimpan ke database
- **dashboard/** — React, antarmuka web untuk visualisasi data

## Status Development

🚧 Dalam pengembangan — lihat [docs/PRD.md](docs/PRD.md) untuk detail requirement dan roadmap.

## Tech Stack

- Agent & Collector: Go
- Database: SQL Server
- Dashboard: React
- Deployment: Docker Compose (collector + dashboard), Windows Service (agent)