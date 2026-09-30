package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
)

// handleGetServers menangani GET /api/servers
func handleGetServers(w http.ResponseWriter, r *http.Request) {
	servers, err := getAllServers()
	if err != nil {
		http.Error(w, "Gagal mengambil data server", http.StatusInternalServerError)
		log.Printf("Gagal query servers: %v", err)
		return
	}
	if servers == nil {
		servers = []Server{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(servers)
}

// handleGetServerMetrics menangani GET /api/servers/{id}/metrics?range=1h
func handleGetServerMetrics(w http.ResponseWriter, r *http.Request) {
	serverID := r.PathValue("id")

	// Baca query parameter "range", default 1 jam kalau tidak diisi
	rangeParam := r.URL.Query().Get("range")
	duration, err := parseRange(rangeParam)
	if err != nil {
		http.Error(w, "Parameter range tidak valid (gunakan format seperti 1h, 30m, 24h)", http.StatusBadRequest)
		return
	}

	since := time.Now().Add(-duration)

	records, err := getMetricsHistory(serverID, since)
	if err != nil {
		http.Error(w, "Gagal mengambil data metrics", http.StatusInternalServerError)
		log.Printf("Gagal query metrics history: %v", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(records)
}

// parseRange mengubah string seperti "1h", "30m", "24h" jadi time.Duration.
// Default 1 jam kalau parameter kosong.
func parseRange(rangeParam string) (time.Duration, error) {
	if rangeParam == "" {
		return 1 * time.Hour, nil
	}
	return time.ParseDuration(rangeParam)
}

// handleGetServerStatus menangani GET /api/servers/{id}/status
func handleGetServerStatus(w http.ResponseWriter, r *http.Request) {
	serverID := r.PathValue("id")

	status, err := getLatestServerStatus(serverID)
	if err != nil {
		http.Error(w, "Server tidak ditemukan atau belum ada data", http.StatusNotFound)
		log.Printf("Gagal ambil status server '%s': %v", serverID, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func handleAssetInfo(w http.ResponseWriter, r *http.Request) {
	var p AssetInfoPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}
	if p.ServerID == "" {
		http.Error(w, "server_id tidak boleh kosong", http.StatusBadRequest)
		return
	}
	if err := upsertAssetInfo(p); err != nil {
		http.Error(w, "Gagal menyimpan asset info", http.StatusInternalServerError)
		log.Printf("Gagal upsert asset_info: %v", err)
		return
	}
	if err := replaceInstalledApplications(p.ServerID, p.Applications); err != nil {
		log.Printf("Gagal simpan applications: %v", err)
	}
	if len(p.DeviceUsers) > 0 {
		if err := upsertDeviceUsers(p.ServerID, p.DeviceUsers); err != nil {
			log.Printf("Gagal simpan device users: %v", err)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"received"}`))
}

func handleGetAssetInfo(w http.ResponseWriter, r *http.Request) {
	serverID := r.PathValue("id")
	asset, err := getAssetInfo(serverID)
	if err != nil {
		http.Error(w, "Asset info tidak ditemukan", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(asset)
}

func handleGetApplications(w http.ResponseWriter, r *http.Request) {
	serverID := r.PathValue("id")
	apps, err := getInstalledApplications(serverID)
	if err != nil {
		http.Error(w, "Gagal mengambil data aplikasi", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(apps)
}

func handleGetDeviceUsers(w http.ResponseWriter, r *http.Request) {
	serverID := r.PathValue("id")
	users, err := getDeviceUsers(serverID)
	if err != nil {
		http.Error(w, "Gagal mengambil data user perangkat", http.StatusInternalServerError)
		log.Printf("Gagal query device_users '%s': %v", serverID, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func handleUpdateServerMetadata(w http.ResponseWriter, r *http.Request) {
	serverID := r.PathValue("id")

	var p struct {
		DeviceType string `json:"device_type"`
		Location   string `json:"location"`
		OwnerName  string `json:"owner_name"`
		Department string `json:"department"`
		Notes      string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	if len(p.OwnerName) > 100 || len(p.Department) > 100 || len(p.Location) > 100 || len(p.Notes) > 255 {
		http.Error(w, "Nilai terlalu panjang: owner_name/department/location maksimal 100 karakter, notes maksimal 255", http.StatusBadRequest)
		return
	}
	if p.DeviceType != "" && p.DeviceType != "server" && p.DeviceType != "pc" && p.DeviceType != "laptop" {
		http.Error(w, "device_type harus salah satu dari: server, pc, laptop", http.StatusBadRequest)
		return
	}

	if err := updateServerMetadata(serverID, p.DeviceType, p.Location, p.OwnerName, p.Department, p.Notes); err != nil {
		if err == pgx.ErrNoRows {
			http.Error(w, "Perangkat tidak ditemukan", http.StatusNotFound)
			return
		}
		http.Error(w, "Gagal menyimpan metadata perangkat", http.StatusInternalServerError)
		log.Printf("Gagal update metadata server '%s': %v", serverID, err)
		return
	}

	log.Printf("Metadata perangkat '%s' diperbarui - Owner: %s, Divisi: %s, Lokasi: %s",
		serverID, p.OwnerName, p.Department, p.Location)

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"updated"}`))
}