//go:build windows

package main

import (
	"log"
	"net"

	"github.com/shirou/gopsutil/v3/host"
	"github.com/yusufpapurcu/wmi"
	"golang.org/x/sys/windows/registry"
)

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

type win32Processor struct {
	Name          string
	NumberOfCores uint32
	MaxClockSpeed uint32
}

type win32DiskDrive struct {
	Size      uint64
	MediaType string
}

type win32PhysicalMemory struct {
	Capacity         uint64
	SMBIOSMemoryType uint16
}

type win32PhysicalMemoryArray struct {
	MemoryDevices uint32
}

type win32BIOS struct {
	SerialNumber string
}

type msStorageDriverFailurePredictStatus struct {
	PredictFailure bool
}

type batteryStaticData struct {
	DesignedCapacity uint32
}

type batteryFullChargedCapacity struct {
	FullChargedCapacity uint32
}

func collectAssetInfo(serverID string, deviceType string) AssetInfo {
	asset := AssetInfo{ServerID: serverID}
	collectProcessorInfo(&asset)
	collectDiskInfo(&asset)
	collectRAMInfo(&asset)
	collectSerialNumber(&asset)
	collectNetworkIdentity(&asset)
	collectOSVersion(&asset)
	asset.Applications = collectInstalledApplications()
	if deviceType == "laptop" {
		collectBatteryHealth(&asset)
	}
	return asset
}

func collectProcessorInfo(asset *AssetInfo) {
	var cpus []win32Processor
	if err := wmi.Query("SELECT Name, NumberOfCores, MaxClockSpeed FROM Win32_Processor", &cpus); err != nil || len(cpus) == 0 {
		log.Printf("Gagal membaca info processor: %v", err)
		return
	}
	asset.ProcessorModel = cpus[0].Name
	asset.ProcessorCores = int(cpus[0].NumberOfCores)
	asset.ProcessorClockGHz = float64(cpus[0].MaxClockSpeed) / 1000
}

func collectDiskInfo(asset *AssetInfo) {
	var disks []win32DiskDrive
	if err := wmi.Query("SELECT Size, MediaType FROM Win32_DiskDrive", &disks); err != nil || len(disks) == 0 {
		log.Printf("Gagal membaca info disk: %v", err)
		return
	}
	asset.DiskSizeGB = bytesToGB(disks[0].Size)
	if disks[0].MediaType == "SSD" || disks[0].MediaType == "Solid State Drive" {
		asset.DiskType = "SSD"
	} else {
		asset.DiskType = "HDD"
	}
	var smart []msStorageDriverFailurePredictStatus
	err := wmi.QueryNamespace("SELECT PredictFailure FROM MSStorageDriver_FailurePredictStatus", &smart, `root\wmi`)
	if err != nil || len(smart) == 0 {
		asset.DiskHealthStatus = "UNKNOWN"
		return
	}
	if smart[0].PredictFailure {
		asset.DiskHealthStatus = "WARNING"
	} else {
		asset.DiskHealthStatus = "OK"
	}
}

func collectRAMInfo(asset *AssetInfo) {
	var memSticks []win32PhysicalMemory
	if err := wmi.Query("SELECT Capacity, SMBIOSMemoryType FROM Win32_PhysicalMemory", &memSticks); err != nil {
		log.Printf("Gagal membaca info RAM: %v", err)
		return
	}
	var totalBytes uint64
	for _, m := range memSticks {
		totalBytes += m.Capacity
	}
	asset.RAMTotalGB = bytesToGB(totalBytes)
	asset.RAMSlotsUsed = len(memSticks)
	if len(memSticks) > 0 {
		asset.RAMDDRType = mapDDRType(memSticks[0].SMBIOSMemoryType)
	}
	var memArray []win32PhysicalMemoryArray
	if err := wmi.Query("SELECT MemoryDevices FROM Win32_PhysicalMemoryArray", &memArray); err == nil && len(memArray) > 0 {
		asset.RAMSlotsTotal = int(memArray[0].MemoryDevices)
	}
}

func collectSerialNumber(asset *AssetInfo) {
	var bios []win32BIOS
	if err := wmi.Query("SELECT SerialNumber FROM Win32_BIOS", &bios); err == nil && len(bios) > 0 {
		asset.SerialNumber = bios[0].SerialNumber
	}
}

func collectOSVersion(asset *AssetInfo) {
	if info, err := host.Info(); err == nil {
		asset.OSVersion = info.Platform + " " + info.PlatformVersion
	}
}

func collectBatteryHealth(asset *AssetInfo) {
	var designed []batteryStaticData
	err1 := wmi.QueryNamespace("SELECT DesignedCapacity FROM BatteryStaticData", &designed, `root\wmi`)
	var full []batteryFullChargedCapacity
	err2 := wmi.QueryNamespace("SELECT FullChargedCapacity FROM BatteryFullChargedCapacity", &full, `root\wmi`)
	if err1 != nil || err2 != nil || len(designed) == 0 || len(full) == 0 || designed[0].DesignedCapacity == 0 {
		log.Printf("Gagal membaca battery health (mungkin device tidak punya baterai)")
		return
	}
	health := (float64(full[0].FullChargedCapacity) / float64(designed[0].DesignedCapacity)) * 100
	asset.BatteryHealthPercent = &health
}

func collectNetworkIdentity(asset *AssetInfo) {
	interfaces, err := net.Interfaces()
	if err != nil {
		log.Printf("Gagal membaca network interfaces: %v", err)
		return
	}
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil || len(addrs) == 0 {
			continue
		}
		for _, addr := range addrs {
			if ipNet, ok := addr.(*net.IPNet); ok && ipNet.IP.To4() != nil {
				asset.IPAddress = ipNet.IP.String()
				asset.MACAddress = iface.HardwareAddr.String()
				return
			}
		}
	}
}

func collectInstalledApplications() []AppInfo {
	var apps []AppInfo
	uninstallPaths := []struct {
		hive registry.Key
		path string
	}{
		{registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`},
		{registry.LOCAL_MACHINE, `SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall`},
	}
	for _, up := range uninstallPaths {
		key, err := registry.OpenKey(up.hive, up.path, registry.READ)
		if err != nil {
			continue
		}
		subKeyNames, err := key.ReadSubKeyNames(-1)
		key.Close()
		if err != nil {
			continue
		}
		for _, name := range subKeyNames {
			appKey, err := registry.OpenKey(up.hive, up.path+`\`+name, registry.READ)
			if err != nil {
				continue
			}
			displayName, _, err := appKey.GetStringValue("DisplayName")
			if err != nil || displayName == "" {
				appKey.Close()
				continue
			}
			version, _, _ := appKey.GetStringValue("DisplayVersion")
			publisher, _, _ := appKey.GetStringValue("Publisher")
			installDate, _, _ := appKey.GetStringValue("InstallDate")
			apps = append(apps, AppInfo{
				Name:        displayName,
				Version:     version,
				Publisher:   publisher,
				InstallDate: installDate,
			})
			appKey.Close()
		}
	}
	return apps
}

func mapDDRType(code uint16) string {
	switch code {
	case 20:
		return "DDR"
	case 21:
		return "DDR2"
	case 24:
		return "DDR3"
	case 26:
		return "DDR4"
	case 34:
		return "DDR5"
	default:
		return "Unknown"
	}
}

func bytesToGB(b uint64) float64 {
	return float64(b) / 1024 / 1024 / 1024
}
