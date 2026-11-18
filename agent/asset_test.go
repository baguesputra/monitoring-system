//go:build windows

package main

import "testing"

func TestRIDFromSID(t *testing.T) {
	cases := []struct {
		sid  string
		want int
	}{
		{"S-1-5-21-1234567890-1234567890-1234567890-1000", 1000},
		{"S-1-5-21-1234567890-1234567890-1234567890-1001", 1001},
		{"S-1-5-21-1234567890-1234567890-1234567890-500", 500},
		{"S-1-5-18", 18},
		{"", -1},
		{"not-a-sid", -1},
		{"S-1-5-21-abc-1000", 1000},
	}
	for _, c := range cases {
		if got := ridFromSID(c.sid); got != c.want {
			t.Errorf("ridFromSID(%q) = %d, want %d", c.sid, got, c.want)
		}
	}
}

func TestExtractWMIKeyValue(t *testing.T) {
	ref := `\\.\root\cimv2:Win32_Account.Domain="DESKTOP-92PUGLA",Name="IT 10"`
	if got := extractWMIKeyValue(ref, "Name"); got != "IT 10" {
		t.Errorf("Name = %q, want %q", got, "IT 10")
	}
	if got := extractWMIKeyValue(ref, "Domain"); got != "DESKTOP-92PUGLA" {
		t.Errorf("Domain = %q, want %q", got, "DESKTOP-92PUGLA")
	}
	if got := extractWMIKeyValue(ref, "Missing"); got != "" {
		t.Errorf("Missing = %q, want empty", got)
	}
	if got := extractWMIKeyValue(`Name="unterminated`, "Name"); got != "" {
		t.Errorf("unterminated = %q, want empty", got)
	}
}

func TestTrimDomainPrefix(t *testing.T) {
	cases := map[string]string{
		`DESKTOP-92PUGLA\IT 10`: "IT 10",
		`IT 10`:                "IT 10",
		`\\HOST\Administrators`: "Administrators",
		``:                     "",
	}
	for in, want := range cases {
		if got := trimDomainPrefix(in); got != want {
			t.Errorf("trimDomainPrefix(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseWMIDate(t *testing.T) {
	if got := parseWMIDate("20210819000000.000000+000"); got != "2021-08-19" {
		t.Errorf("parseWMIDate valid = %q, want 2021-08-19", got)
	}
	if got := parseWMIDate("bad"); got != "" {
		t.Errorf("parseWMIDate bad = %q, want empty", got)
	}
}

func TestClassifyDiskType(t *testing.T) {
	if got := classifyDiskType(17, 4, "Samsung PM9A1"); got != "NVMe SSD" {
		t.Errorf("NVMe = %q", got)
	}
	if got := classifyDiskType(11, 4, "Kingston"); got != "SSD" {
		t.Errorf("SATA SSD = %q", got)
	}
	if got := classifyDiskType(11, 3, "WDC WD10EZEX"); got != "HDD" {
		t.Errorf("HDD = %q", got)
	}
}

func TestMapDDRTypeAndFormFactor(t *testing.T) {
	if got := mapDDRType(30); got != "LPDDR4" {
		t.Errorf("LPDDR4 = %q", got)
	}
	if got := mapDDRType(35); got != "LPDDR5" {
		t.Errorf("LPDDR5 = %q", got)
	}
	if got := mapRAMFormFactor(12); got != "SODIMM" {
		t.Errorf("SODIMM = %q", got)
	}
}

func TestIsSystemSIDAndDeduplicateApps(t *testing.T) {
	if !isSystemSID(".DEFAULT") || !isSystemSID("S-1-5-18") || !isSystemSID("S-1-5-21-123_Classes") {
		t.Errorf("expected system/classes SIDs to be filtered")
	}
	if isSystemSID("S-1-5-21-123456-1001") {
		t.Errorf("user SID should not be filtered")
	}

	apps := deduplicateApps([]AppInfo{
		{Name: "Chrome", Version: "1.0", InstallScope: "system"},
		{Name: "chrome", Version: "1.0", InstallScope: "system"},
		{Name: "Zoom", Version: "5.0", InstallScope: "user"},
	})
	if len(apps) != 2 {
		t.Errorf("deduplicateApps len = %d, want 2", len(apps))
	}
}

func TestCollectAssetInfoSmoke(t *testing.T) {
	asset := collectAssetInfo("TEST-LOCAL", "pc")
	if asset.ServerID != "TEST-LOCAL" {
		t.Fatalf("ServerID = %q", asset.ServerID)
	}
	t.Logf("Model=%s %s, BIOS=%s, GPU=%s", asset.DeviceManufacturer, asset.DeviceModel, asset.BIOSReleaseDate, asset.GPUModel)
	t.Logf("CPU=%s, Disk=%s %.1fGB (SMART=%s %v), RAM=%.1fGB %s %dMHz %s (%d/%d max %.1fGB)",
		asset.ProcessorModel, asset.DiskType, asset.DiskSizeGB, asset.DiskHealthStatus, asset.DiskHealthPercent,
		asset.RAMTotalGB, asset.RAMDDRType, asset.RAMSpeedMHz, asset.RAMFormFactor, asset.RAMSlotsUsed, asset.RAMSlotsTotal, asset.RAMMaxGB)
	t.Logf("Apps=%d, Licenses=%d, Security=%+v, Volumes=%d, Procs=%d",
		len(asset.Applications), len(asset.Licenses), asset.SecurityStatus, len(asset.DiskVolumes), len(asset.RunningProcesses))
	if asset.ProcessorModel == "" || asset.RAMTotalGB <= 0 {
		t.Fatalf("expected hardware info populated, got CPU=%q RAM=%.2f", asset.ProcessorModel, asset.RAMTotalGB)
	}
	if len(asset.DiskVolumes) == 0 || len(asset.RunningProcesses) == 0 {
		t.Fatalf("expected volumes (%d) and running processes (%d)", len(asset.DiskVolumes), len(asset.RunningProcesses))
	}
}
