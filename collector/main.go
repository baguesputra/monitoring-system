package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type MetricsPayload struct {
	ServerID  string    `json:"server_id"`
	CPU       float64   `json:"cpu_percent"`
	RAM       float64   `json:"ram_percent"`
	Disk      float64   `json:"disk_percent"`
	Timestamp time.Time `json:"timestamp"`
}

func main() {
	// Load .env dari root project (2 folder di atas collector/)
	if err := godotenv.Load("../.env"); err != nil {
		log.Println("Warning: .env file tidak ditemukan, menggunakan environment variable sistem")
	}

	initDB()

	http.HandleFunc("/api/metrics", handleMetrics)

	port := os.Getenv("COLLECTOR_PORT")
	if port == "" {
		port = "8081"
	}

	log.Printf("Collector API starting on :%s...", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server gagal jalan: %v", err)
	}
}

func handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload MetricsPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		log.Printf("Gagal parse payload: %v", err)
		return
	}

	if err := saveMetrics(payload); err != nil {
		http.Error(w, "Gagal menyimpan data", http.StatusInternalServerError)
		log.Printf("Gagal simpan metrics ke database: %v", err)
		return
	}

	log.Printf("Metrics tersimpan - [%s] CPU: %.2f%%, RAM: %.2f%%, Disk: %.2f%%",
		payload.ServerID, payload.CPU, payload.RAM, payload.Disk)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"received"}`))
}