package main

import (
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	psnet "github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

type processTracker struct {
	mu   sync.Mutex
	seen map[string]string // lowercase -> display name
}

func newProcessTracker() *processTracker {
	return &processTracker{seen: make(map[string]string)}
}

func (pt *processTracker) Record(name string) {
	if pt == nil {
		return
	}
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return
	}
	key := strings.ToLower(trimmed)
	pt.mu.Lock()
	defer pt.mu.Unlock()
	if len(pt.seen) >= 500 {
		return
	}
	if _, exists := pt.seen[key]; !exists {
		pt.seen[key] = trimmed
	}
}

func (pt *processTracker) DrainSorted() []string {
	if pt == nil {
		return nil
	}
	pt.mu.Lock()
	out := make([]string, 0, len(pt.seen))
	for _, name := range pt.seen {
		out = append(out, name)
	}
	pt.seen = make(map[string]string)
	pt.mu.Unlock()
	sort.Strings(out)
	return out
}

func getCPU() float64 {
	percentages, err := cpu.Percent(1*time.Second, false)
	if err != nil {
		log.Printf("Gagal membaca CPU usage: %v", err)
		return 0
	}
	if len(percentages) > 0 {
		return percentages[0]
	}
	return 0
}

func getRAM() float64 {
	vmStat, err := mem.VirtualMemory()
	if err != nil {
		log.Printf("Gagal membaca RAM usage: %v", err)
		return 0
	}
	return vmStat.UsedPercent
}

func getDisk() float64 {
	diskStat, err := disk.Usage("C:\\")
	if err != nil {
		log.Printf("Gagal membaca Disk usage: %v", err)
		return 0
	}
	return diskStat.UsedPercent
}

// getNetworkUsage mengukur throughput network (bytes/sec) dengan mengambil
// 2 sample dengan jeda waktu tertentu, lalu menghitung selisihnya.
func getNetworkUsage() (bytesSentPerSec, bytesRecvPerSec float64) {
	before, err := psnet.IOCounters(false)
	if err != nil || len(before) == 0 {
		log.Printf("Gagal membaca network stat (sample 1): %v", err)
		return 0, 0
	}

	time.Sleep(1 * time.Second)

	after, err := psnet.IOCounters(false)
	if err != nil || len(after) == 0 {
		log.Printf("Gagal membaca network stat (sample 2): %v", err)
		return 0, 0
	}

	sentDiff := float64(after[0].BytesSent - before[0].BytesSent)
	recvDiff := float64(after[0].BytesRecv - before[0].BytesRecv)

	// Karena jeda 1 detik, hasil selisih ini sudah otomatis merepresentasikan bytes/detik
	return sentDiff, recvDiff
}

// checkServices mengecek apakah proses dengan nama tertentu sedang berjalan.
// Return map[nama_service]status (true = running, false = not running)
func checkServices(serviceNames []string) map[string]bool {
	return checkServicesAndTrack(serviceNames, nil)
}

func checkServicesAndTrack(serviceNames []string, tracker *processTracker) map[string]bool {
	result := make(map[string]bool)
	for _, name := range serviceNames {
		result[name] = false // default: dianggap tidak jalan
	}

	processes, err := process.Processes()
	if err != nil {
		log.Printf("Gagal membaca daftar proses: %v", err)
		return result
	}

	for _, p := range processes {
		name, err := p.Name()
		if err != nil {
			continue
		}
		tracker.Record(name)
		for _, target := range serviceNames {
			if strings.EqualFold(name, target) || strings.EqualFold(name, target+".exe") {
				result[target] = true
			}
		}
	}

	return result
}