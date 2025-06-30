//go:build linux

package meminfo

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
)

// GetCachedMemory fetches cached memory from /proc/memory. Returns memory in pages
func GetCachedMemory(page_size int64) (int64, error) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		text := scanner.Text()
		// Looking for 'Cached:         236072800 kB'
		parts := strings.Split(text, ":")
		metric_name := parts[0]
		if metric_name != "Cached" {
			continue
		}

		valueStr := strings.Split(strings.Trim(parts[1], " "), " ")[0]
		value, err := strconv.ParseInt(valueStr, 10, 32)
		if err != nil {
			slog.Error("Couldn't parse Cached memory", "error", err)
			return 0, err
		}

		// meminfo provides memory in kB
		// We want to convert it in pages
		return value / (page_size / 1024), err
	}

	return 0, fmt.Errorf("Cached memory not found")
}
