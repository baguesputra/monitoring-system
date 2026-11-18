package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

type UpgradeRecommendation struct {
	Category string `json:"category"` // storage | ram | cpu | battery | age
	Severity string `json:"severity"` // critical | recommended
	Title    string `json:"title"`
	Detail   string `json:"detail"`
	Action   string `json:"action"`
}

type DeviceUpgradeInsight struct {
	ServerID             string                  `json:"server_id"`
	Hostname             string                  `json:"hostname"`
	DeviceType           string                  `json:"device_type"`
	OwnerName            string                  `json:"owner_name"`
	Department           string                  `json:"department"`
	DeviceManufacturer   string                  `json:"device_manufacturer"`
	DeviceModel          string                  `json:"device_model"`
	BIOSReleaseDate      string                  `json:"bios_release_date"`
	DeviceAgeYears       *int                    `json:"device_age_years"`
	ProcessorModel       string                  `json:"processor_model"`
	GPUModel             string                  `json:"gpu_model"`
	RAMTotalGB           float64                 `json:"ram_total_gb"`
	RAMSlotsUsed         int                     `json:"ram_slots_used"`
	RAMSlotsTotal        int                     `json:"ram_slots_total"`
	RAMDDRType           string                  `json:"ram_ddr_type"`
	RAMSpeedMHz          int                     `json:"ram_speed_mhz"`
	RAMMaxGB             float64                 `json:"ram_max_gb"`
	RAMFormFactor        string                  `json:"ram_form_factor"`
	DiskType             string                  `json:"disk_type"`
	DiskSizeGB           float64                 `json:"disk_size_gb"`
	DiskHealthStatus     string                  `json:"disk_health_status"`
	DiskHealthPercent    *int                    `json:"disk_health_percent"`
	DiskVolumes          []DiskVolume            `json:"disk_volumes"`
	BatteryHealthPercent *float64                `json:"battery_health_percent"`
	AvgCPUPercent        float64                 `json:"avg_cpu_percent"`
	P95CPUPercent        float64                 `json:"p95_cpu_percent"`
	AvgRAMPercent        float64                 `json:"avg_ram_percent"`
	P95RAMPercent        float64                 `json:"p95_ram_percent"`
	LatestDiskPercent    float64                 `json:"latest_disk_percent"`
	MetricSamples        int                     `json:"metric_samples"`
	Priority             string                  `json:"priority"` // critical | recommended | healthy
	Recommendations      []UpgradeRecommendation `json:"recommendations"`
}

type UpgradeInsightsResponse struct {
	WindowDays       int                    `json:"window_days"`
	TotalDevices     int                    `json:"total_devices"`
	CriticalCount    int                    `json:"critical_count"`
	RecommendedCount int                    `json:"recommended_count"`
	HealthyCount     int                    `json:"healthy_count"`
	Devices          []DeviceUpgradeInsight `json:"devices"`
}

func parseBIOSAgeYears(releaseDate string, now time.Time) *int {
	if len(releaseDate) < 4 {
		return nil
	}
	year, err := strconv.Atoi(releaseDate[:4])
	if err != nil || year < 2000 || year > now.Year() {
		return nil
	}
	age := now.Year() - year
	return &age
}

func formatRAMSpec(ddr string, speedMHz int, formFactor string) string {
	parts := []string{}
	if ddr != "" && ddr != "Unknown" {
		parts = append(parts, ddr)
	}
	if speedMHz > 0 {
		parts = append(parts, fmt.Sprintf("%d MHz", speedMHz))
	}
	if formFactor != "" {
		parts = append(parts, formFactor)
	}
	if len(parts) == 0 {
		return "RAM"
	}
	return strings.Join(parts, " ")
}

func evaluateUpgradeDevice(d DeviceUpgradeInsight, now time.Time) DeviceUpgradeInsight {
	d.DeviceAgeYears = parseBIOSAgeYears(d.BIOSReleaseDate, now)
	recs := []UpgradeRecommendation{}

	// 1. Storage type (HDD -> SSD/NVMe)
	if strings.EqualFold(d.DiskType, "HDD") {
		recs = append(recs, UpgradeRecommendation{
			Category: "storage",
			Severity: "critical",
			Title:    "Storage masih memakai HDD",
			Detail:   fmt.Sprintf("Disk utama berjenis HDD (%.0f GB) yang menjadi bottleneck utama performa Windows.", d.DiskSizeGB),
			Action:   "Upgrade storage utama ke SSD / NVMe SSD.",
		})
	}

	// 2. Disk health (SMART / Wear)
	if strings.EqualFold(d.DiskHealthStatus, "WARNING") || (d.DiskHealthPercent != nil && *d.DiskHealthPercent < 50) {
		detail := fmt.Sprintf("SMART status: %s", d.DiskHealthStatus)
		if d.DiskHealthPercent != nil {
			detail = fmt.Sprintf("Kesehatan disk tersisa %d%% (SMART: %s)", *d.DiskHealthPercent, d.DiskHealthStatus)
		}
		recs = append(recs, UpgradeRecommendation{
			Category: "storage",
			Severity: "critical",
			Title:    "Kesehatan disk kritis",
			Detail:   detail,
			Action:   "Segera backup data penting dan ganti unit SSD/HDD baru.",
		})
	} else if d.DiskHealthPercent != nil && *d.DiskHealthPercent < 80 {
		recs = append(recs, UpgradeRecommendation{
			Category: "storage",
			Severity: "recommended",
			Title:    "Wear level disk mulai menurun",
			Detail:   fmt.Sprintf("Kesehatan SSD tersisa %d%%.", *d.DiskHealthPercent),
			Action:   "Jadwalkan penggantian SSD saat siklus perawatan berikutnya.",
		})
	}

	// 3. Disk capacity full (check primary & all volumes)
	maxVolPct := d.LatestDiskPercent
	fullDrive := "C:"
	for _, v := range d.DiskVolumes {
		if v.UsedPercent > maxVolPct {
			maxVolPct = v.UsedPercent
			fullDrive = v.Drive
		}
	}
	if maxVolPct >= 85 {
		sev := "recommended"
		if maxVolPct >= 92 {
			sev = "critical"
		}
		recs = append(recs, UpgradeRecommendation{
			Category: "storage",
			Severity: sev,
			Title:    fmt.Sprintf("Kapasitas disk %s hampir penuh (%.0f%%)", fullDrive, maxVolPct),
			Detail:   fmt.Sprintf("Partisi %s telah terpakai %.1f%%.", fullDrive, maxVolPct),
			Action:   "Bersihkan file arsip atau upgrade kapasitas SSD lebih besar.",
		})
	}

	// 4. RAM bottleneck & exact upgrade recommendation
	ramPressured := (d.MetricSamples > 0 && (d.AvgRAMPercent >= 80 || d.P95RAMPercent >= 90)) || (d.RAMTotalGB > 0 && d.RAMTotalGB <= 4.5)
	if ramPressured {
		spec := formatRAMSpec(d.RAMDDRType, d.RAMSpeedMHz, d.RAMFormFactor)
		sev := "recommended"
		if d.AvgRAMPercent >= 88 || (d.RAMTotalGB > 0 && d.RAMTotalGB <= 4.5) {
			sev = "critical"
		}
		var action string
		if d.RAMSlotsTotal > 0 && d.RAMSlotsUsed < d.RAMSlotsTotal {
			freeSlots := d.RAMSlotsTotal - d.RAMSlotsUsed
			action = fmt.Sprintf("Tambah keping %s — masih tersedia %d dari %d slot kosong.", spec, freeSlots, d.RAMSlotsTotal)
		} else if d.RAMSlotsTotal > 0 && d.RAMSlotsUsed >= d.RAMSlotsTotal {
			if d.RAMMaxGB > d.RAMTotalGB+1 {
				action = fmt.Sprintf("Semua %d slot RAM penuh (%.0f GB). Ganti ke kapasitas lebih besar (%s, maks motherboard %.0f GB).", d.RAMSlotsTotal, d.RAMTotalGB, spec, d.RAMMaxGB)
			} else {
				action = fmt.Sprintf("Slot RAM penuh/maksimal (%.0f GB). Pertimbangkan upgrade unit baru jika beban kerja meningkat.", d.RAMTotalGB)
			}
		} else {
			action = fmt.Sprintf("Upgrade kapasitas RAM dari %.0f GB (rekomendasi minimal 16 GB, %s).", d.RAMTotalGB, spec)
		}
		recs = append(recs, UpgradeRecommendation{
			Category: "ram",
			Severity: sev,
			Title:    "Kapasitas RAM tertekan",
			Detail:   fmt.Sprintf("RAM terpasang %.0f GB (rata-rata terpakai %.1f%%, puncak P95 %.1f%%).", d.RAMTotalGB, d.AvgRAMPercent, d.P95RAMPercent),
			Action:   action,
		})
	}

	// 5. CPU load & device age
	if d.MetricSamples > 0 && (d.AvgCPUPercent >= 75 || d.P95CPUPercent >= 90) {
		sev := "recommended"
		if d.AvgCPUPercent >= 85 {
			sev = "critical"
		}
		recs = append(recs, UpgradeRecommendation{
			Category: "cpu",
			Severity: sev,
			Title:    "Beban processor tinggi secara konsisten",
			Detail:   fmt.Sprintf("CPU rata-rata %.1f%% (puncak P95 %.1f%%) pada %s.", d.AvgCPUPercent, d.P95CPUPercent, d.ProcessorModel),
			Action:   "Audit aplikasi berat yang berjalan atau rencanakan upgrade unit dengan CPU generasi lebih baru.",
		})
	}
	if d.DeviceAgeYears != nil && *d.DeviceAgeYears >= 5 {
		recs = append(recs, UpgradeRecommendation{
			Category: "age",
			Severity: "recommended",
			Title:    fmt.Sprintf("Usia perangkat sekitar %d tahun", *d.DeviceAgeYears),
			Detail:   fmt.Sprintf("Tanggal rilis BIOS %s (%s %s).", d.BIOSReleaseDate, d.DeviceManufacturer, d.DeviceModel),
			Action:   "Masukkan perangkat ke daftar evaluasi peremajaan aset IT.",
		})
	}

	// 6. Battery health for laptops
	if strings.EqualFold(d.DeviceType, "laptop") && d.BatteryHealthPercent != nil {
		if *d.BatteryHealthPercent < 50 {
			recs = append(recs, UpgradeRecommendation{
				Category: "battery",
				Severity: "critical",
				Title:    fmt.Sprintf("Baterai laptop drop (%.0f%%)", *d.BatteryHealthPercent),
				Detail:   "Kapasitas baterai tinggal di bawah 50% dari desain pabrik.",
				Action:   "Ganti baterai laptop agar mobilitas kerja tidak terganggu.",
			})
		} else if *d.BatteryHealthPercent < 70 {
			recs = append(recs, UpgradeRecommendation{
				Category: "battery",
				Severity: "recommended",
				Title:    fmt.Sprintf("Kesehatan baterai menurun (%.0f%%)", *d.BatteryHealthPercent),
				Detail:   "Daya tahan baterai mulai melemah.",
				Action:   "Siapkan pengadaan baterai pengganti dalam beberapa bulan ke depan.",
			})
		}
	}

	d.Priority = "healthy"
	for _, r := range recs {
		if r.Severity == "critical" {
			d.Priority = "critical"
			break
		}
		if r.Severity == "recommended" {
			d.Priority = "recommended"
		}
	}
	d.Recommendations = recs
	return d
}

func priorityRank(p string) int {
	switch p {
	case "critical":
		return 0
	case "recommended":
		return 1
	default:
		return 2
	}
}

func getUpgradeInsights(windowDays int) (*UpgradeInsightsResponse, error) {
	if windowDays <= 0 {
		windowDays = 30
	}
	since := time.Now().Add(-time.Duration(windowDays) * 24 * time.Hour)

	query := `
		WITH metric_agg AS (
			SELECT server_id,
			       COALESCE(AVG(cpu_percent), 0)::float8 AS avg_cpu,
			       COALESCE(PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY cpu_percent), 0)::float8 AS p95_cpu,
			       COALESCE(AVG(ram_percent), 0)::float8 AS avg_ram,
			       COALESCE(PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY ram_percent), 0)::float8 AS p95_ram,
			       COUNT(*)::int AS samples
			FROM metrics
			WHERE recorded_at >= $1
			GROUP BY server_id
		),
		latest_m AS (
			SELECT DISTINCT ON (server_id) server_id, disk_percent::float8 AS latest_disk
			FROM metrics
			ORDER BY server_id, recorded_at DESC
		)
		SELECT s.server_id, COALESCE(s.hostname,''), COALESCE(s.device_type,'pc'),
		       COALESCE(s.owner_name,''), COALESCE(s.department,''),
		       COALESCE(a.device_manufacturer,''), COALESCE(a.device_model,''), COALESCE(a.bios_release_date,''),
		       COALESCE(a.processor_model,''), COALESCE(a.gpu_model,''),
		       COALESCE(a.ram_total_gb,0)::float8, COALESCE(a.ram_slots_used,0), COALESCE(a.ram_slots_total,0),
		       COALESCE(a.ram_ddr_type,''), COALESCE(a.ram_speed_mhz,0), COALESCE(a.ram_max_gb,0)::float8, COALESCE(a.ram_form_factor,''),
		       COALESCE(a.disk_type,''), COALESCE(a.disk_size_gb,0)::float8, COALESCE(a.disk_health_status,''), a.disk_health_percent,
		       COALESCE(a.disk_volumes,'[]'::jsonb), a.battery_health_percent::float8,
		       COALESCE(ma.avg_cpu,0), COALESCE(ma.p95_cpu,0), COALESCE(ma.avg_ram,0), COALESCE(ma.p95_ram,0),
		       COALESCE(lm.latest_disk,0), COALESCE(ma.samples,0)
		FROM servers s
		LEFT JOIN asset_info a ON a.server_id = s.server_id
		LEFT JOIN metric_agg ma ON ma.server_id = s.server_id
		LEFT JOIN latest_m lm ON lm.server_id = s.server_id
		ORDER BY s.server_id ASC
	`

	rows, err := dbPool.Query(context.Background(), query, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	now := time.Now()
	devices := []DeviceUpgradeInsight{}
	for rows.Next() {
		var d DeviceUpgradeInsight
		var volumesRaw []byte
		if err := rows.Scan(
			&d.ServerID, &d.Hostname, &d.DeviceType, &d.OwnerName, &d.Department,
			&d.DeviceManufacturer, &d.DeviceModel, &d.BIOSReleaseDate,
			&d.ProcessorModel, &d.GPUModel,
			&d.RAMTotalGB, &d.RAMSlotsUsed, &d.RAMSlotsTotal,
			&d.RAMDDRType, &d.RAMSpeedMHz, &d.RAMMaxGB, &d.RAMFormFactor,
			&d.DiskType, &d.DiskSizeGB, &d.DiskHealthStatus, &d.DiskHealthPercent,
			&volumesRaw, &d.BatteryHealthPercent,
			&d.AvgCPUPercent, &d.P95CPUPercent, &d.AvgRAMPercent, &d.P95RAMPercent,
			&d.LatestDiskPercent, &d.MetricSamples,
		); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(volumesRaw, &d.DiskVolumes)
		if d.DiskVolumes == nil {
			d.DiskVolumes = []DiskVolume{}
		}
		d = evaluateUpgradeDevice(d, now)
		devices = append(devices, d)
	}

	sort.SliceStable(devices, func(i, j int) bool {
		ri, rj := priorityRank(devices[i].Priority), priorityRank(devices[j].Priority)
		if ri != rj {
			return ri < rj
		}
		return devices[i].ServerID < devices[j].ServerID
	})

	resp := &UpgradeInsightsResponse{
		WindowDays:   windowDays,
		TotalDevices: len(devices),
		Devices:      devices,
	}
	for _, d := range devices {
		switch d.Priority {
		case "critical":
			resp.CriticalCount++
		case "recommended":
			resp.RecommendedCount++
		default:
			resp.HealthyCount++
		}
	}
	return resp, nil
}

// ---- Software Policies & Policy Audit ----

type SoftwarePolicy struct {
	ID            int       `json:"id"`
	Pattern       string    `json:"pattern"`
	PolicyType    string    `json:"policy_type"` // forbidden | required | licensed
	Category      string    `json:"category"`
	LicensedSeats int       `json:"licensed_seats"`
	Notes         string    `json:"notes"`
	CreatedAt     time.Time `json:"created_at"`
}

type PolicyViolation struct {
	PolicyPattern string `json:"policy_pattern"`
	Category      string `json:"category"`
	MatchedName   string `json:"matched_name"`
	Source        string `json:"source"`        // installed | running_process
	InstallScope  string `json:"install_scope"` // system | user | process
	Notes         string `json:"notes"`
}

type LicenseFinding struct {
	ProductName   string `json:"product_name"`
	Family        string `json:"family"`
	Channel       string `json:"channel"`
	LicenseStatus string `json:"license_status"`
	Flag          string `json:"flag"` // ok | suspect_kms | unlicensed
	Reason        string `json:"reason"`
}

type DevicePolicyAudit struct {
	ServerID              string            `json:"server_id"`
	OwnerName             string            `json:"owner_name"`
	Department            string            `json:"department"`
	DeviceType            string            `json:"device_type"`
	ComplianceStatus      string            `json:"compliance_status"` // violation | warning | compliant
	ForbiddenHits         []PolicyViolation `json:"forbidden_hits"`
	MissingRequired       []string          `json:"missing_required"`
	LicenseFindings       []LicenseFinding  `json:"license_findings"`
	SecurityStatus        SecurityStatus    `json:"security_status"`
	SecurityWarnings      []string          `json:"security_warnings"`
	AdminUsers            []string          `json:"admin_users"`
	InstalledAppsCount    int               `json:"installed_apps_count"`
	UserScopeAppsCount    int               `json:"user_scope_apps_count"`
	RunningProcessesCount int               `json:"running_processes_count"`
}

type LicensedSeatSummary struct {
	Pattern       string   `json:"pattern"`
	Category      string   `json:"category"`
	LicensedSeats int      `json:"licensed_seats"`
	InstalledOn   []string `json:"installed_on"`
	OverAllocated bool     `json:"over_allocated"`
}

type PolicyAuditResponse struct {
	TotalDevices          int                   `json:"total_devices"`
	ViolationDevices      int                   `json:"violation_devices"`
	WarningDevices        int                   `json:"warning_devices"`
	CompliantDevices      int                   `json:"compliant_devices"`
	TotalForbiddenHits    int                   `json:"total_forbidden_hits"`
	SuspectLicenseDevices int                   `json:"suspect_license_devices"`
	WeakSecurityDevices   int                   `json:"weak_security_devices"`
	Policies              []SoftwarePolicy      `json:"policies"`
	LicenseSummaries      []LicensedSeatSummary `json:"license_summaries"`
	Devices               []DevicePolicyAudit   `json:"devices"`
}

func getSoftwarePolicies() ([]SoftwarePolicy, error) {
	rows, err := dbPool.Query(context.Background(),
		`SELECT id, pattern, policy_type, COALESCE(category,''), licensed_seats, COALESCE(notes,''), created_at
		 FROM software_policies ORDER BY policy_type ASC, pattern ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	policies := []SoftwarePolicy{}
	for rows.Next() {
		var p SoftwarePolicy
		if err := rows.Scan(&p.ID, &p.Pattern, &p.PolicyType, &p.Category, &p.LicensedSeats, &p.Notes, &p.CreatedAt); err != nil {
			return nil, err
		}
		policies = append(policies, p)
	}
	return policies, nil
}

func createSoftwarePolicy(p SoftwarePolicy) (*SoftwarePolicy, error) {
	var created SoftwarePolicy
	err := dbPool.QueryRow(context.Background(),
		`INSERT INTO software_policies (pattern, policy_type, category, licensed_seats, notes)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, pattern, policy_type, category, licensed_seats, notes, created_at`,
		p.Pattern, p.PolicyType, p.Category, p.LicensedSeats, p.Notes,
	).Scan(&created.ID, &created.Pattern, &created.PolicyType, &created.Category, &created.LicensedSeats, &created.Notes, &created.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &created, nil
}

func deleteSoftwarePolicy(id int) error {
	_, err := dbPool.Exec(context.Background(), `DELETE FROM software_policies WHERE id=$1`, id)
	return err
}

func evaluateLicenseFinding(l LicenseInfo) LicenseFinding {
	f := LicenseFinding{
		ProductName:   l.ProductName,
		Family:        l.Family,
		Channel:       l.Channel,
		LicenseStatus: l.LicenseStatus,
		Flag:          "ok",
	}
	chUpper := strings.ToUpper(l.Channel)
	stUpper := strings.ToUpper(l.LicenseStatus)
	if stUpper == "UNLICENSED" || stUpper == "NONGENUINE" || stUpper == "NOTIFICATION" || strings.Contains(stUpper, "GRACE") {
		f.Flag = "unlicensed"
		f.Reason = fmt.Sprintf("Status lisensi tidak aktif (%s)", l.LicenseStatus)
		return f
	}
	if strings.Contains(chUpper, "KMS") {
		f.Flag = "suspect_kms"
		if l.KMSMachine != "" {
			f.Reason = fmt.Sprintf("Aktivasi via KMS (%s) — verifikasi apakah server KMS resmi atau activator", l.KMSMachine)
		} else {
			f.Reason = "Aktivasi via channel KMS — verifikasi keaslian lisensi (rawan KMSpico/MAS)"
		}
	}
	return f
}

func evaluateDevicePolicy(
	srv Server,
	asset *AssetInfo,
	apps []InstalledApp,
	users []DeviceUser,
	policies []SoftwarePolicy,
) DevicePolicyAudit {
	audit := DevicePolicyAudit{
		ServerID:         srv.ServerID,
		OwnerName:        srv.OwnerName,
		Department:       srv.Department,
		DeviceType:       srv.DeviceType,
		ForbiddenHits:    []PolicyViolation{},
		MissingRequired:  []string{},
		LicenseFindings:  []LicenseFinding{},
		SecurityWarnings: []string{},
		AdminUsers:       []string{},
		InstalledAppsCount: len(apps),
	}

	for _, a := range apps {
		if strings.EqualFold(a.InstallScope, "user") {
			audit.UserScopeAppsCount++
		}
	}

	var procs []string
	if asset != nil {
		procs = asset.RunningProcesses
		audit.RunningProcessesCount = len(procs)
		audit.SecurityStatus = asset.SecurityStatus
		for _, lic := range asset.Licenses {
			audit.LicenseFindings = append(audit.LicenseFindings, evaluateLicenseFinding(lic))
		}
		if asset.SecurityStatus.Reported {
			if !asset.SecurityStatus.AntivirusEnabled {
				audit.SecurityWarnings = append(audit.SecurityWarnings, "Antivirus tidak aktif")
			} else if !asset.SecurityStatus.RealtimeProtection {
				audit.SecurityWarnings = append(audit.SecurityWarnings, "Real-time Protection Windows Defender / AV dimatikan")
			}
			if !asset.SecurityStatus.FirewallEnabled {
				audit.SecurityWarnings = append(audit.SecurityWarnings, "Windows Firewall tidak aktif")
			}
		}
	}

	for _, u := range users {
		if strings.EqualFold(u.Privilege, "administrator") && !u.IsDisabled {
			audit.AdminUsers = append(audit.AdminUsers, u.Username)
		}
	}

	for _, pol := range policies {
		patLower := strings.ToLower(strings.TrimSpace(pol.Pattern))
		if patLower == "" {
			continue
		}
		switch pol.PolicyType {
		case "forbidden":
			for _, app := range apps {
				if strings.Contains(strings.ToLower(app.AppName), patLower) {
					audit.ForbiddenHits = append(audit.ForbiddenHits, PolicyViolation{
						PolicyPattern: pol.Pattern,
						Category:      pol.Category,
						MatchedName:   app.AppName,
						Source:        "installed",
						InstallScope:  app.InstallScope,
						Notes:         pol.Notes,
					})
				}
			}
			for _, proc := range procs {
				if strings.Contains(strings.ToLower(proc), patLower) {
					audit.ForbiddenHits = append(audit.ForbiddenHits, PolicyViolation{
						PolicyPattern: pol.Pattern,
						Category:      pol.Category,
						MatchedName:   proc,
						Source:        "running_process",
						InstallScope:  "process",
						Notes:         pol.Notes,
					})
				}
			}
		case "required":
			if len(apps) > 0 {
				found := false
				for _, app := range apps {
					if strings.Contains(strings.ToLower(app.AppName), patLower) {
						found = true
						break
					}
				}
				if !found {
					audit.MissingRequired = append(audit.MissingRequired, pol.Pattern)
				}
			}
		}
	}

	hasViolation := len(audit.ForbiddenHits) > 0 || len(audit.MissingRequired) > 0
	hasLicenseIssue := false
	for _, lf := range audit.LicenseFindings {
		if lf.Flag != "ok" {
			hasLicenseIssue = true
			break
		}
	}
	hasSecurityWarning := len(audit.SecurityWarnings) > 0

	switch {
	case hasViolation || hasSecurityWarning:
		audit.ComplianceStatus = "violation"
	case hasLicenseIssue || (len(audit.AdminUsers) > 1):
		audit.ComplianceStatus = "warning"
	default:
		audit.ComplianceStatus = "compliant"
	}

	return audit
}

func getPolicyAudit() (*PolicyAuditResponse, error) {
	ctx := context.Background()

	servers, err := getAllServers()
	if err != nil {
		return nil, err
	}
	policies, err := getSoftwarePolicies()
	if err != nil {
		return nil, err
	}

	// Bulk fetch asset_info
	assetMap := make(map[string]*AssetInfo)
	aRows, err := dbPool.Query(ctx, `
		SELECT server_id,
		       COALESCE(device_manufacturer,''), COALESCE(device_model,''), COALESCE(bios_release_date,''), COALESCE(gpu_model,''),
		       COALESCE(processor_model,''), COALESCE(processor_cores,0), COALESCE(processor_clock_ghz,0),
		       COALESCE(disk_type,''), COALESCE(disk_size_gb,0), COALESCE(disk_health_status,''), disk_health_percent, COALESCE(disk_volumes,'[]'::jsonb),
		       COALESCE(ram_total_gb,0), COALESCE(ram_slots_used,0), COALESCE(ram_slots_total,0), COALESCE(ram_ddr_type,''),
		       COALESCE(ram_speed_mhz,0), COALESCE(ram_max_gb,0), COALESCE(ram_form_factor,''),
		       battery_health_percent,
		       COALESCE(serial_number,''), COALESCE(ip_address,''), COALESCE(mac_address,''), COALESCE(os_version,''),
		       COALESCE(licenses,'[]'::jsonb), COALESCE(security_status,'{}'::jsonb), COALESCE(running_processes,'[]'::jsonb), updated_at
		FROM asset_info`)
	if err != nil {
		return nil, err
	}
	for aRows.Next() {
		a, err := scanAssetInfoRow(aRows)
		if err != nil {
			aRows.Close()
			return nil, err
		}
		assetMap[a.ServerID] = a
	}
	aRows.Close()

	// Bulk fetch installed_applications
	appMap := make(map[string][]InstalledApp)
	appRows, err := dbPool.Query(ctx,
		`SELECT server_id, app_name, COALESCE(app_version,''), COALESCE(publisher,''), COALESCE(install_date,''), COALESCE(install_scope,'system'), reported_at
		 FROM installed_applications ORDER BY app_name ASC`)
	if err != nil {
		return nil, err
	}
	for appRows.Next() {
		var sid string
		var a InstalledApp
		if err := appRows.Scan(&sid, &a.AppName, &a.AppVersion, &a.Publisher, &a.InstallDate, &a.InstallScope, &a.ReportedAt); err != nil {
			appRows.Close()
			return nil, err
		}
		appMap[sid] = append(appMap[sid], a)
	}
	appRows.Close()

	// Bulk fetch device_users
	userMap := make(map[string][]DeviceUser)
	uRows, err := dbPool.Query(ctx,
		`SELECT server_id, username, COALESCE(full_name,''), privilege, is_disabled, is_logged_in, reported_at
		 FROM device_users ORDER BY username ASC`)
	if err != nil {
		return nil, err
	}
	for uRows.Next() {
		var u DeviceUser
		if err := uRows.Scan(&u.ServerID, &u.Username, &u.FullName, &u.Privilege, &u.IsDisabled, &u.IsLoggedIn, &u.ReportedAt); err != nil {
			uRows.Close()
			return nil, err
		}
		userMap[u.ServerID] = append(userMap[u.ServerID], u)
	}
	uRows.Close()

	resp := &PolicyAuditResponse{
		TotalDevices:     len(servers),
		Policies:         policies,
		LicenseSummaries: []LicensedSeatSummary{},
		Devices:          []DevicePolicyAudit{},
	}

	for _, srv := range servers {
		audit := evaluateDevicePolicy(srv, assetMap[srv.ServerID], appMap[srv.ServerID], userMap[srv.ServerID], policies)
		resp.TotalForbiddenHits += len(audit.ForbiddenHits)
		if len(audit.SecurityWarnings) > 0 {
			resp.WeakSecurityDevices++
		}
		for _, lf := range audit.LicenseFindings {
			if lf.Flag != "ok" {
				resp.SuspectLicenseDevices++
				break
			}
		}
		switch audit.ComplianceStatus {
		case "violation":
			resp.ViolationDevices++
		case "warning":
			resp.WarningDevices++
		default:
			resp.CompliantDevices++
		}
		resp.Devices = append(resp.Devices, audit)
	}

	sort.SliceStable(resp.Devices, func(i, j int) bool {
		rank := func(s string) int {
			switch s {
			case "violation":
				return 0
			case "warning":
				return 1
			default:
				return 2
			}
		}
		ri, rj := rank(resp.Devices[i].ComplianceStatus), rank(resp.Devices[j].ComplianceStatus)
		if ri != rj {
			return ri < rj
		}
		return resp.Devices[i].ServerID < resp.Devices[j].ServerID
	})

	// Hitung pemakaian kursi lisensi untuk policy bertipe 'licensed'
	for _, pol := range policies {
		if pol.PolicyType != "licensed" {
			continue
		}
		patLower := strings.ToLower(strings.TrimSpace(pol.Pattern))
		var installedOn []string
		for _, srv := range servers {
			for _, app := range appMap[srv.ServerID] {
				if strings.Contains(strings.ToLower(app.AppName), patLower) {
					installedOn = append(installedOn, srv.ServerID)
					break
				}
			}
		}
		if installedOn == nil {
			installedOn = []string{}
		}
		resp.LicenseSummaries = append(resp.LicenseSummaries, LicensedSeatSummary{
			Pattern:       pol.Pattern,
			Category:      pol.Category,
			LicensedSeats: pol.LicensedSeats,
			InstalledOn:   installedOn,
			OverAllocated: pol.LicensedSeats > 0 && len(installedOn) > pol.LicensedSeats,
		})
	}

	return resp, nil
}
