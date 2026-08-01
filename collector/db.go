package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

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