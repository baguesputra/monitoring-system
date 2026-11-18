package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Server struct {
	ID         int       `json:"id"`
	ServerID   string    `json:"server_id"`
	Hostname   string    `json:"hostname"`
	IPAddress  string    `json:"ip_address"`
	Location   string    `json:"location"`
	DeviceType string    `json:"device_type"`
	OwnerName  string    `json:"owner_name"`
	Department string    `json:"department"`
	Notes      string    `json:"notes"`
	IsActive   bool      `json:"is_active"`
	CreatedAt  time.Time `json:"created_at"`
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

// ErrServerIDTaken dikembalikan saat server_id baru sudah dipakai perangkat lain.
var ErrServerIDTaken = errors.New("server_id sudah dipakai perangkat lain")

// serverIDExists mengembalikan true jika server_id ada di tabel servers.
func serverIDExists(serverID string) (bool, error) {
	var exists bool
	err := dbPool.QueryRow(context.Background(),
		`SELECT EXISTS(SELECT 1 FROM servers WHERE server_id = $1)`, serverID).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

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

	if err := ensureSchema(); err != nil {
		log.Fatalf("Gagal memastikan skema database: %v", err)
	}
}

// ensureSchema menjalankan migrasi tambahan secara idempoten agar database lama
// tetap kompatibel saat collector diperbarui.
func ensureSchema() error {
	ctx := context.Background()
	statements := []string{
		`ALTER TABLE asset_info
			ADD COLUMN IF NOT EXISTS device_manufacturer VARCHAR(100),
			ADD COLUMN IF NOT EXISTS device_model VARCHAR(150),
			ADD COLUMN IF NOT EXISTS bios_release_date VARCHAR(20),
			ADD COLUMN IF NOT EXISTS gpu_model VARCHAR(200),
			ADD COLUMN IF NOT EXISTS ram_speed_mhz INT,
			ADD COLUMN IF NOT EXISTS ram_max_gb NUMERIC(10,2),
			ADD COLUMN IF NOT EXISTS ram_form_factor VARCHAR(30),
			ADD COLUMN IF NOT EXISTS disk_volumes JSONB NOT NULL DEFAULT '[]'::jsonb,
			ADD COLUMN IF NOT EXISTS licenses JSONB NOT NULL DEFAULT '[]'::jsonb,
			ADD COLUMN IF NOT EXISTS security_status JSONB NOT NULL DEFAULT '{}'::jsonb,
			ADD COLUMN IF NOT EXISTS running_processes JSONB NOT NULL DEFAULT '[]'::jsonb`,
		`ALTER TABLE installed_applications
			ADD COLUMN IF NOT EXISTS install_scope VARCHAR(20) NOT NULL DEFAULT 'system'`,
		`CREATE TABLE IF NOT EXISTS software_policies (
			id              SERIAL PRIMARY KEY,
			pattern         VARCHAR(150) NOT NULL UNIQUE,
			policy_type     VARCHAR(20) NOT NULL,
			category        VARCHAR(50) NOT NULL DEFAULT '',
			licensed_seats  INT NOT NULL DEFAULT 0,
			notes           VARCHAR(255) NOT NULL DEFAULT '',
			created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`INSERT INTO software_policies (pattern, policy_type, category, licensed_seats, notes)
		VALUES
			('KMSpico', 'forbidden', 'Crack / Activator', 0, 'Activator bajakan Windows/Office'),
			('AutoKMS', 'forbidden', 'Crack / Activator', 0, 'Activator bajakan Windows/Office'),
			('KMSAuto', 'forbidden', 'Crack / Activator', 0, 'Activator bajakan Windows/Office'),
			('uTorrent', 'forbidden', 'P2P / Torrent', 0, 'Aplikasi torrent berisiko malware & boros bandwidth'),
			('BitTorrent', 'forbidden', 'P2P / Torrent', 0, 'Aplikasi torrent tidak diizinkan policy kantor'),
			('qBittorrent', 'forbidden', 'P2P / Torrent', 0, 'Aplikasi torrent tidak diizinkan policy kantor'),
			('TLauncher', 'forbidden', 'Game / Non-Work', 0, 'Game launcher tidak resmi'),
			('Cheat Engine', 'forbidden', 'Crack / Utility', 0, 'Tool modifikasi memori / game')
		ON CONFLICT (pattern) DO NOTHING`,
	}
	for _, stmt := range statements {
		if _, err := dbPool.Exec(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
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
		SELECT id, server_id, hostname, COALESCE(ip_address, ''), COALESCE(location, ''),
		       COALESCE(device_type, 'server'), COALESCE(owner_name, ''), COALESCE(department, ''), COALESCE(notes, ''),
		       is_active, created_at
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
		if err := rows.Scan(&s.ID, &s.ServerID, &s.Hostname, &s.IPAddress, &s.Location,
			&s.DeviceType, &s.OwnerName, &s.Department, &s.Notes, &s.IsActive, &s.CreatedAt); err != nil {
			return nil, err
		}
		servers = append(servers, s)
	}

	return servers, nil
}

// updateServerMetadata menyimpan label perangkat yang bisa diedit dari dashboard
func updateServerMetadata(serverID, deviceType, location, ownerName, department, notes string) error {
	query := `
		UPDATE servers SET
			device_type = COALESCE(NULLIF($2, ''), device_type),
			location    = $3,
			owner_name  = $4,
			department  = $5,
			notes       = $6
		WHERE server_id = $1
	`
	tag, err := dbPool.Exec(context.Background(), query, serverID, deviceType, location, ownerName, department, notes)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// renameServerID mengganti server_id danopters cascade ke seluruh tabel anak
// (metrics, service_status, asset_info, installed_applications, device_users).
// Memerlukan FK dengan ON UPDATE CASCADE — lihat migrasi 006.
func renameServerID(oldID, newID string) error {
	ctx := context.Background()
	tx, err := dbPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var taken bool
	if err := tx.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM servers WHERE server_id = $1)`, newID).Scan(&taken); err != nil {
		return err
	}
	if taken {
		return ErrServerIDTaken
	}

	tag, err := tx.Exec(ctx, `UPDATE servers SET server_id = $1 WHERE server_id = $2`, newID, oldID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	// hostname biasanya masih server_id lama; ikut samarkan supaya tidak membingungkan
	if _, err := tx.Exec(ctx,
		`UPDATE servers SET hostname = $1 WHERE server_id = $2 AND hostname = $3`, newID, newID, oldID); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// getMetricsHistory mengambil data metrics 1 server dalam rentang waktu tertentu.
// Untuk rentang > 25 jam (misal 7d / 30d), data diagregasi per jam agar grafik tetap ringan.
func getMetricsHistory(serverID string, since time.Time) ([]MetricRecord, error) {
	var query string
	if time.Since(since) > 25*time.Hour {
		query = `
			SELECT server_id,
			       ROUND(AVG(cpu_percent)::numeric, 2)::float8,
			       ROUND(AVG(ram_percent)::numeric, 2)::float8,
			       ROUND(AVG(disk_percent)::numeric, 2)::float8,
			       ROUND(AVG(network_sent_bps)::numeric, 2)::float8,
			       ROUND(AVG(network_recv_bps)::numeric, 2)::float8,
			       date_trunc('hour', recorded_at) AS bucket
			FROM metrics
			WHERE server_id = $1 AND recorded_at >= $2
			GROUP BY server_id, bucket
			ORDER BY bucket ASC
		`
	} else {
		query = `
			SELECT server_id, cpu_percent, ram_percent, disk_percent,
			       network_sent_bps, network_recv_bps, recorded_at
			FROM metrics
			WHERE server_id = $1 AND recorded_at >= $2
			ORDER BY recorded_at ASC
		`
	}

	rows, err := dbPool.Query(context.Background(), query, serverID, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := []MetricRecord{}
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

type DiskVolume struct {
	Drive       string  `json:"drive"`
	TotalGB     float64 `json:"total_gb"`
	UsedGB      float64 `json:"used_gb"`
	UsedPercent float64 `json:"used_percent"`
	FileSystem  string  `json:"file_system"`
}

type LicenseInfo struct {
	ProductName   string `json:"product_name"`
	Family        string `json:"family"`
	Channel       string `json:"channel"`
	LicenseStatus string `json:"license_status"`
	PartialKey    string `json:"partial_key"`
	KMSMachine    string `json:"kms_machine"`
}

type SecurityStatus struct {
	AntivirusEnabled   bool     `json:"antivirus_enabled"`
	RealtimeProtection bool     `json:"realtime_protection"`
	AntivirusNames     []string `json:"antivirus_names"`
	FirewallEnabled    bool     `json:"firewall_enabled"`
	Reported           bool     `json:"reported"`
}

type AssetInfo struct {
	ServerID             string         `json:"server_id"`
	DeviceManufacturer   string         `json:"device_manufacturer"`
	DeviceModel          string         `json:"device_model"`
	BIOSReleaseDate      string         `json:"bios_release_date"`
	GPUModel             string         `json:"gpu_model"`
	ProcessorModel       string         `json:"processor_model"`
	ProcessorCores       int            `json:"processor_cores"`
	ProcessorClockGHz    float64        `json:"processor_clock_ghz"`
	DiskType             string         `json:"disk_type"`
	DiskSizeGB           float64        `json:"disk_size_gb"`
	DiskHealthStatus     string         `json:"disk_health_status"`
	DiskHealthPercent    *int           `json:"disk_health_percent"`
	DiskVolumes          []DiskVolume   `json:"disk_volumes"`
	RAMTotalGB           float64        `json:"ram_total_gb"`
	RAMSlotsUsed         int            `json:"ram_slots_used"`
	RAMSlotsTotal        int            `json:"ram_slots_total"`
	RAMDDRType           string         `json:"ram_ddr_type"`
	RAMSpeedMHz          int            `json:"ram_speed_mhz"`
	RAMMaxGB             float64        `json:"ram_max_gb"`
	RAMFormFactor        string         `json:"ram_form_factor"`
	BatteryHealthPercent *float64       `json:"battery_health_percent"`
	SerialNumber         string         `json:"serial_number"`
	IPAddress            string         `json:"ip_address"`
	MACAddress           string         `json:"mac_address"`
	OSVersion            string         `json:"os_version"`
	Licenses             []LicenseInfo  `json:"licenses"`
	SecurityStatus       SecurityStatus `json:"security_status"`
	RunningProcesses     []string       `json:"running_processes"`
	UpdatedAt            time.Time      `json:"updated_at"`
}

type AppInfo struct {
	Name         string `json:"app_name"`
	Version      string `json:"app_version"`
	Publisher    string `json:"publisher"`
	InstallDate  string `json:"install_date"`
	InstallScope string `json:"install_scope"`
}

type AssetInfoPayload struct {
	ServerID             string         `json:"server_id"`
	DeviceManufacturer   string         `json:"device_manufacturer"`
	DeviceModel          string         `json:"device_model"`
	BIOSReleaseDate      string         `json:"bios_release_date"`
	GPUModel             string         `json:"gpu_model"`
	ProcessorModel       string         `json:"processor_model"`
	ProcessorCores       int            `json:"processor_cores"`
	ProcessorClockGHz    float64        `json:"processor_clock_ghz"`
	DiskType             string         `json:"disk_type"`
	DiskSizeGB           float64        `json:"disk_size_gb"`
	DiskHealthStatus     string         `json:"disk_health_status"`
	DiskHealthPercent    *int           `json:"disk_health_percent"`
	DiskVolumes          []DiskVolume   `json:"disk_volumes"`
	RAMTotalGB           float64        `json:"ram_total_gb"`
	RAMSlotsUsed         int            `json:"ram_slots_used"`
	RAMSlotsTotal        int            `json:"ram_slots_total"`
	RAMDDRType           string         `json:"ram_ddr_type"`
	RAMSpeedMHz          int            `json:"ram_speed_mhz"`
	RAMMaxGB             float64        `json:"ram_max_gb"`
	RAMFormFactor        string         `json:"ram_form_factor"`
	BatteryHealthPercent *float64       `json:"battery_health_percent"`
	SerialNumber         string         `json:"serial_number"`
	IPAddress            string         `json:"ip_address"`
	MACAddress           string         `json:"mac_address"`
	OSVersion            string         `json:"os_version"`
	Licenses             []LicenseInfo  `json:"licenses"`
	SecurityStatus       SecurityStatus `json:"security_status"`
	RunningProcesses     []string       `json:"running_processes"`
	Applications         []AppInfo      `json:"applications"`
	DeviceUsers          []DeviceUser   `json:"device_users"`
}

type InstalledApp struct {
	AppName      string    `json:"app_name"`
	AppVersion   string    `json:"app_version"`
	Publisher    string    `json:"publisher"`
	InstallDate  string    `json:"install_date"`
	InstallScope string    `json:"install_scope"`
	ReportedAt   time.Time `json:"reported_at"`
}

func marshalJSONOrFallback(v interface{}, fallback string) string {
	b, err := json.Marshal(v)
	if err != nil || string(b) == "null" {
		return fallback
	}
	return string(b)
}

func upsertAssetInfo(p AssetInfoPayload) error {
	_, err := dbPool.Exec(context.Background(),
		`INSERT INTO servers (server_id, hostname) VALUES ($1, $1) ON CONFLICT (server_id) DO NOTHING`, p.ServerID)
	if err != nil {
		return err
	}

	volumesJSON := marshalJSONOrFallback(p.DiskVolumes, "[]")
	licensesJSON := marshalJSONOrFallback(p.Licenses, "[]")
	securityJSON := marshalJSONOrFallback(p.SecurityStatus, "{}")
	processesJSON := marshalJSONOrFallback(p.RunningProcesses, "[]")

	query := `
		INSERT INTO asset_info (
			server_id, device_manufacturer, device_model, bios_release_date, gpu_model,
			processor_model, processor_cores, processor_clock_ghz,
			disk_type, disk_size_gb, disk_health_status, disk_health_percent, disk_volumes,
			ram_total_gb, ram_slots_used, ram_slots_total, ram_ddr_type, ram_speed_mhz, ram_max_gb, ram_form_factor,
			battery_health_percent, serial_number, ip_address, mac_address, os_version,
			licenses, security_status, running_processes, updated_at
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13::jsonb,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26::jsonb,$27::jsonb,$28::jsonb,NOW())
		ON CONFLICT (server_id) DO UPDATE SET
			device_manufacturer=EXCLUDED.device_manufacturer, device_model=EXCLUDED.device_model,
			bios_release_date=EXCLUDED.bios_release_date, gpu_model=EXCLUDED.gpu_model,
			processor_model=EXCLUDED.processor_model, processor_cores=EXCLUDED.processor_cores, processor_clock_ghz=EXCLUDED.processor_clock_ghz,
			disk_type=EXCLUDED.disk_type, disk_size_gb=EXCLUDED.disk_size_gb, disk_health_status=EXCLUDED.disk_health_status,
			disk_health_percent=EXCLUDED.disk_health_percent, disk_volumes=EXCLUDED.disk_volumes,
			ram_total_gb=EXCLUDED.ram_total_gb, ram_slots_used=EXCLUDED.ram_slots_used, ram_slots_total=EXCLUDED.ram_slots_total,
			ram_ddr_type=EXCLUDED.ram_ddr_type, ram_speed_mhz=EXCLUDED.ram_speed_mhz, ram_max_gb=EXCLUDED.ram_max_gb, ram_form_factor=EXCLUDED.ram_form_factor,
			battery_health_percent=EXCLUDED.battery_health_percent, serial_number=EXCLUDED.serial_number,
			ip_address=EXCLUDED.ip_address, mac_address=EXCLUDED.mac_address, os_version=EXCLUDED.os_version,
			licenses=EXCLUDED.licenses, security_status=EXCLUDED.security_status, running_processes=EXCLUDED.running_processes,
			updated_at=NOW()
	`
	_, err = dbPool.Exec(context.Background(), query,
		p.ServerID, p.DeviceManufacturer, p.DeviceModel, p.BIOSReleaseDate, p.GPUModel,
		p.ProcessorModel, p.ProcessorCores, p.ProcessorClockGHz,
		p.DiskType, p.DiskSizeGB, p.DiskHealthStatus, p.DiskHealthPercent, volumesJSON,
		p.RAMTotalGB, p.RAMSlotsUsed, p.RAMSlotsTotal, p.RAMDDRType, p.RAMSpeedMHz, p.RAMMaxGB, p.RAMFormFactor,
		p.BatteryHealthPercent, p.SerialNumber, p.IPAddress, p.MACAddress, p.OSVersion,
		licensesJSON, securityJSON, processesJSON)
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
	if len(apps) > 0 {
		batch := &pgx.Batch{}
		for _, a := range apps {
			scope := a.InstallScope
			if scope == "" {
				scope = "system"
			}
			batch.Queue(
				`INSERT INTO installed_applications (server_id, app_name, app_version, publisher, install_date, install_scope, reported_at) VALUES ($1,$2,$3,$4,$5,$6,NOW())`,
				serverID, a.Name, a.Version, a.Publisher, a.InstallDate, scope,
			)
		}
		br := tx.SendBatch(ctx, batch)
		for range apps {
			if _, err := br.Exec(); err != nil {
				br.Close()
				return err
			}
		}
		if err := br.Close(); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func scanAssetInfoRow(row pgx.Row) (*AssetInfo, error) {
	var a AssetInfo
	var volumesRaw, licensesRaw, securityRaw, processesRaw []byte
	err := row.Scan(
		&a.ServerID, &a.DeviceManufacturer, &a.DeviceModel, &a.BIOSReleaseDate, &a.GPUModel,
		&a.ProcessorModel, &a.ProcessorCores, &a.ProcessorClockGHz,
		&a.DiskType, &a.DiskSizeGB, &a.DiskHealthStatus, &a.DiskHealthPercent, &volumesRaw,
		&a.RAMTotalGB, &a.RAMSlotsUsed, &a.RAMSlotsTotal, &a.RAMDDRType, &a.RAMSpeedMHz, &a.RAMMaxGB, &a.RAMFormFactor,
		&a.BatteryHealthPercent, &a.SerialNumber, &a.IPAddress, &a.MACAddress, &a.OSVersion,
		&licensesRaw, &securityRaw, &processesRaw, &a.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(volumesRaw, &a.DiskVolumes)
	_ = json.Unmarshal(licensesRaw, &a.Licenses)
	_ = json.Unmarshal(securityRaw, &a.SecurityStatus)
	_ = json.Unmarshal(processesRaw, &a.RunningProcesses)
	if a.DiskVolumes == nil {
		a.DiskVolumes = []DiskVolume{}
	}
	if a.Licenses == nil {
		a.Licenses = []LicenseInfo{}
	}
	if a.RunningProcesses == nil {
		a.RunningProcesses = []string{}
	}
	if a.SecurityStatus.AntivirusNames == nil {
		a.SecurityStatus.AntivirusNames = []string{}
	}
	return &a, nil
}

func getAssetInfo(serverID string) (*AssetInfo, error) {
	row := dbPool.QueryRow(context.Background(), `
		SELECT server_id,
		       COALESCE(device_manufacturer,''), COALESCE(device_model,''), COALESCE(bios_release_date,''), COALESCE(gpu_model,''),
		       COALESCE(processor_model,''), COALESCE(processor_cores,0), COALESCE(processor_clock_ghz,0),
		       COALESCE(disk_type,''), COALESCE(disk_size_gb,0), COALESCE(disk_health_status,''), disk_health_percent, COALESCE(disk_volumes,'[]'::jsonb),
		       COALESCE(ram_total_gb,0), COALESCE(ram_slots_used,0), COALESCE(ram_slots_total,0), COALESCE(ram_ddr_type,''),
		       COALESCE(ram_speed_mhz,0), COALESCE(ram_max_gb,0), COALESCE(ram_form_factor,''),
		       battery_health_percent,
		       COALESCE(serial_number,''), COALESCE(ip_address,''), COALESCE(mac_address,''), COALESCE(os_version,''),
		       COALESCE(licenses,'[]'::jsonb), COALESCE(security_status,'{}'::jsonb), COALESCE(running_processes,'[]'::jsonb), updated_at
		FROM asset_info WHERE server_id=$1`, serverID)
	return scanAssetInfoRow(row)
}

func getInstalledApplications(serverID string) ([]InstalledApp, error) {
	rows, err := dbPool.Query(context.Background(),
		`SELECT app_name, COALESCE(app_version,''), COALESCE(publisher,''), COALESCE(install_date,''), COALESCE(install_scope,'system'), reported_at
		 FROM installed_applications WHERE server_id=$1 ORDER BY app_name ASC`, serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var apps []InstalledApp
	for rows.Next() {
		var a InstalledApp
		if err := rows.Scan(&a.AppName, &a.AppVersion, &a.Publisher, &a.InstallDate, &a.InstallScope, &a.ReportedAt); err != nil {
			return nil, err
		}
		apps = append(apps, a)
	}
	if apps == nil {
		apps = []InstalledApp{}
	}
	return apps, nil
}

type DeviceUser struct {
	ServerID   string    `json:"server_id"`
	Username   string    `json:"username"`
	FullName   string    `json:"full_name"`
	Privilege  string    `json:"privilege"`
	IsDisabled bool      `json:"is_disabled"`
	IsLoggedIn bool      `json:"is_logged_in"`
	ReportedAt time.Time `json:"reported_at"`
}

func upsertDeviceUsers(serverID string, users []DeviceUser) error {
	ctx := context.Background()
	tx, err := dbPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `DELETE FROM device_users WHERE server_id=$1`, serverID); err != nil {
		return err
	}
	for _, u := range users {
		if _, err := tx.Exec(ctx,
			`INSERT INTO device_users (server_id, username, full_name, privilege, is_disabled, is_logged_in, reported_at) VALUES ($1,$2,$3,$4,$5,$6,NOW())`,
			serverID, u.Username, u.FullName, u.Privilege, u.IsDisabled, u.IsLoggedIn); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func getDeviceUsers(serverID string) ([]DeviceUser, error) {
	rows, err := dbPool.Query(context.Background(),
		`SELECT server_id, username, COALESCE(full_name,''), privilege, is_disabled, is_logged_in, reported_at FROM device_users WHERE server_id=$1 ORDER BY privilege DESC, username ASC`, serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []DeviceUser
	for rows.Next() {
		var u DeviceUser
		if err := rows.Scan(&u.ServerID, &u.Username, &u.FullName, &u.Privilege, &u.IsDisabled, &u.IsLoggedIn, &u.ReportedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	if users == nil {
		users = []DeviceUser{}
	}
	return users, nil
}
