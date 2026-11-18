package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"strings"
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
	log.Printf("Config loaded - Server ID: %s (%s), Collector: %s, Interval: %ds, AssetInterval: %dh",
		cfg.ServerID, cfg.DeviceType, cfg.CollectorURL, cfg.IntervalSeconds, cfg.AssetIntervalHours)

	tracker := newProcessTracker()
	asset := collectAssetInfo(cfg.ServerID, cfg.DeviceType)
	sendAssetInfo(cfg.CollectorURL+assetInfoPath, asset)

	metricsTicker := time.NewTicker(time.Duration(cfg.IntervalSeconds) * time.Second)
	defer metricsTicker.Stop()
	assetTicker := time.NewTicker(time.Duration(cfg.AssetIntervalHours) * time.Hour)
	defer assetTicker.Stop()

	for {
		select {
		case <-metricsTicker.C:
			payload := collectMetrics(cfg, tracker)
			sendMetrics(cfg.CollectorURL+metricsPath, payload)
		case <-assetTicker.C:
			asset := collectAssetInfo(cfg.ServerID, cfg.DeviceType)
			asset.RunningProcesses = mergeUniqueProcesses(asset.RunningProcesses, tracker.DrainSorted())
			sendAssetInfo(cfg.CollectorURL+assetInfoPath, asset)
		}
	}
}

func mergeUniqueProcesses(a, b []string) []string {
	seen := make(map[string]string, len(a)+len(b))
	for _, list := range [][]string{a, b} {
		for _, p := range list {
			trimmed := strings.TrimSpace(p)
			if trimmed == "" {
				continue
			}
			key := strings.ToLower(trimmed)
			if _, ok := seen[key]; !ok {
				seen[key] = trimmed
			}
		}
	}
	out := make([]string, 0, len(seen))
	for _, v := range seen {
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

func collectMetrics(cfg Config, tracker *processTracker) MetricsPayload {
	sentBps, recvBps := getNetworkUsage()
	return MetricsPayload{
		ServerID:       cfg.ServerID,
		CPU:            getCPU(),
		RAM:            getRAM(),
		Disk:           getDisk(),
		NetworkSentBps: sentBps,
		NetworkRecvBps: recvBps,
		ServiceStatus:  checkServicesAndTrack(cfg.ServicesToCheck, tracker),
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
