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

const (
	metricsPath   = "/api/metrics"
	assetInfoPath = "/api/asset-info"
)

func main() {
	log.Println("Monitoring agent starting...")
	cfg := loadConfig("config.yaml")
	log.Printf("Config loaded - Server ID: %s (%s), Collector: %s, Interval: %ds",
		cfg.ServerID, cfg.DeviceType, cfg.CollectorURL, cfg.IntervalSeconds)

	asset := collectAssetInfo(cfg.ServerID, cfg.DeviceType)
	sendAssetInfo(cfg.CollectorURL+assetInfoPath, asset)

	metricsTicker := time.NewTicker(time.Duration(cfg.IntervalSeconds) * time.Second)
	defer metricsTicker.Stop()
	assetTicker := time.NewTicker(24 * time.Hour)
	defer assetTicker.Stop()

	for {
		select {
		case <-metricsTicker.C:
			payload := collectMetrics(cfg)
			sendMetrics(cfg.CollectorURL+metricsPath, payload)
		case <-assetTicker.C:
			asset := collectAssetInfo(cfg.ServerID, cfg.DeviceType)
			sendAssetInfo(cfg.CollectorURL+assetInfoPath, asset)
		}
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

func sendMetrics(url string, payload MetricsPayload) {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Gagal encode payload ke JSON: %v", err)
		return
	}
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
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

func sendAssetInfo(url string, asset AssetInfo) {
	jsonData, err := json.Marshal(asset)
	if err != nil {
		log.Printf("Gagal encode asset info: %v", err)
		return
	}
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Gagal mengirim asset info: %v", err)
		return
	}
	defer resp.Body.Close()
	log.Printf("Asset info terkirim, status: %d", resp.StatusCode)
}
