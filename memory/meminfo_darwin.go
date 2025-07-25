//go:build darwin

package memory

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

// GetCachedMemory fetches the size of cached memory in kb
func GetCachedMemory(pageSize int64) (int64, error) {
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

	patternCached := regexp.MustCompile("File-backed pages: +([0-9]+).")
	for scanner.Scan() {
		text := scanner.Text()
		res := patternCached.FindStringSubmatch(text)
		if res == nil {
			continue
		}
		file_backed_pages, err := strconv.ParseInt(res[1], 10, 32)
		if err != nil {
			slog.Error("couldn't parse cached memory", "error", err)
			return 0, err
		}

		return file_backed_pages * pageSize / 1024, err
	}

	return 0, fmt.Errorf("cached memory not found")
}
