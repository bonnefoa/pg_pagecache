//go:build linux

package memory

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
)

func getValue(filePath string, startPattern string) (int64, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		text := scanner.Text()
		parts := strings.Fields(text)
		metricName := parts[0]
		if metricName != startPattern {
			continue
		}

		value, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			slog.Error("Couldn't parse value as int", "error", err)
			return 0, err
		}

		// meminfo provides memory in kB
		// We want to convert it in pages
		return value, err
	}

	return 0, fmt.Errorf("Pattern not found")
}

// GetCachedMemory fetches cached memory from /proc/memory. Returns memory in kb
func GetCachedMemory(pageSize int64) (int64, error) {
	// Check cgroupv2 first
	fileMem, err := getValue("/sys/fs/cgroup/memory.stat", "file")
	if err == nil {
		return fileMem / 1024, nil
	}

	// Check cgroupv1
	cacheMem, err := getValue("/sys/fs/cgroup/memory/memory.stat", "cache")
	if err == nil {
		return cacheMem / 1024, nil
	}

	// Fallback to meminfo
	meminfoVal, err := getValue("/proc/meminfo", "Cached:")
	return meminfoVal, err
}
