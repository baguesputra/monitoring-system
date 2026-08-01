package main

import (
	"encoding/json"
	"log"
	"net/http"
)

// handleGetServers menangani GET /api/servers
func handleGetServers(w http.ResponseWriter, r *http.Request) {
	servers, err := getAllServers()
	if err != nil {
		http.Error(w, "Gagal mengambil data server", http.StatusInternalServerError)
		log.Printf("Gagal query servers: %v", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(servers)
}