package main

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

// Config merepresentasikan struktur file config.yaml
type Config struct {
	ServerID         string `yaml:"server_id"`
	CollectorURL     string `yaml:"collector_url"`
	IntervalSeconds  int    `yaml:"interval_seconds"`
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

	// Validasi sederhana, pastikan field penting gak kosong
	if cfg.ServerID == "" {
		log.Fatal("config.yaml: server_id tidak boleh kosong")
	}
	if cfg.CollectorURL == "" {
		log.Fatal("config.yaml: collector_url tidak boleh kosong")
	}
	if cfg.IntervalSeconds <= 0 {
		cfg.IntervalSeconds = 5 // default fallback
	}

	return cfg
}