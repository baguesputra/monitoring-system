package main

import (
	"log"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	psnet "github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

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
		for _, target := range serviceNames {
			if name == target || name == target+".exe" {
				result[target] = true
			}
		}
	}

	return result
}