package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// MetricsPayload merepresentasikan data yang dikirim oleh agent
type MetricsPayload struct {
	ServerID  string    `json:"server_id"`
	CPU       float64   `json:"cpu_percent"`
	RAM       float64   `json:"ram_percent"`
	Disk      float64   `json:"disk_percent"`
	Timestamp time.Time `json:"timestamp"`
}

func main() {
	http.HandleFunc("/api/metrics", handleMetrics)

	log.Println("Collector API starting on :8081...")
	if err := http.ListenAndServe(":8081", nil); err != nil {
		log.Fatalf("Server gagal jalan: %v", err)
	}
}

// handleMetrics menerima data metrics dari agent via POST request
func handleMetrics(w http.ResponseWriter, r *http.Request) {
	// Cuma terima method POST
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload MetricsPayload

	// Decode JSON dari body request ke struct MetricsPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		log.Printf("Gagal parse payload: %v", err)
		return
	}

	// Untuk sekarang, cukup log ke console dulu
	log.Printf("Received metrics from [%s] - CPU: %.2f%%, RAM: %.2f%%, Disk: %.2f%%",
		payload.ServerID, payload.CPU, payload.RAM, payload.Disk)

	// Kirim response sukses ke agent
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"received"}`))
}