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
	NetworkSentBps  float64         `json:"network_sent_bps"`
	NetworkRecvBps  float64         `json:"network_recv_bps"`
	ServiceStatus   map[string]bool `json:"service_status"`
	Timestamp time.Time `json:"timestamp"`
}

func main() {
	if err := godotenv.Load("../.env"); err != nil {
		log.Println("Warning: .env file tidak ditemukan, menggunakan environment variable sistem")
	}

	initDB()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/metrics", handleMetrics)
	mux.HandleFunc("GET /api/servers", handleGetServers)
	mux.HandleFunc("GET /api/servers/{id}/metrics", handleGetServerMetrics)
	mux.HandleFunc("GET /api/servers/{id}/status", handleGetServerStatus)

	port := os.Getenv("COLLECTOR_PORT")
	if port == "" {
		port = "8081"
	}

	log.Printf("Collector API starting on :%s...", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
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

	if err := saveServiceStatus(payload.ServerID, payload.ServiceStatus, payload.Timestamp); err != nil {
		log.Printf("Gagal simpan service status: %v", err)
		// tidak return error ke agent, karena metrics utama sudah berhasil tersimpan
	}

	log.Printf("Metrics tersimpan - [%s] CPU: %.2f%%, RAM: %.2f%%, Disk: %.2f%%, Net: %.0f/%.0f Bps, Services: %v",
		payload.ServerID, payload.CPU, payload.RAM, payload.Disk,
		payload.NetworkSentBps, payload.NetworkRecvBps, payload.ServiceStatus)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"received"}`))
}