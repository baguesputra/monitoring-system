package main

import (
	"context"
	"fmt"
	"log"
	"os"
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

	pool, err := pgxpool.New(context.Background(), connString)
	if err != nil {
		log.Fatalf("Gagal membuat connection pool ke database: %v", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		log.Fatalf("Gagal ping database: %v", err)
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