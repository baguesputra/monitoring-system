//go:build !windows

package main

type AssetInfo struct {
	ServerID             string    `json:"server_id"`
	ProcessorModel       string    `json:"processor_model"`
	ProcessorCores       int       `json:"processor_cores"`
	ProcessorClockGHz    float64   `json:"processor_clock_ghz"`
	DiskType             string    `json:"disk_type"`
	DiskSizeGB           float64   `json:"disk_size_gb"`
	DiskHealthStatus     string    `json:"disk_health_status"`
	RAMTotalGB           float64   `json:"ram_total_gb"`
	RAMSlotsUsed         int       `json:"ram_slots_used"`
	RAMSlotsTotal        int       `json:"ram_slots_total"`
	RAMDDRType           string    `json:"ram_ddr_type"`
	BatteryHealthPercent *float64  `json:"battery_health_percent"`
	SerialNumber         string    `json:"serial_number"`
	IPAddress            string    `json:"ip_address"`
	MACAddress           string    `json:"mac_address"`
	OSVersion            string    `json:"os_version"`
	Applications         []AppInfo `json:"applications"`
}

type AppInfo struct {
	Name        string `json:"app_name"`
	Version     string `json:"app_version"`
	Publisher   string `json:"publisher"`
	InstallDate string `json:"install_date"`
}

func collectAssetInfo(serverID string, deviceType string) AssetInfo {
	return AssetInfo{ServerID: serverID}
}
