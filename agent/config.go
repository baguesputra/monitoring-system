package main

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/yusufpapurcu/wmi"
	"gopkg.in/yaml.v3"
)

// Build-time variables (set via -ldflags)
// Example: -ldflags "-X main.embeddedCollectorURL=http://192.168.1.50:8081 -X main.embeddedIntervalSeconds=15"
var (
	embeddedCollectorURL    string
	embeddedIntervalSeconds string
	embeddedDeviceType      string
	embeddedServicesToCheck string
	embeddedServerID        string
)

type Config struct {
	ServerID        string   `yaml:"server_id"`
	DeviceType      string   `yaml:"device_type"`
	CollectorURL    string   `yaml:"collector_url"`
	IntervalSeconds int      `yaml:"interval_seconds"`
	ServicesToCheck []string `yaml:"services_to_check"`
}

// loadConfig membaca dan parse file config.yaml
// Jika file tidak ditemukan, menggunakan nilai embedded (build-time) sebagai fallback.
// Nilai di config.yaml selalu mengoverride nilai embedded.
func loadConfig(path string) Config {
	var cfg Config

	// 1. Apply embedded defaults first (lowest priority)
	if embeddedCollectorURL != "" {
		cfg.CollectorURL = embeddedCollectorURL
	}
	if embeddedIntervalSeconds != "" {
		if v, err := strconv.Atoi(embeddedIntervalSeconds); err == nil && v > 0 {
			cfg.IntervalSeconds = v
		}
	}
	if embeddedDeviceType != "" {
		cfg.DeviceType = embeddedDeviceType
	}
	if embeddedServicesToCheck != "" {
		cfg.ServicesToCheck = strings.Split(embeddedServicesToCheck, ",")
		for i := range cfg.ServicesToCheck {
			cfg.ServicesToCheck[i] = strings.TrimSpace(cfg.ServicesToCheck[i])
		}
	}
	if embeddedServerID != "" {
		cfg.ServerID = embeddedServerID
	}

	// 2. Try to load and override from config.yaml (highest priority)
	data, err := os.ReadFile(path)
	if err == nil {
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			log.Fatalf("Gagal parse config file: %v", err)
		}
	} else {
		log.Printf("Config file '%s' tidak ditemukan, menggunakan embedded defaults", path)
	}

	// 3. Fallback: auto-generate ServerID dari BIOS jika masih kosong
	if cfg.ServerID == "" {
		if bios, err := getBIOSSerial(); err == nil && bios != "" {
			cfg.ServerID = bios
			log.Printf("ServerID di-generate dari BIOS: %s", cfg.ServerID)
		} else {
			cfg.ServerID = getHostname()
			log.Printf("ServerID fallback ke hostname: %s", cfg.ServerID)
		}
	}

	// 4. Validation & defaults
	if cfg.CollectorURL == "" {
		log.Fatal("collector_url tidak boleh kosong (set via config.yaml atau -ldflags embeddedCollectorURL)")
	}
	cfg.CollectorURL = strings.TrimRight(cfg.CollectorURL, "/")

	validTypes := map[string]bool{"server": true, "pc": true, "laptop": true}
	if !validTypes[cfg.DeviceType] {
		cfg.DeviceType = "pc" // default
	}
	if cfg.IntervalSeconds <= 0 {
		cfg.IntervalSeconds = 15
	}
	if len(cfg.ServicesToCheck) == 0 {
		cfg.ServicesToCheck = []string{"nginx", "postgres"}
	}

	log.Printf("Config loaded - ServerID: %s (%s), Collector: %s, Interval: %ds, Services: %v",
		cfg.ServerID, cfg.DeviceType, cfg.CollectorURL, cfg.IntervalSeconds, cfg.ServicesToCheck)

	return cfg
}

// getBIOSSerial reads BIOS serial number via WMI
func getBIOSSerial() (string, error) {
	var bios []struct{ SerialNumber string }
	if err := wmi.Query("SELECT SerialNumber FROM Win32_BIOS", &bios); err != nil {
		return "", err
	}
	if len(bios) > 0 {
		return strings.TrimSpace(bios[0].SerialNumber), nil
	}
	return "", nil
}

// getHostname returns the computer name
func getHostname() string {
	if hn, err := os.Hostname(); err == nil {
		return hn
	}
	return "unknown-host"
}