package main

import (
	"strings"
	"testing"
	"time"
)

func TestParseRangeSupportsDaysAndDurations(t *testing.T) {
	if d, err := parseRange(""); err != nil || d != time.Hour {
		t.Fatalf("empty range = (%v, %v), want (1h, nil)", d, err)
	}
	if d, err := parseRange("7d"); err != nil || d != 7*24*time.Hour {
		t.Fatalf("7d range = (%v, %v), want (168h, nil)", d, err)
	}
	if d, err := parseRange("30d"); err != nil || d != 30*24*time.Hour {
		t.Fatalf("30d range = (%v, %v), want (720h, nil)", d, err)
	}
	if _, err := parseRange("0d"); err == nil {
		t.Fatalf("0d should fail")
	}
}

func TestEvaluateUpgradeDevice(t *testing.T) {
	now := time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC)
	health := 40.0
	d := DeviceUpgradeInsight{
		ServerID:             "LT-FIN-01",
		DeviceType:           "laptop",
		BIOSReleaseDate:      "2020-05-10",
		DiskType:             "HDD",
		DiskSizeGB:           500,
		RAMTotalGB:           8,
		RAMSlotsUsed:         1,
		RAMSlotsTotal:        2,
		RAMDDRType:           "DDR4",
		RAMSpeedMHz:          2666,
		RAMFormFactor:        "SODIMM",
		AvgRAMPercent:        86,
		P95RAMPercent:        93,
		MetricSamples:        100,
		BatteryHealthPercent: &health,
	}

	res := evaluateUpgradeDevice(d, now)
	if res.Priority != "critical" {
		t.Fatalf("Priority = %q, want critical", res.Priority)
	}
	if res.DeviceAgeYears == nil || *res.DeviceAgeYears != 6 {
		t.Fatalf("DeviceAgeYears = %v, want 6", res.DeviceAgeYears)
	}

	var foundRAMAction string
	for _, r := range res.Recommendations {
		if r.Category == "ram" {
			foundRAMAction = r.Action
		}
	}
	if !strings.Contains(foundRAMAction, "DDR4 2666 MHz SODIMM") || !strings.Contains(foundRAMAction, "1 dari 2 slot kosong") {
		t.Fatalf("RAM action unexpected: %q", foundRAMAction)
	}
}

func TestEvaluateDevicePolicyDetectsPiracyAndViolations(t *testing.T) {
	srv := Server{ServerID: "PC-DESAIN-01", DeviceType: "pc"}
	asset := &AssetInfo{
		ServerID: "PC-DESAIN-01",
		Licenses: []LicenseInfo{
			{ProductName: "Windows 11 Pro", Family: "Windows", Channel: "VOLUME_KMSCLIENT", LicenseStatus: "Licensed", KMSMachine: "127.0.0.2"},
		},
		SecurityStatus: SecurityStatus{
			AntivirusEnabled:   true,
			RealtimeProtection: false,
			FirewallEnabled:    true,
			Reported:           true,
		},
		RunningProcesses: []string{"explorer.exe", "KMSpico.exe"},
	}
	apps := []InstalledApp{
		{AppName: "uTorrent Web", InstallScope: "user"},
		{AppName: "Adobe Photoshop 2024", InstallScope: "system"},
	}
	users := []DeviceUser{
		{Username: "Administrator", Privilege: "administrator"},
		{Username: "budi", Privilege: "administrator"},
	}
	policies := []SoftwarePolicy{
		{Pattern: "KMSpico", PolicyType: "forbidden", Category: "Crack / Activator"},
		{Pattern: "uTorrent", PolicyType: "forbidden", Category: "P2P / Torrent"},
		{Pattern: "FortiClient", PolicyType: "required", Category: "Security"},
	}

	audit := evaluateDevicePolicy(srv, asset, apps, users, policies)
	if audit.ComplianceStatus != "violation" {
		t.Fatalf("ComplianceStatus = %q, want violation", audit.ComplianceStatus)
	}
	if len(audit.ForbiddenHits) != 2 {
		t.Fatalf("ForbiddenHits len = %d, want 2 (%+v)", len(audit.ForbiddenHits), audit.ForbiddenHits)
	}
	if len(audit.MissingRequired) != 1 || audit.MissingRequired[0] != "FortiClient" {
		t.Fatalf("MissingRequired = %v, want [FortiClient]", audit.MissingRequired)
	}
	if len(audit.LicenseFindings) != 1 || audit.LicenseFindings[0].Flag != "suspect_kms" {
		t.Fatalf("LicenseFindings = %+v, want suspect_kms", audit.LicenseFindings)
	}
	if len(audit.SecurityWarnings) != 1 {
		t.Fatalf("SecurityWarnings = %v, want 1 warning", audit.SecurityWarnings)
	}
	if audit.UserScopeAppsCount != 1 {
		t.Fatalf("UserScopeAppsCount = %d, want 1", audit.UserScopeAppsCount)
	}
}
