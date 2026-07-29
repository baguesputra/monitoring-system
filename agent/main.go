package main

import (
	"fmt"
	"log"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
)

func main() {
	log.Println("Monitoring agent starting...")

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		collectCPU()
		collectRAM()
		collectDisk()
		fmt.Println("---")
	}
}

// collectCPU membaca persentase penggunaan CPU saat ini
func collectCPU() {
	percentages, err := cpu.Percent(1*time.Second, false)
	if err != nil {
		log.Printf("Gagal membaca CPU usage: %v", err)
		return
	}

	if len(percentages) > 0 {
		fmt.Printf("[%s] CPU Usage: %.2f%%\n", time.Now().Format("15:04:05"), percentages[0])
	}
}

// collectRAM membaca penggunaan memory (RAM) saat ini
func collectRAM() {
	vmStat, err := mem.VirtualMemory()
	if err != nil {
		log.Printf("Gagal membaca RAM usage: %v", err)
		return
	}

	fmt.Printf("[%s] RAM Usage: %.2f%% (Used: %.2f GB / Total: %.2f GB)\n",
		time.Now().Format("15:04:05"),
		vmStat.UsedPercent,
		bytesToGB(vmStat.Used),
		bytesToGB(vmStat.Total),
	)
}

// collectDisk membaca penggunaan disk pada drive utama (C:)
func collectDisk() {
	diskStat, err := disk.Usage("C:\\")
	if err != nil {
		log.Printf("Gagal membaca Disk usage: %v", err)
		return
	}

	fmt.Printf("[%s] Disk Usage (C:): %.2f%% (Used: %.2f GB / Total: %.2f GB)\n",
		time.Now().Format("15:04:05"),
		diskStat.UsedPercent,
		bytesToGB(diskStat.Used),
		bytesToGB(diskStat.Total),
	)
}

// bytesToGB adalah helper untuk konversi bytes ke gigabytes agar mudah dibaca
func bytesToGB(bytes uint64) float64 {
	return float64(bytes) / 1024 / 1024 / 1024
}