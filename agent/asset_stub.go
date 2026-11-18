//go:build !windows

package main

type DiskVolume struct {
	Drive       string  `json:"drive"`
	TotalGB     float64 `json:"total_gb"`
	UsedGB      float64 `json:"used_gb"`
	UsedPercent float64 `json:"used_percent"`
	FileSystem  string  `json:"file_system"`
}

type LicenseInfo struct {
	ProductName   string `json:"product_name"`
	Family        string `json:"family"`
	Channel       string `json:"channel"`
	LicenseStatus string `json:"license_status"`
	PartialKey    string `json:"partial_key"`
	KMSMachine    string `json:"kms_machine"`
}

type SecurityStatus struct {
	AntivirusEnabled   bool     `json:"antivirus_enabled"`
	RealtimeProtection bool     `json:"realtime_protection"`
	AntivirusNames     []string `json:"antivirus_names"`
	FirewallEnabled    bool     `json:"firewall_enabled"`
	Reported           bool     `json:"reported"`
}

type AssetInfo struct {
	ServerID             string         `json:"server_id"`
	DeviceManufacturer   string         `json:"device_manufacturer"`
	DeviceModel          string         `json:"device_model"`
	BIOSReleaseDate      string         `json:"bios_release_date"`
	GPUModel             string         `json:"gpu_model"`
	ProcessorModel       string         `json:"processor_model"`
	ProcessorCores       int            `json:"processor_cores"`
	ProcessorClockGHz    float64        `json:"processor_clock_ghz"`
	DiskType             string         `json:"disk_type"`
	DiskSizeGB           float64        `json:"disk_size_gb"`
	DiskHealthStatus     string         `json:"disk_health_status"`
	DiskHealthPercent    *int           `json:"disk_health_percent"`
	DiskVolumes          []DiskVolume   `json:"disk_volumes"`
	RAMTotalGB           float64        `json:"ram_total_gb"`
	RAMSlotsUsed         int            `json:"ram_slots_used"`
	RAMSlotsTotal        int            `json:"ram_slots_total"`
	RAMDDRType           string         `json:"ram_ddr_type"`
	RAMSpeedMHz          int            `json:"ram_speed_mhz"`
	RAMMaxGB             float64        `json:"ram_max_gb"`
	RAMFormFactor        string         `json:"ram_form_factor"`
	BatteryHealthPercent *float64       `json:"battery_health_percent"`
	SerialNumber         string         `json:"serial_number"`
	IPAddress            string         `json:"ip_address"`
	MACAddress           string         `json:"mac_address"`
	OSVersion            string         `json:"os_version"`
	Licenses             []LicenseInfo  `json:"licenses"`
	SecurityStatus       SecurityStatus `json:"security_status"`
	RunningProcesses     []string       `json:"running_processes"`
	Applications         []AppInfo      `json:"applications"`
	DeviceUsers          []DeviceUser   `json:"device_users"`
}

type DeviceUser struct {
	Username   string `json:"username"`
	FullName   string `json:"full_name"`
	Privilege  string `json:"privilege"`
	IsDisabled bool   `json:"is_disabled"`
	IsLoggedIn bool   `json:"is_logged_in"`
}

type AppInfo struct {
	Name         string `json:"app_name"`
	Version      string `json:"app_version"`
	Publisher    string `json:"publisher"`
	InstallDate  string `json:"install_date"`
	InstallScope string `json:"install_scope"`
}

func collectAssetInfo(serverID string, deviceType string) AssetInfo {
	return AssetInfo{ServerID: serverID}
}
