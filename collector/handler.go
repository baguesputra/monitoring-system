package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
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

	// Validasi: server_id harus sudah terdaftar di tabel servers
	exists, err := serverIDExists(serverID)
	if err != nil {
		http.Error(w, "Gagal memeriksa keberadaan server_id", http.StatusInternalServerError)
		log.Printf("Gagal query serverIDExists: %v", err)
		return
	}
	if !exists {
		http.Error(w, fmt.Sprintf("server_id '%s' tidak terdaftar. Jalankan instalasi agent dulu.", serverID), http.StatusNotFound)
		return
	}

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

// parseRange mengubah string seperti "1h", "30m", "24h", "7d", "30d" jadi time.Duration.
// Default 1 jam kalau parameter kosong.
func parseRange(rangeParam string) (time.Duration, error) {
	if rangeParam == "" {
		return 1 * time.Hour, nil
	}
	if strings.HasSuffix(rangeParam, "d") {
		daysStr := strings.TrimSuffix(rangeParam, "d")
		days, err := strconv.Atoi(daysStr)
		if err != nil || days <= 0 || days > 365 {
			return 0, errors.New("invalid day range")
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	return time.ParseDuration(rangeParam)
}

// handleGetServerStatus menangani GET /api/servers/{id}/status
func handleGetServerStatus(w http.ResponseWriter, r *http.Request) {
	serverID := r.PathValue("id")

	// Validasi: server_id harus sudah terdaftar di tabel servers
	exists, err := serverIDExists(serverID)
	if err != nil {
		http.Error(w, "Gagal memeriksa keberadaan server_id", http.StatusInternalServerError)
		log.Printf("Gagal query serverIDExists: %v", err)
		return
	}
	if !exists {
		http.Error(w, fmt.Sprintf("server_id '%s' tidak terdaftar. Jalankan instalasi agent dulu.", serverID), http.StatusNotFound)
		return
	}

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
	// Validasi: IP address tidak boleh dipakai server_id lain
	if p.IPAddress != "" {
		conflict, err := serverIDForIP(p.IPAddress)
		if err != nil {
			http.Error(w, "Gagal memeriksa konflik IP", http.StatusInternalServerError)
			log.Printf("Gagal query serverIDForIP: %v", err)
			return
		}
		if conflict != "" && conflict != p.ServerID {
			http.Error(w, fmt.Sprintf("IP %s sudah dipakai perangkat '%s'. Pastikan setiap agent berjalan di mesin terpisah.", p.IPAddress, conflict), http.StatusConflict)
			return
		}
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

// serverIDForIP mengembalikan server_id yang sudah menggunakan IP tertentu, atau string kosong jika tidak ada.
func serverIDForIP(ip string) (string, error) {
	var serverID string
	err := dbPool.QueryRow(context.Background(),
		`SELECT server_id FROM servers WHERE ip_address = $1`, ip).Scan(&serverID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return serverID, nil
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

func handleRenameServer(w http.ResponseWriter, r *http.Request) {
	serverID := r.PathValue("id")

	var p struct {
		NewServerID string `json:"new_server_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	newID := strings.TrimSpace(p.NewServerID)
	if newID == "" {
		http.Error(w, "new_server_id tidak boleh kosong", http.StatusBadRequest)
		return
	}
	if len(newID) > 50 {
		http.Error(w, "new_server_id maksimal 50 karakter", http.StatusBadRequest)
		return
	}
	// Izinkan huruf, angka, titik, underscore, dan tanda hubung saja.
	// Karakter lain bisa merusak URL dashboard (/api/servers/{id}/...) dan nama file.
	for _, r := range newID {
		valid := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-'
		if !valid {
			http.Error(w, "new_server_id hanya boleh berisi huruf, angka, titik, underscore, dan tanda hubung", http.StatusBadRequest)
			return
		}
	}
	if newID == serverID {
		http.Error(w, "new_server_id sama dengan server_id sekarang", http.StatusBadRequest)
		return
	}

	if err := renameServerID(serverID, newID); err != nil {
		switch {
		case errors.Is(err, ErrServerIDTaken):
			http.Error(w, "Nama '"+newID+"' sudah dipakai perangkat lain. Pilih nama lain.", http.StatusConflict)
		case errors.Is(err, pgx.ErrNoRows):
			http.Error(w, "Perangkat tidak ditemukan", http.StatusNotFound)
		default:
			http.Error(w, "Gagal mengganti nama perangkat", http.StatusInternalServerError)
			log.Printf("Gagal rename server '%s' -> '%s': %v", serverID, newID, err)
		}
		return
	}

	log.Printf("Perangkat di-rename: '%s' -> '%s' (histori ikut termigrasi)", serverID, newID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":        "renamed",
		"old_server_id": serverID,
		"new_server_id": newID,
	})
}

func handleGetUpgradeInsights(w http.ResponseWriter, r *http.Request) {
	days := 30
	if raw := r.URL.Query().Get("days"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 && parsed <= 365 {
			days = parsed
		}
	}
	resp, err := getUpgradeInsights(days)
	if err != nil {
		http.Error(w, "Gagal menganalisa kebutuhan upgrade", http.StatusInternalServerError)
		log.Printf("Gagal query upgrade insights: %v", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleGetPolicies(w http.ResponseWriter, r *http.Request) {
	policies, err := getSoftwarePolicies()
	if err != nil {
		http.Error(w, "Gagal mengambil daftar policy", http.StatusInternalServerError)
		log.Printf("Gagal query software_policies: %v", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(policies)
}

func handleCreatePolicy(w http.ResponseWriter, r *http.Request) {
	var p SoftwarePolicy
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}
	p.Pattern = strings.TrimSpace(p.Pattern)
	p.Category = strings.TrimSpace(p.Category)
	p.Notes = strings.TrimSpace(p.Notes)
	if p.Pattern == "" || len(p.Pattern) > 150 {
		http.Error(w, "pattern wajib diisi (maks 150 karakter)", http.StatusBadRequest)
		return
	}
	if p.PolicyType != "forbidden" && p.PolicyType != "required" && p.PolicyType != "licensed" {
		http.Error(w, "policy_type harus salah satu dari: forbidden, required, licensed", http.StatusBadRequest)
		return
	}
	if p.LicensedSeats < 0 {
		p.LicensedSeats = 0
	}

	created, err := createSoftwarePolicy(p)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			http.Error(w, "Pattern policy sudah ada", http.StatusConflict)
			return
		}
		http.Error(w, "Gagal menyimpan policy", http.StatusInternalServerError)
		log.Printf("Gagal create policy: %v", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(created)
}

func handleDeletePolicy(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		http.Error(w, "ID policy tidak valid", http.StatusBadRequest)
		return
	}
	if err := deleteSoftwarePolicy(id); err != nil {
		http.Error(w, "Gagal menghapus policy", http.StatusInternalServerError)
		log.Printf("Gagal delete policy %d: %v", id, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"deleted"}`))
}

func handleGetPolicyAudit(w http.ResponseWriter, r *http.Request) {
	resp, err := getPolicyAudit()
	if err != nil {
		http.Error(w, "Gagal menjalankan audit policy", http.StatusInternalServerError)
		log.Printf("Gagal query policy audit: %v", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
