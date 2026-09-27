package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Server struct {
	ID        int       `json:"id"`
	ServerID  string    `json:"server_id"`
	Hostname  string    `json:"hostname"`
	IPAddress string    `json:"ip_address"`
	Location  string    `json:"location"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

// MetricRecord merepresentasikan 1 baris data historis dari tabel metrics
type MetricRecord struct {
	ServerID       string    `json:"server_id"`
	CPU            float64   `json:"cpu_percent"`
	RAM            float64   `json:"ram_percent"`
	Disk           float64   `json:"disk_percent"`
	NetworkSentBps float64   `json:"network_sent_bps"`
	NetworkRecvBps float64   `json:"network_recv_bps"`
	RecordedAt     time.Time `json:"recorded_at"`
}

// ServerStatus merepresentasikan kondisi terkini 1 server:
// metrics terbaru + status semua service yang dicek
type ServerStatus struct {
	ServerID      string          `json:"server_id"`
	Hostname      string          `json:"hostname"`
	CPU           float64         `json:"cpu_percent"`
	RAM           float64         `json:"ram_percent"`
	Disk          float64         `json:"disk_percent"`
	LastSeenAt    time.Time       `json:"last_seen_at"`
	ServiceStatus map[string]bool `json:"service_status"`
}

var dbPool *pgxpool.Pool

func initDB() {
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbName := os.Getenv("DB_NAME")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")

	connString := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s",
		dbUser, dbPassword, dbHost, dbPort, dbName,
	)

	cfg, err := pgxpool.ParseConfig(connString)
	if err != nil {
		log.Fatalf("Gagal parse connection string: %v", err)
	}
	if v := os.Getenv("DB_MAX_CONNS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.MaxConns = int32(n)
		}
	} else {
		cfg.MaxConns = 25
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		log.Fatalf("Gagal membuat connection pool ke database: %v", err)
	}

	for i := 0; i < 10; i++ {
		if err := pool.Ping(context.Background()); err == nil {
			break
		} else if i == 9 {
			log.Fatalf("Gagal ping database setelah retry: %v", err)
		}
		log.Printf("Database belum siap, retry %d/10: %v", i+1, err)
		time.Sleep(2 * time.Second)
	}

	dbPool = pool
	log.Println("Berhasil terkoneksi ke database")
}

// saveMetrics menyimpan data CPU/RAM/Disk/Network ke tabel metrics
func saveMetrics(payload MetricsPayload) error {
	query := `
		INSERT INTO metrics (server_id, cpu_percent, ram_percent, disk_percent, network_sent_bps, network_recv_bps, recorded_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := dbPool.Exec(context.Background(), query,
		payload.ServerID, payload.CPU, payload.RAM, payload.Disk,
		payload.NetworkSentBps, payload.NetworkRecvBps, payload.Timestamp,
	)

	return err
}

// saveServiceStatus menyimpan status tiap service yang dicek ke tabel service_status
func saveServiceStatus(serverID string, statuses map[string]bool, checkedAt interface{}) error {
	query := `
		INSERT INTO service_status (server_id, service_name, is_running, checked_at)
		VALUES ($1, $2, $3, $4)
	`

	for serviceName, isRunning := range statuses {
		_, err := dbPool.Exec(context.Background(), query,
			serverID, serviceName, isRunning, checkedAt,
		)
		if err != nil {
			return fmt.Errorf("gagal simpan status service '%s': %w", serviceName, err)
		}
	}

	return nil
}

// getAllServers mengambil semua server yang terdaftar
func getAllServers() ([]Server, error) {
	query := `
		SELECT id, server_id, hostname, COALESCE(ip_address, ''), COALESCE(location, ''), is_active, created_at
		FROM servers
		ORDER BY hostname ASC
	`

	rows, err := dbPool.Query(context.Background(), query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var servers []Server
	for rows.Next() {
		var s Server
		if err := rows.Scan(&s.ID, &s.ServerID, &s.Hostname, &s.IPAddress, &s.Location, &s.IsActive, &s.CreatedAt); err != nil {
			return nil, err
		}
		servers = append(servers, s)
	}

	return servers, nil
}

// getMetricsHistory mengambil data metrics 1 server dalam rentang waktu tertentu
func getMetricsHistory(serverID string, since time.Time) ([]MetricRecord, error) {
	query := `
		SELECT server_id, cpu_percent, ram_percent, disk_percent,
		       network_sent_bps, network_recv_bps, recorded_at
		FROM metrics
		WHERE server_id = $1 AND recorded_at >= $2
		ORDER BY recorded_at ASC
	`

	rows, err := dbPool.Query(context.Background(), query, serverID, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []MetricRecord
	for rows.Next() {
		var m MetricRecord
		if err := rows.Scan(&m.ServerID, &m.CPU, &m.RAM, &m.Disk,
			&m.NetworkSentBps, &m.NetworkRecvBps, &m.RecordedAt); err != nil {
			return nil, err
		}
		records = append(records, m)
	}

	return records, nil
}

// getLatestServerStatus mengambil metrics terbaru + service status terbaru 1 server
func getLatestServerStatus(serverID string) (*ServerStatus, error) {
	// Ambil metrics paling baru
	metricQuery := `
		SELECT m.server_id, s.hostname, m.cpu_percent, m.ram_percent, m.disk_percent, m.recorded_at
		FROM metrics m
		JOIN servers s ON s.server_id = m.server_id
		WHERE m.server_id = $1
		ORDER BY m.recorded_at DESC
		LIMIT 1
	`

	var status ServerStatus
	err := dbPool.QueryRow(context.Background(), metricQuery, serverID).Scan(
		&status.ServerID, &status.Hostname, &status.CPU, &status.RAM, &status.Disk, &status.LastSeenAt,
	)
	if err != nil {
		return nil, err
	}

	// Ambil service status terbaru untuk tiap service (distinct per service_name)
	serviceQuery := `
		SELECT DISTINCT ON (service_name) service_name, is_running
		FROM service_status
		WHERE server_id = $1
		ORDER BY service_name, checked_at DESC
	`

	rows, err := dbPool.Query(context.Background(), serviceQuery, serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	status.ServiceStatus = make(map[string]bool)
	for rows.Next() {
		var name string
		var isRunning bool
		if err := rows.Scan(&name, &isRunning); err != nil {
			return nil, err
		}
		status.ServiceStatus[name] = isRunning
	}

	return &status, nil
}

type AssetInfo struct {
	ServerID             string   `json:"server_id"`
	ProcessorModel       string   `json:"processor_model"`
	ProcessorCores       int      `json:"processor_cores"`
	ProcessorClockGHz    float64  `json:"processor_clock_ghz"`
	DiskType             string   `json:"disk_type"`
	DiskSizeGB           float64  `json:"disk_size_gb"`
	DiskHealthStatus     string   `json:"disk_health_status"`
	RAMTotalGB           float64  `json:"ram_total_gb"`
	RAMSlotsUsed         int      `json:"ram_slots_used"`
	RAMSlotsTotal        int      `json:"ram_slots_total"`
	RAMDDRType           string   `json:"ram_ddr_type"`
	BatteryHealthPercent *float64 `json:"battery_health_percent"`
	SerialNumber         string   `json:"serial_number"`
	IPAddress            string   `json:"ip_address"`
	MACAddress           string   `json:"mac_address"`
	OSVersion            string   `json:"os_version"`
	UpdatedAt            time.Time `json:"updated_at"`
}

type AppInfo struct {
	Name        string `json:"app_name"`
	Version     string `json:"app_version"`
	Publisher   string `json:"publisher"`
	InstallDate string `json:"install_date"`
}

type AssetInfoPayload struct {
	ServerID             string    `json:"server_id"`
	ProcessorModel       string    `json:"processor_model"`
	ProcessorCores       int       `json:"processor_cores"`
	ProcessorClockGHz    float64   `json:"processor_clock_ghz"`
	DiskType             string    `json:"disk_type"`
	DiskSizeGB           float64   `json:"disk_size_gb"`
	DiskHealthStatus     string    `json:"disk_health_status"`
	RAMTotalGB           float64   `json:"ram_total_gb"`
	RAMSlotsUsed         int       `json:"ram_slots_used"`
	RAMSlotsTotal        int       `json:"ram_slots_total"`
	RAMDDRType           string    `json:"ram_ddr_type"`
	BatteryHealthPercent *float64  `json:"battery_health_percent"`
	SerialNumber         string    `json:"serial_number"`
	IPAddress            string    `json:"ip_address"`
	MACAddress           string    `json:"mac_address"`
	OSVersion            string    `json:"os_version"`
	Applications         []AppInfo `json:"applications"`
}

type InstalledApp struct {
	AppName     string    `json:"app_name"`
	AppVersion  string    `json:"app_version"`
	Publisher   string    `json:"publisher"`
	InstallDate string    `json:"install_date"`
	ReportedAt  time.Time `json:"reported_at"`
}

func upsertAssetInfo(p AssetInfoPayload) error {
	_, err := dbPool.Exec(context.Background(),
		`INSERT INTO servers (server_id, hostname) VALUES ($1, $1) ON CONFLICT (server_id) DO NOTHING`, p.ServerID)
	if err != nil {
		return err
	}
	query := `
		INSERT INTO asset_info (server_id, processor_model, processor_cores, processor_clock_ghz, disk_type, disk_size_gb, disk_health_status, ram_total_gb, ram_slots_used, ram_slots_total, ram_ddr_type, battery_health_percent, serial_number, ip_address, mac_address, os_version, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,NOW())
		ON CONFLICT (server_id) DO UPDATE SET
			processor_model=EXCLUDED.processor_model, processor_cores=EXCLUDED.processor_cores, processor_clock_ghz=EXCLUDED.processor_clock_ghz,
			disk_type=EXCLUDED.disk_type, disk_size_gb=EXCLUDED.disk_size_gb, disk_health_status=EXCLUDED.disk_health_status,
			ram_total_gb=EXCLUDED.ram_total_gb, ram_slots_used=EXCLUDED.ram_slots_used, ram_slots_total=EXCLUDED.ram_slots_total, ram_ddr_type=EXCLUDED.ram_ddr_type,
			battery_health_percent=EXCLUDED.battery_health_percent, serial_number=EXCLUDED.serial_number, ip_address=EXCLUDED.ip_address, mac_address=EXCLUDED.mac_address,
			os_version=EXCLUDED.os_version, updated_at=NOW()
	`
	_, err = dbPool.Exec(context.Background(), query,
		p.ServerID, p.ProcessorModel, p.ProcessorCores, p.ProcessorClockGHz, p.DiskType, p.DiskSizeGB, p.DiskHealthStatus,
		p.RAMTotalGB, p.RAMSlotsUsed, p.RAMSlotsTotal, p.RAMDDRType, p.BatteryHealthPercent, p.SerialNumber, p.IPAddress, p.MACAddress, p.OSVersion)
	if err != nil {
		return err
	}
	if p.IPAddress != "" {
		_, _ = dbPool.Exec(context.Background(), `UPDATE servers SET ip_address=$1 WHERE server_id=$2`, p.IPAddress, p.ServerID)
	}
	return nil
}

func replaceInstalledApplications(serverID string, apps []AppInfo) error {
	ctx := context.Background()
	tx, err := dbPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `DELETE FROM installed_applications WHERE server_id=$1`, serverID); err != nil {
		return err
	}
	for _, a := range apps {
		if _, err := tx.Exec(ctx,
			`INSERT INTO installed_applications (server_id, app_name, app_version, publisher, install_date, reported_at) VALUES ($1,$2,$3,$4,$5,NOW())`,
			serverID, a.Name, a.Version, a.Publisher, a.InstallDate); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func getAssetInfo(serverID string) (*AssetInfo, error) {
	var a AssetInfo
	err := dbPool.QueryRow(context.Background(), `
		SELECT server_id, COALESCE(processor_model,''), COALESCE(processor_cores,0), COALESCE(processor_clock_ghz,0),
		       COALESCE(disk_type,''), COALESCE(disk_size_gb,0), COALESCE(disk_health_status,''), COALESCE(ram_total_gb,0),
		       COALESCE(ram_slots_used,0), COALESCE(ram_slots_total,0), COALESCE(ram_ddr_type,''), battery_health_percent,
		       COALESCE(serial_number,''), COALESCE(ip_address,''), COALESCE(mac_address,''), COALESCE(os_version,''), updated_at
		FROM asset_info WHERE server_id=$1`, serverID).Scan(
		&a.ServerID, &a.ProcessorModel, &a.ProcessorCores, &a.ProcessorClockGHz, &a.DiskType, &a.DiskSizeGB, &a.DiskHealthStatus,
		&a.RAMTotalGB, &a.RAMSlotsUsed, &a.RAMSlotsTotal, &a.RAMDDRType, &a.BatteryHealthPercent, &a.SerialNumber, &a.IPAddress, &a.MACAddress, &a.OSVersion, &a.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func getInstalledApplications(serverID string) ([]InstalledApp, error) {
	rows, err := dbPool.Query(context.Background(),
		`SELECT app_name, COALESCE(app_version,''), COALESCE(publisher,''), COALESCE(install_date,''), reported_at FROM installed_applications WHERE server_id=$1 ORDER BY app_name ASC`, serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var apps []InstalledApp
	for rows.Next() {
		var a InstalledApp
		if err := rows.Scan(&a.AppName, &a.AppVersion, &a.Publisher, &a.InstallDate, &a.ReportedAt); err != nil {
			return nil, err
		}
		apps = append(apps, a)
	}
	if apps == nil {
		apps = []InstalledApp{}
	}
	return apps, nil
}