package metrics

import (
	"fmt"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

type SystemStats struct {
	CPUUsage     float64
	CPUCores     int
	MemTotal     uint64
	MemUsed      uint64
	MemUsage     float64
	DiskTotal    uint64
	DiskUsed     uint64
	DiskUsage    float64
	NetRx        uint64
	NetTx        uint64
	NetRxSpeed   float64 // KB/s
	NetTxSpeed   float64 // KB/s
}

const historySize = 60

var (
	lastNetRx uint64
	lastNetTx uint64
	lastTime  time.Time
	mu        sync.Mutex
	
	historyMutex sync.RWMutex
	historyIndex int
	historyList  [historySize]*SystemStats
	cores        int
)

func init() {
	count, err := cpu.Counts(true)
	if err == nil {
		cores = count
	} else {
		cores = 1
	}
}

func formatMB(bytes uint64) float64 {
	return float64(bytes) / 1024 / 1024
}

// GetStats compiles current system metrics.
func GetStats() *SystemStats {
	stats := &SystemStats{
		CPUCores: cores,
	}

	// CPU
	pcts, err := cpu.Percent(0, false)
	if err == nil && len(pcts) > 0 {
		stats.CPUUsage = pcts[0]
	}

	// Memory
	v, err := mem.VirtualMemory()
	if err == nil {
		stats.MemTotal = v.Total
		stats.MemUsed = v.Used
		stats.MemUsage = v.UsedPercent
	}

	// Disk
	d, err := disk.Usage("/")
	if err == nil {
		stats.DiskTotal = d.Total
		stats.DiskUsed = d.Used
		stats.DiskUsage = d.UsedPercent
	}

	// Network
	mu.Lock()
	defer mu.Unlock()

	netStats, err := net.IOCounters(false)
	if err == nil && len(netStats) > 0 {
		currentRx := netStats[0].BytesRecv
		currentTx := netStats[0].BytesSent
		now := time.Now()

		if !lastTime.IsZero() {
			duration := now.Sub(lastTime).Seconds()
			if duration > 0 {
				rxDiff := currentRx - lastNetRx
				txDiff := currentTx - lastNetTx
				stats.NetRxSpeed = float64(rxDiff) / duration / 1024 // KB/s
				stats.NetTxSpeed = float64(txDiff) / duration / 1024 // KB/s
			}
		}

		stats.NetRx = currentRx
		stats.NetTx = currentTx
		lastNetRx = currentRx
		lastNetTx = currentTx
		lastTime = now
	}

	// Add to history
	historyMutex.Lock()
	historyList[historyIndex] = stats
	historyIndex = (historyIndex + 1) % historySize
	historyMutex.Unlock()

	return stats
}

// GeneratePoints creates a space-separated string of points for an SVG polyline.
// maxVal is the highest expected value (e.g., 100 for percentages).
func GeneratePoints(width, height, maxVal float64, picker func(*SystemStats) float64) string {
	historyMutex.RLock()
	defer historyMutex.RUnlock()

	// Reconstruct chronological history
	ordered := make([]float64, 0, historySize)
	for i := 0; i < historySize; i++ {
		idx := (historyIndex + i) % historySize
		if historyList[idx] != nil {
			ordered = append(ordered, picker(historyList[idx]))
		}
	}

	if len(ordered) == 0 {
		return ""
	}

	step := width / float64(historySize-1)
	if len(ordered) == 1 {
		return fmt.Sprintf("0,%.2f %f,%.2f", height-(ordered[0]/maxVal)*height, width, height-(ordered[0]/maxVal)*height)
	}

	var points string
	for i, val := range ordered {
		x := float64(i) * step
		// Clamp value to maxVal
		if val > maxVal {
			val = maxVal
		}
		// Clamp value to non-negative
		if val < 0 {
			val = 0
		}
		y := height - (val/maxVal)*height
		points += fmt.Sprintf("%.2f,%.2f ", x, y)
	}
	return points
}

// FormatStats provides pre-formatted strings for easy HTML injection.
type FormattedStats struct {
	CPU          string
	CPUCores     int
	Mem          string
	MemPct       string
	Disk         string
	DiskPct      string
	NetRx        string
	NetTx        string
	CPUPoints    string
	MemPoints    string
	NetRxPoints  string
	NetTxPoints  string
}

func GetFormatted() FormattedStats {
	s := GetStats()
	
	// Pre-generate SVG points for 100x30 default viewboxes
	return FormattedStats{
		CPU:      fmt.Sprintf("%.1f%%", s.CPUUsage),
		CPUCores: s.CPUCores,
		Mem:      fmt.Sprintf("%.2f GB / %.2f GB", formatMB(s.MemUsed)/1024, formatMB(s.MemTotal)/1024),
		MemPct:   fmt.Sprintf("%.1f%%", s.MemUsage),
		Disk:     fmt.Sprintf("%.2f GB / %.2f GB", formatMB(s.DiskUsed)/1024, formatMB(s.DiskTotal)/1024),
		DiskPct:  fmt.Sprintf("%.1f%%", s.DiskUsage),
		NetRx:    fmt.Sprintf("%.2f KB/s", s.NetRxSpeed),
		NetTx:    fmt.Sprintf("%.2f KB/s", s.NetTxSpeed),
		CPUPoints: GeneratePoints(100, 30, 100, func(st *SystemStats) float64 { return st.CPUUsage }),
		MemPoints: GeneratePoints(100, 30, 100, func(st *SystemStats) float64 { return st.MemUsage }),
		// Assuming max 100 MB/s for scale representation on dashboard
		NetRxPoints: GeneratePoints(100, 30, 100, func(st *SystemStats) float64 { return st.NetRxSpeed }),
		NetTxPoints: GeneratePoints(100, 30, 100, func(st *SystemStats) float64 { return st.NetTxSpeed }),
	}
}
