package main

import (
	"log"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	ServerID        string   `yaml:"server_id"`
	DeviceType      string   `yaml:"device_type"`
	CollectorURL    string   `yaml:"collector_url"`
	IntervalSeconds int      `yaml:"interval_seconds"`
	ServicesToCheck []string `yaml:"services_to_check"`
}

// loadConfig membaca dan parse file config.yaml
func loadConfig(path string) Config {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("Gagal membaca config file '%s': %v", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		log.Fatalf("Gagal parse config file: %v", err)
	}

	if cfg.ServerID == "" {
		log.Fatal("config.yaml: server_id tidak boleh kosong")
	}
	if cfg.CollectorURL == "" {
		log.Fatal("config.yaml: collector_url tidak boleh kosong")
	}
	cfg.CollectorURL = strings.TrimRight(cfg.CollectorURL, "/")
	validTypes := map[string]bool{"server": true, "pc": true, "laptop": true}
	if !validTypes[cfg.DeviceType] {
		log.Fatalf("config.yaml: device_type harus salah satu dari: server, pc, laptop (didapat: '%s')", cfg.DeviceType)
	}
	if cfg.IntervalSeconds <= 0 {
		cfg.IntervalSeconds = 5
	}
	return cfg
}