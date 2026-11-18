//go:build windows

package main

import (
	"fmt"
	"log"
	"net"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	psdisk "github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/process"
	"github.com/yusufpapurcu/wmi"
	"golang.org/x/sys/windows/registry"
)

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

type AppInfo struct {
	Name         string `json:"app_name"`
	Version      string `json:"app_version"`
	Publisher    string `json:"publisher"`
	InstallDate  string `json:"install_date"`
	InstallScope string `json:"install_scope"`
}

type DeviceUser struct {
	Username   string `json:"username"`
	FullName   string `json:"full_name"`
	Privilege  string `json:"privilege"`
	IsDisabled bool   `json:"is_disabled"`
	IsLoggedIn bool   `json:"is_logged_in"`
}

type win32Processor struct {
	Name          string
	NumberOfCores uint32
	MaxClockSpeed uint32
}

type win32DiskDrive struct {
	Size      uint64
	MediaType string
	Model     string
}

type msftPhysicalDisk struct {
	FriendlyName string
	MediaType    uint16
	BusType      uint16
	Size         uint64
}

type msftStorageReliabilityCounter struct {
	Wear         uint8
	Temperature  uint8
	PowerOnHours uint32
}

type win32PhysicalMemory struct {
	Capacity             uint64
	SMBIOSMemoryType     uint16
	MemoryType           uint16
	Speed                uint32
	ConfiguredClockSpeed uint32
	FormFactor           uint16
}

type win32PhysicalMemoryArray struct {
	MemoryDevices uint32
	MaxCapacity   uint32
	MaxCapacityEx uint64
}

type win32BIOS struct {
	SerialNumber string
	ReleaseDate  string
}

type win32ComputerSystem struct {
	Manufacturer string
	Model        string
}

type win32VideoController struct {
	Name string
}

type softwareLicensingProduct struct {
	Name                      string
	Description               string
	LicenseStatus             uint32
	PartialProductKey         string
	ProductKeyChannel         string
	KeyManagementServiceMachine string
}

type msftMpComputerStatus struct {
	AntivirusEnabled          bool
	RealTimeProtectionEnabled bool
}

type antiVirusProduct struct {
	DisplayName  string
	ProductState uint32
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
	collectSystemIdentity(&asset)
	collectProcessorInfo(&asset)
	collectDiskInfo(&asset)
	collectDiskVolumes(&asset)
	collectRAMInfo(&asset)
	collectNetworkIdentity(&asset)
	collectOSVersion(&asset)
	asset.Licenses = collectLicenses()
	asset.SecurityStatus = collectSecurityStatus()
	asset.RunningProcesses = collectRunningProcesses()
	asset.Applications = collectInstalledApplications()
	asset.DeviceUsers = collectDeviceUsers()
	if deviceType == "laptop" {
		collectBatteryHealth(&asset)
	}
	return asset
}

func collectSystemIdentity(asset *AssetInfo) {
	var cs []win32ComputerSystem
	if err := wmi.Query("SELECT Manufacturer, Model FROM Win32_ComputerSystem", &cs); err == nil && len(cs) > 0 {
		asset.DeviceManufacturer = strings.TrimSpace(cs[0].Manufacturer)
		asset.DeviceModel = strings.TrimSpace(cs[0].Model)
	}

	var bios []win32BIOS
	if err := wmi.Query("SELECT SerialNumber, ReleaseDate FROM Win32_BIOS", &bios); err == nil && len(bios) > 0 {
		asset.SerialNumber = strings.TrimSpace(bios[0].SerialNumber)
		asset.BIOSReleaseDate = parseWMIDate(bios[0].ReleaseDate)
	}

	var gpus []win32VideoController
	if err := wmi.Query("SELECT Name FROM Win32_VideoController", &gpus); err == nil && len(gpus) > 0 {
		seen := make(map[string]bool)
		var names []string
		for _, g := range gpus {
			name := strings.TrimSpace(g.Name)
			if name == "" || seen[name] {
				continue
			}
			seen[name] = true
			names = append(names, name)
		}
		asset.GPUModel = strings.Join(names, ", ")
	}
}

func parseWMIDate(raw string) string {
	s := strings.TrimSpace(raw)
	if len(s) < 8 {
		return ""
	}
	year, err1 := strconv.Atoi(s[0:4])
	month, err2 := strconv.Atoi(s[4:6])
	day, err3 := strconv.Atoi(s[6:8])
	if err1 != nil || err2 != nil || err3 != nil || year < 1990 || month < 1 || month > 12 || day < 1 || day > 31 {
		return ""
	}
	return fmt.Sprintf("%04d-%02d-%02d", year, month, day)
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

type msStorageDriverFailurePredictData struct {
	PredictFailure bool
	VendorSpecific []byte
	Reason         uint32
}

func classifyDiskType(busType, mediaType uint16, model string) string {
	if busType == 17 || strings.Contains(strings.ToUpper(model), "NVME") {
		return "NVMe SSD"
	}
	if mediaType == 4 || strings.Contains(strings.ToUpper(model), "SSD") {
		return "SSD"
	}
	if mediaType == 3 {
		return "HDD"
	}
	return "HDD"
}

func collectDiskInfo(asset *AssetInfo) {
	// 1. Prioritaskan MSFT_PhysicalDisk (akurat bedakan HDD / SSD / NVMe SSD di Win 10/11)
	var physDisks []msftPhysicalDisk
	if err := wmi.QueryNamespace("SELECT FriendlyName, MediaType, BusType, Size FROM MSFT_PhysicalDisk", &physDisks, `root\Microsoft\Windows\Storage`); err == nil && len(physDisks) > 0 {
		chosen := physDisks[0]
		for _, pd := range physDisks {
			if pd.BusType != 7 && pd.Size > 0 { // abaikan USB external jika ada internal disk
				chosen = pd
				break
			}
		}
		asset.DiskSizeGB = bytesToGB(chosen.Size)
		asset.DiskType = classifyDiskType(chosen.BusType, chosen.MediaType, chosen.FriendlyName)
	} else {
		var disks []win32DiskDrive
		if err := wmi.Query("SELECT Size, MediaType, Model FROM Win32_DiskDrive", &disks); err != nil || len(disks) == 0 {
			log.Printf("Gagal membaca info disk: %v", err)
			return
		}
		asset.DiskSizeGB = bytesToGB(disks[0].Size)
		modelOrMedia := disks[0].Model + " " + disks[0].MediaType
		if disks[0].MediaType == "SSD" || disks[0].MediaType == "Solid State Drive" || strings.Contains(strings.ToUpper(modelOrMedia), "SSD") {
			asset.DiskType = "SSD"
		} else if strings.Contains(strings.ToUpper(modelOrMedia), "NVME") {
			asset.DiskType = "NVMe SSD"
		} else {
			asset.DiskType = "HDD"
		}
	}

	var smart []msStorageDriverFailurePredictStatus
	err := wmi.QueryNamespace("SELECT PredictFailure FROM MSStorageDriver_FailurePredictStatus", &smart, `root\wmi`)
	if err != nil || len(smart) == 0 {
		asset.DiskHealthStatus = "UNKNOWN"
	} else {
		if smart[0].PredictFailure {
			asset.DiskHealthStatus = "WARNING"
		} else {
			asset.DiskHealthStatus = "OK"
		}
	}

	// 2. SMART wear level modern (NVMe & SATA SSD via Storage namespace)
	var counters []msftStorageReliabilityCounter
	if err := wmi.QueryNamespace("SELECT Wear, Temperature, PowerOnHours FROM MSFT_StorageReliabilityCounter", &counters, `root\Microsoft\Windows\Storage`); err == nil && len(counters) > 0 {
		for _, c := range counters {
			if c.Wear <= 100 && (c.PowerOnHours > 0 || c.Temperature > 0 || c.Wear > 0) {
				healthPercent := 100 - int(c.Wear)
				asset.DiskHealthPercent = &healthPercent
				break
			}
		}
	}

	// 3. Fallback SMART wear level lama (SATA AHCI via root\wmi)
	if asset.DiskHealthPercent == nil {
		var smartData []msStorageDriverFailurePredictData
		err = wmi.QueryNamespace("SELECT PredictFailure, VendorSpecific, Reason FROM MSStorageDriver_FailurePredictData", &smartData, `root\wmi`)
		if err == nil && len(smartData) > 0 && len(smartData[0].VendorSpecific) >= 4 {
			vendorData := smartData[0].VendorSpecific
			if len(vendorData) >= 512 {
				offset := 0
				if vendorData[0] == 0x01 {
					offset = 2
				}
				if len(vendorData) > offset+10 {
					wearLevel := int(vendorData[offset+5])
					if wearLevel >= 0 && wearLevel <= 100 {
						healthPercent := 100 - wearLevel
						asset.DiskHealthPercent = &healthPercent
					}
				}
			}
		}
	}
}

type win32LogicalDisk struct {
	DeviceID   string
	Size       uint64
	FreeSpace  uint64
	FileSystem string
}

func collectDiskVolumes(asset *AssetInfo) {
	var volumes []DiskVolume
	if parts, err := psdisk.Partitions(false); err == nil {
		for _, p := range parts {
			if strings.Contains(strings.ToLower(strings.Join(p.Opts, ",")), "cdrom") || p.Mountpoint == "" {
				continue
			}
			u, err := psdisk.Usage(p.Mountpoint)
			if err != nil || u.Total == 0 {
				continue
			}
			volumes = append(volumes, DiskVolume{
				Drive:       strings.TrimRight(p.Mountpoint, `\`),
				TotalGB:     bytesToGB(u.Total),
				UsedGB:      bytesToGB(u.Used),
				UsedPercent: u.UsedPercent,
				FileSystem:  p.Fstype,
			})
		}
	}

	// Fallback via WMI Win32_LogicalDisk (DriveType 3 = Local Fixed Disk)
	if len(volumes) == 0 {
		var ldisks []win32LogicalDisk
		if err := wmi.Query("SELECT DeviceID, Size, FreeSpace, FileSystem FROM Win32_LogicalDisk WHERE DriveType = 3 AND Size > 0", &ldisks); err == nil {
			for _, ld := range ldisks {
				used := uint64(0)
				if ld.Size > ld.FreeSpace {
					used = ld.Size - ld.FreeSpace
				}
				pct := (float64(used) / float64(ld.Size)) * 100
				volumes = append(volumes, DiskVolume{
					Drive:       strings.TrimSpace(ld.DeviceID),
					TotalGB:     bytesToGB(ld.Size),
					UsedGB:      bytesToGB(used),
					UsedPercent: pct,
					FileSystem:  strings.TrimSpace(ld.FileSystem),
				})
			}
		}
	}

	asset.DiskVolumes = volumes
}

func collectRAMInfo(asset *AssetInfo) {
	var memSticks []win32PhysicalMemory
	if err := wmi.Query("SELECT Capacity, SMBIOSMemoryType, MemoryType, Speed, ConfiguredClockSpeed, FormFactor FROM Win32_PhysicalMemory", &memSticks); err != nil {
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
		memType := memSticks[0].SMBIOSMemoryType
		if memType == 0 {
			memType = memSticks[0].MemoryType
		}
		asset.RAMDDRType = mapDDRType(memType)
		if memSticks[0].ConfiguredClockSpeed > 0 {
			asset.RAMSpeedMHz = int(memSticks[0].ConfiguredClockSpeed)
		} else {
			asset.RAMSpeedMHz = int(memSticks[0].Speed)
		}
		asset.RAMFormFactor = mapRAMFormFactor(memSticks[0].FormFactor)
	}
	var memArray []win32PhysicalMemoryArray
	if err := wmi.Query("SELECT MemoryDevices, MaxCapacity, MaxCapacityEx FROM Win32_PhysicalMemoryArray", &memArray); err == nil && len(memArray) > 0 {
		asset.RAMSlotsTotal = int(memArray[0].MemoryDevices)
		if memArray[0].MaxCapacityEx > 0 {
			asset.RAMMaxGB = float64(memArray[0].MaxCapacityEx) / 1024 / 1024
		} else if memArray[0].MaxCapacity > 0 {
			asset.RAMMaxGB = float64(memArray[0].MaxCapacity) / 1024 / 1024
		}
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

func readUninstallRegistry(hive registry.Key, regPath, scope string) []AppInfo {
	var apps []AppInfo
	key, err := registry.OpenKey(hive, regPath, registry.READ)
	if err != nil {
		return apps
	}
	subKeyNames, err := key.ReadSubKeyNames(-1)
	key.Close()
	if err != nil {
		return apps
	}
	for _, name := range subKeyNames {
		appKey, err := registry.OpenKey(hive, regPath+`\`+name, registry.READ)
		if err != nil {
			continue
		}
		displayName, _, err := appKey.GetStringValue("DisplayName")
		if err != nil || strings.TrimSpace(displayName) == "" {
			appKey.Close()
			continue
		}
		version, _, _ := appKey.GetStringValue("DisplayVersion")
		publisher, _, _ := appKey.GetStringValue("Publisher")
		installDate, _, _ := appKey.GetStringValue("InstallDate")
		apps = append(apps, AppInfo{
			Name:         strings.TrimSpace(displayName),
			Version:      strings.TrimSpace(version),
			Publisher:    strings.TrimSpace(publisher),
			InstallDate:  strings.TrimSpace(installDate),
			InstallScope: scope,
		})
		appKey.Close()
	}
	return apps
}

func isSystemSID(sid string) bool {
	upper := strings.ToUpper(strings.TrimSpace(sid))
	return upper == "" ||
		upper == ".DEFAULT" ||
		strings.HasSuffix(upper, "_CLASSES") ||
		upper == "S-1-5-18" ||
		upper == "S-1-5-19" ||
		upper == "S-1-5-20"
}

func deduplicateApps(apps []AppInfo) []AppInfo {
	seen := make(map[string]bool, len(apps))
	out := make([]AppInfo, 0, len(apps))
	for _, a := range apps {
		key := strings.ToLower(a.Name) + "|" + strings.ToLower(a.Version) + "|" + a.InstallScope
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, a)
	}
	return out
}

func collectInstalledApplications() []AppInfo {
	var apps []AppInfo
	uninstallSubPaths := []string{
		`SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`,
		`SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall`,
	}
	for _, p := range uninstallSubPaths {
		apps = append(apps, readUninstallRegistry(registry.LOCAL_MACHINE, p, "system")...)
	}

	// Scan per-user installations (karena agent jalan sebagai SYSTEM, baca lewat HKEY_USERS\<SID>)
	if sids, err := registry.USERS.ReadSubKeyNames(-1); err == nil {
		for _, sid := range sids {
			if isSystemSID(sid) {
				continue
			}
			for _, p := range uninstallSubPaths {
				apps = append(apps, readUninstallRegistry(registry.USERS, sid+`\`+p, "user")...)
			}
		}
	}

	return deduplicateApps(apps)
}

func mapLicenseStatus(code uint32) string {
	switch code {
	case 0:
		return "Unlicensed"
	case 1:
		return "Licensed"
	case 2:
		return "OOBGrace"
	case 3:
		return "OOTGrace"
	case 4:
		return "NonGenuine"
	case 5:
		return "Notification"
	case 6:
		return "ExtendedGrace"
	default:
		return fmt.Sprintf("Unknown(%d)", code)
	}
}

func extractChannelFromDesc(channel, desc string) string {
	if c := strings.TrimSpace(channel); c != "" {
		return c
	}
	upper := strings.ToUpper(desc)
	for _, marker := range []string{"VOLUME_KMSCLIENT", "VOLUME_MAK", "OEM_DM", "OEM_COA_NSLP", "RETAIL"} {
		if strings.Contains(upper, marker) {
			return marker
		}
	}
	return strings.TrimSpace(desc)
}

func classifyLicenseFamily(name string) string {
	lower := strings.ToLower(name)
	if strings.Contains(lower, "windows") {
		return "Windows"
	}
	if strings.Contains(lower, "office") {
		return "Office"
	}
	return "Other"
}

func collectLicenses() []LicenseInfo {
	var products []softwareLicensingProduct
	err := wmi.Query(
		"SELECT Name, Description, LicenseStatus, PartialProductKey, ProductKeyChannel, KeyManagementServiceMachine FROM SoftwareLicensingProduct WHERE PartialProductKey IS NOT NULL",
		&products,
	)
	if err != nil {
		log.Printf("Gagal membaca info lisensi Windows/Office: %v", err)
		return nil
	}
	var out []LicenseInfo
	for _, p := range products {
		name := strings.TrimSpace(p.Name)
		if name == "" {
			continue
		}
		out = append(out, LicenseInfo{
			ProductName:   name,
			Family:        classifyLicenseFamily(name),
			Channel:       extractChannelFromDesc(p.ProductKeyChannel, p.Description),
			LicenseStatus: mapLicenseStatus(p.LicenseStatus),
			PartialKey:    strings.TrimSpace(p.PartialProductKey),
			KMSMachine:    strings.TrimSpace(p.KeyManagementServiceMachine),
		})
	}
	return out
}

func decodeAVProductState(state uint32) bool {
	return (state & 0x1000) != 0
}

func collectSecurityStatus() SecurityStatus {
	sec := SecurityStatus{AntivirusNames: []string{}}
	avSeen := make(map[string]bool)

	var avProducts []antiVirusProduct
	if err := wmi.QueryNamespace("SELECT DisplayName, ProductState FROM AntiVirusProduct", &avProducts, `root\SecurityCenter2`); err == nil && len(avProducts) > 0 {
		sec.Reported = true
		for _, av := range avProducts {
			name := strings.TrimSpace(av.DisplayName)
			if name != "" && !avSeen[name] {
				avSeen[name] = true
				sec.AntivirusNames = append(sec.AntivirusNames, name)
			}
			if decodeAVProductState(av.ProductState) {
				sec.AntivirusEnabled = true
				sec.RealtimeProtection = true
			}
		}
	}

	var defender []msftMpComputerStatus
	if err := wmi.QueryNamespace("SELECT AntivirusEnabled, RealTimeProtectionEnabled FROM MSFT_MpComputerStatus", &defender, `root\Microsoft\Windows\Defender`); err == nil && len(defender) > 0 {
		sec.Reported = true
		if defender[0].AntivirusEnabled {
			sec.AntivirusEnabled = true
			if !avSeen["Windows Defender"] {
				sec.AntivirusNames = append(sec.AntivirusNames, "Windows Defender")
			}
		}
		if defender[0].RealTimeProtectionEnabled {
			sec.RealtimeProtection = true
		}
	}

	// Cek Windows Firewall langsung via Registry (cepat & tidak bergantung locale bahasa)
	fwProfiles := []string{
		`SYSTEM\CurrentControlSet\Services\SharedAccess\Parameters\FirewallPolicy\StandardProfile`,
		`SYSTEM\CurrentControlSet\Services\SharedAccess\Parameters\FirewallPolicy\PublicProfile`,
		`SYSTEM\CurrentControlSet\Services\SharedAccess\Parameters\FirewallPolicy\DomainProfile`,
	}
	for _, path := range fwProfiles {
		k, err := registry.OpenKey(registry.LOCAL_MACHINE, path, registry.READ)
		if err != nil {
			continue
		}
		val, _, err := k.GetIntegerValue("EnableFirewall")
		k.Close()
		if err == nil {
			sec.Reported = true
			if val == 1 {
				sec.FirewallEnabled = true
			}
		}
	}

	return sec
}

func collectRunningProcesses() []string {
	processes, err := process.Processes()
	if err != nil {
		return nil
	}
	seen := make(map[string]string)
	for _, p := range processes {
		name, err := p.Name()
		if err != nil {
			continue
		}
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, ok := seen[key]; !ok {
			seen[key] = trimmed
		}
	}
	out := make([]string, 0, len(seen))
	for _, name := range seen {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

type win32UserAccount struct {
	SID          string
	Name         string
	FullName     string
	Disabled     bool
	LocalAccount bool
}

type win32LoggedOnUser struct {
	Antecedent string
	Dependent  string
}

// ridFromSID mengambil Relative ID (angka terakhir setelah tanda '- terakhir).
// RID >= 1000 berarti akun dibuat manual oleh manusia; RID < 1000 akun bawaan sistem Windows.
func ridFromSID(sid string) int {
	idx := strings.LastIndex(sid, "-")
	if idx < 0 {
		return -1
	}
	rid, err := strconv.Atoi(sid[idx+1:])
	if err != nil {
		return -1
	}
	return rid
}

func collectDeviceUsers() []DeviceUser {
	var accounts []win32UserAccount
	if err := wmi.Query("SELECT SID, Name, FullName, Disabled, LocalAccount FROM Win32_UserAccount WHERE LocalAccount = TRUE", &accounts); err != nil {
		log.Printf("Gagal membaca daftar user lokal: %v", err)
		return nil
	}

	adminNames := collectLocalGroupMembers("Administrators")
	userNames := collectLocalGroupMembers("Users")
	loggedIn := collectLoggedInUsers()

	var users []DeviceUser
	for _, a := range accounts {
		name := strings.ToLower(a.Name)
		upperName := strings.ToUpper(a.Name)

		// Hanya akun aktif. "Administrator" juga ditampilkan asal di-enable manual oleh IT.
		if a.Disabled {
			continue
		}

		// RID >= 1000 = akun dibuat manual oleh manusia.
		// RID < 1000 = akun bawaan sistem (Guest, DefaultAccount, WDAGUtilityAccount, dll),
		// kecuali "Administrator" (RID 500) yang boleh muncul kalau aktif.
		rid := ridFromSID(a.SID)
		if rid >= 0 && rid < 1000 && name != "administrator" {
			continue
		}

		// Buang sisa akun sistem yang tidak punya RID standar
		if strings.HasSuffix(name, "$") ||
			strings.Contains(name, "sandbox") ||
			strings.Contains(name, "wdag") ||
			name == "defaultaccount" {
			continue
		}

		u := DeviceUser{
			Username:   a.Name,
			FullName:   a.FullName,
			Privilege:  "standard_user",
			IsDisabled: a.Disabled,
			IsLoggedIn: loggedIn[upperName],
		}
		if adminNames[upperName] {
			u.Privilege = "administrator"
		} else if !userNames[upperName] {
			u.Privilege = "limited"
		}
		users = append(users, u)
	}
	return users
}

// trimDomainPrefix membuang prefix "DOMAIN\" dari path WMI, misal
// collectLocalGroupMembers mengembalikan nama anggota grup lokal (uppercase) via `net localgroup`.
// Class asosiasi WMI (Win32GroupUser) tidak bisa di-query dengan SELECT biasa, jadi pakai tool
// bawaan Windows yang hasilnya stabil.
// ponytail: ganti dengan Win32 API NetUserGetLocalGroups kalau butuh performa tinggi — laporan
// asset dikirim 1×/24 jam, `net localgroup` sudah cukup.
func collectLocalGroupMembers(groupName string) map[string]bool {
	members := make(map[string]bool)
	out, err := exec.Command("net", "localgroup", groupName).Output()
	if err != nil {
		log.Printf("Gagal membaca anggota grup %q: %v", groupName, err)
		return members
	}
	// Baris pertama = "Alias name ...", baris kedua = "----------------", sisanya anggota
	// yang diakhiri "--" pada locale Inggris.
	lines := strings.Split(string(out), "\n")
	started := false
	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		trimmed := strings.TrimSpace(line)
		if !started {
			// lewati 2 baris header
			if strings.HasPrefix(trimmed, "Alias name") || strings.HasPrefix(trimmed, "Group name") || strings.HasPrefix(trimmed, "---") {
				started = true
			}
			continue
		}
		if trimmed == "" || strings.HasPrefix(trimmed, "The command completed") ||
			strings.HasPrefix(trimmed, "Perintah selesai") {
			continue
		}
		// lewati garis pemisah "-----" dan baris keterangan bawaan Windows
		if strings.HasPrefix(trimmed, "---") || strings.HasPrefix(trimmed, "===") {
			continue
		}
		// baris "COMMENT ..." / "MEMBERS ..." bukan nama akun
		if fields := strings.Fields(trimmed); len(fields) > 0 {
			switch strings.ToUpper(fields[0]) {
			case "COMMENT", "MEMBERS":
				continue
			}
		}
		if idx := strings.Index(trimmed, "--"); idx > 0 {
			trimmed = strings.TrimSpace(trimmed[:idx])
		}
		name := strings.TrimSpace(trimDomainPrefix(trimmed))
		if name != "" {
			members[strings.ToUpper(name)] = true
		}
	}
	return members
}

// trimDomainPrefix membuang prefix "DOMAIN\" dari path Windows, misal
// "DESKTOP-92PUGLA\IT 10" -> "IT 10"
func trimDomainPrefix(s string) string {
	if idx := strings.LastIndex(s, `\`); idx >= 0 {
		return s[idx+1:]
	}
	return s
}

// collectLoggedInUsers mengembalikan nama user yang punya sesi login aktif.
// Nama user ada di sisi Antecedent (Win32_Account), sedangkan Dependent berisi LogonId.
func collectLoggedInUsers() map[string]bool {
	loggedIn := make(map[string]bool)
	var sessions []win32LoggedOnUser
	if err := wmi.Query("SELECT Antecedent, Dependent FROM Win32_LoggedOnUser", &sessions); err != nil {
		log.Printf("Gagal membaca sesi login: %v", err)
		return loggedIn
	}
	for _, s := range sessions {
		name := extractWMIKeyValue(s.Antecedent, "Name")
		if name == "" {
			continue
		}
		loggedIn[strings.ToUpper(trimDomainPrefix(name))] = true
	}
	return loggedIn
}

// extractWMIKeyValue mengambil nilai dari referensi objek WMI, misal
// `\\.\root\cimv2:Win32_Account.Domain="DESKTOP",Name="IT 10"` + key "Name" -> "IT 10"
func extractWMIKeyValue(ref, key string) string {
	marker := key + "=\""
	idx := strings.Index(ref, marker)
	if idx < 0 {
		return ""
	}
	rest := ref[idx+len(marker):]
	end := strings.Index(rest, `"`)
	if end < 0 {
		return ""
	}
	return rest[:end]
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
	case 27:
		return "LPDDR"
	case 28:
		return "LPDDR2"
	case 29:
		return "LPDDR3"
	case 30:
		return "LPDDR4"
	case 34:
		return "DDR5"
	case 35:
		return "LPDDR5"
	default:
		return "Unknown"
	}
}

func mapRAMFormFactor(code uint16) string {
	switch code {
	case 8:
		return "DIMM"
	case 12:
		return "SODIMM"
	case 13:
		return "SRIMM"
	default:
		return ""
	}
}

func bytesToGB(b uint64) float64 {
	return float64(b) / 1024 / 1024 / 1024
}
