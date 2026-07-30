package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

var dbPool *pgxpool.Pool

// initDB membuka connection pool ke PostgreSQL
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

	// Test koneksi beneran nyambung, bukan cuma bikin objek pool-nya doang
	if err := pool.Ping(context.Background()); err != nil {
		log.Fatalf("Gagal ping database: %v", err)
	}

	dbPool = pool
	log.Println("Berhasil terkoneksi ke database")
}

// saveMetrics menyimpan 1 payload metrics ke tabel metrics
func saveMetrics(payload MetricsPayload) error {
	query := `
		INSERT INTO metrics (server_id, cpu_percent, ram_percent, disk_percent, recorded_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := dbPool.Exec(context.Background(), query,
		payload.ServerID, payload.CPU, payload.RAM, payload.Disk, payload.Timestamp,
	)

	return err
}