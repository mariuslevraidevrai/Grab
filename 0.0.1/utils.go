package main

import (
	"fmt"
	"time"
)

func formatBytes(bytes int64) string {
	const unit = 1000
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func printSummary(filePath string, fileSize int64, duration time.Duration, speed float64) {
	fmt.Println("\n" + greenText("----------------------------------------"))
	fmt.Println(greenText("[+]") + " Download completed successfully!")
	fmt.Println(greenText("----------------------------------------"))
	fmt.Printf("  Saved to  : %s\n", filePath)
	fmt.Printf("  Size      : %s (%d bytes)\n", formatBytes(fileSize), fileSize)
	fmt.Printf("  Time      : %.2fs\n", duration.Seconds())
	fmt.Printf("  Avg speed : %.2f MB/s\n", speed)
	fmt.Println(greenText("----------------------------------------"))
}