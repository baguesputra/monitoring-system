//go:build windows

package main

import (
	"log"
	"net"
	"os/exec"
	"strconv"
	"strings"

	"github.com/shirou/gopsutil/v3/host"
	"github.com/yusufpapurcu/wmi"
	"golang.org/x/sys/windows/registry"
)

type AssetInfo struct {
	ServerID             string       `json:"server_id"`
	ProcessorModel       string       `json:"processor_model"`
	ProcessorCores       int          `json:"processor_cores"`
	ProcessorClockGHz    float64      `json:"processor_clock_ghz"`
	DiskType             string       `json:"disk_type"`
	DiskSizeGB           float64      `json:"disk_size_gb"`
	DiskHealthStatus     string       `json:"disk_health_status"`
	DiskHealthPercent    *int         `json:"disk_health_percent"`
	RAMTotalGB           float64      `json:"ram_total_gb"`
	RAMSlotsUsed         int          `json:"ram_slots_used"`
	RAMSlotsTotal        int          `json:"ram_slots_total"`
	RAMDDRType           string       `json:"ram_ddr_type"`
	BatteryHealthPercent *float64     `json:"battery_health_percent"`
	SerialNumber         string       `json:"serial_number"`
	IPAddress            string       `json:"ip_address"`
	MACAddress           string       `json:"mac_address"`
	OSVersion            string       `json:"os_version"`
	Applications         []AppInfo    `json:"applications"`
	DeviceUsers          []DeviceUser `json:"device_users"`
}

type AppInfo struct {
	Name        string `json:"app_name"`
	Version     string `json:"app_version"`
	Publisher   string `json:"publisher"`
	InstallDate string `json:"install_date"`
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
	asset.DeviceUsers = collectDeviceUsers()
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

type msStorageDriverFailurePredictData struct {
	PredictFailure bool
	VendorSpecific []byte
	Reason         uint32
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
	} else {
		if smart[0].PredictFailure {
			asset.DiskHealthStatus = "WARNING"
		} else {
			asset.DiskHealthStatus = "OK"
		}
	}

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
	case 34:
		return "DDR5"
	default:
		return "Unknown"
	}
}

func bytesToGB(b uint64) float64 {
	return float64(b) / 1024 / 1024 / 1024
}
