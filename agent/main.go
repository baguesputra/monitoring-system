package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

type MetricsPayload struct {
	ServerID       string          `json:"server_id"`
	CPU            float64         `json:"cpu_percent"`
	RAM            float64         `json:"ram_percent"`
	Disk           float64         `json:"disk_percent"`
	NetworkSentBps float64         `json:"network_sent_bps"`
	NetworkRecvBps float64         `json:"network_recv_bps"`
	ServiceStatus  map[string]bool `json:"service_status"`
	Timestamp      time.Time       `json:"timestamp"`
}

func main() {
	log.Println("Monitoring agent starting...")

	cfg := loadConfig("config.yaml")
	log.Printf("Config loaded - Server ID: %s, Collector: %s, Interval: %ds",
		cfg.ServerID, cfg.CollectorURL, cfg.IntervalSeconds)

	ticker := time.NewTicker(time.Duration(cfg.IntervalSeconds) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		payload := collectMetrics(cfg)
		sendMetrics(cfg.CollectorURL, payload)
	}
}

func collectMetrics(cfg Config) MetricsPayload {
	sentBps, recvBps := getNetworkUsage()

	return MetricsPayload{
		ServerID:       cfg.ServerID,
		CPU:            getCPU(),
		RAM:            getRAM(),
		Disk:           getDisk(),
		NetworkSentBps: sentBps,
		NetworkRecvBps: recvBps,
		ServiceStatus:  checkServices(cfg.ServicesToCheck),
		Timestamp:      time.Now(),
	}
}

func sendMetrics(collectorURL string, payload MetricsPayload) {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Gagal encode payload ke JSON: %v", err)
		return
	}

	resp, err := http.Post(collectorURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Gagal mengirim data ke Collector: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		log.Printf("Metrics terkirim - CPU: %.2f%%, RAM: %.2f%%, Disk: %.2f%%, Net: %.0f/%.0f Bps",
			payload.CPU, payload.RAM, payload.Disk, payload.NetworkSentBps, payload.NetworkRecvBps)
	} else {
		log.Printf("Collector merespons dengan status: %d", resp.StatusCode)
	}
}