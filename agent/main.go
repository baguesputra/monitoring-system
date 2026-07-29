package main

import (
	"fmt"
	"log"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
)

func main() {
	log.Println("Monitoring agent starting...")

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		collectCPU()
	}
}

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