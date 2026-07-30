package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
)

type MetricsPayload struct {
	ServerID  string    `json:"server_id"`
	CPU       float64   `json:"cpu_percent"`
	RAM       float64   `json:"ram_percent"`
	Disk      float64   `json:"disk_percent"`
	Timestamp time.Time `json:"timestamp"`
}

func main() {
	log.Println("Monitoring agent starting...")

	cfg := loadConfig("config.yaml")
	log.Printf("Config loaded - Server ID: %s, Collector: %s, Interval: %ds",
		cfg.ServerID, cfg.CollectorURL, cfg.IntervalSeconds)

	ticker := time.NewTicker(time.Duration(cfg.IntervalSeconds) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		payload := collectMetrics(cfg.ServerID)
		sendMetrics(cfg.CollectorURL, payload)
	}
}

func collectMetrics(serverID string) MetricsPayload {
	return MetricsPayload{
		ServerID:  serverID,
		CPU:       getCPU(),
		RAM:       getRAM(),
		Disk:      getDisk(),
		Timestamp: time.Now(),
	}
}

func getCPU() float64 {
	percentages, err := cpu.Percent(1*time.Second, false)
	if err != nil {
		log.Printf("Gagal membaca CPU usage: %v", err)
		return 0
	}
	if len(percentages) > 0 {
		return percentages[0]
	}
	return 0
}

func getRAM() float64 {
	vmStat, err := mem.VirtualMemory()
	if err != nil {
		log.Printf("Gagal membaca RAM usage: %v", err)
		return 0
	}
	return vmStat.UsedPercent
}

func getDisk() float64 {
	diskStat, err := disk.Usage("C:\\")
	if err != nil {
		log.Printf("Gagal membaca Disk usage: %v", err)
		return 0
	}
	return diskStat.UsedPercent
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
		log.Printf("Metrics terkirim - CPU: %.2f%%, RAM: %.2f%%, Disk: %.2f%%",
			payload.CPU, payload.RAM, payload.Disk)
	} else {
		log.Printf("Collector merespons dengan status: %d", resp.StatusCode)
	}
}