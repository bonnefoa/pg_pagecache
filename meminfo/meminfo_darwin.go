//go:build darwin

package meminfo

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"regexp"
	"strconv"
	"time"
)

// GetCachedMemory fetches the size of cached memory in pages
func GetCachedMemory(page_size int64) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "vm_stat")
	out, err := cmd.StdoutPipe()
	if err != nil {
		return 0, err
	}

	if err := cmd.Start(); err != nil {
		return 0, err
	}

	scanner := bufio.NewScanner(out)

	pattern_cached := regexp.MustCompile("File-backed pages: +([0-9]+).")
	for scanner.Scan() {
		text := scanner.Text()
		res := pattern_cached.FindStringSubmatch(text)
		if res == nil {
			continue
		}
		file_backed_pages, err := strconv.ParseInt(res[1], 10, 32)
		if err != nil {
			slog.Error("Couldn't parse cached memory", "error", err)
			return 0, err
		}

		return file_backed_pages / 1024, err
	}

	return 0, fmt.Errorf("Cached memory not found")
}
