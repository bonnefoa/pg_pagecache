package pcstats

import (
	"fmt"
	"os"
	"strconv"
	"unsafe"

	"syscall"

	"golang.org/x/sys/unix"
)

type PcStats struct {
	PageCached int
	PageCount  int
}

func (p *PcStats) Add(b PcStats) {
	p.PageCount += b.PageCount
	p.PageCached += b.PageCached
}

func GetPageSize() int64 {
	return int64(os.Getpagesize())
}

func getPagecacheStats(fd int, size int64, pageSize int64) (PcStats, error) {
	var mmap []byte
	pcStats := PcStats{}
	// void *mmap(void addr[.length], size_t length, int prot, int flags, int fd, off_t offset);
	mmap, err := unix.Mmap(fd, 0, int(size), unix.PROT_NONE, unix.MAP_SHARED)
	if err != nil {
		return pcStats, fmt.Errorf("Error while mmaping: %v", err)
	}
	defer unix.Munmap(mmap)

	// Mincore signature:
	// int mincore(void addr[.length], size_t length, unsigned char *vec);

	// Build the result vec. From mincore doc: The vec argument must point to an
	// array containing at least (length+PAGE_SIZE-1) / PAGE_SIZE bytes
	vecSize := (size + pageSize - 1) / pageSize
	vec := make([]byte, vecSize)

	mmapPtr := uintptr(unsafe.Pointer(&mmap[0]))
	sizePtr := uintptr(size)
	vecPtr := uintptr(unsafe.Pointer(&vec[0]))

	ret, _, err := syscall.Syscall(syscall.SYS_MINCORE, mmapPtr, sizePtr, vecPtr)
	if ret != 0 {
		return pcStats, fmt.Errorf("syscall SYS_MINCORE failed: %v", err)
	}

	pcStats.PageCount = len(vec)
	pcStats.PageCached = 0
	for _, v := range vec {
		// On return, the least significant bit of each byte will be set if the corresponding page is currently resident in memory, and be clear otherwise
		if v&0x1 > 0 {
			pcStats.PageCached = pcStats.PageCached + 1
		}
	}

	return pcStats, nil
}

func (p *PcStats) GetCachedPct() string {
	if p.PageCached > 0 {
		value := 100 * float64(p.PageCached) / float64(p.PageCount)
		return strconv.FormatFloat(value, 'f', 2, 64)
	}
	return "0"
}

func (p *PcStats) GetTotalCachedPct(totalCached int64) string {
	if p.PageCached > 0 && totalCached > 0 {
		value := 100 * float64(p.PageCached) / float64(totalCached)
		return strconv.FormatFloat(value, 'f', 2, 64)
	}
	return "0"
}

func GetPcStats(fullPath string, pagesize int64) (PcStats, error) {
	pcStats := PcStats{}
	file, err := os.Open(fullPath)
	if err != nil {
		return pcStats, fmt.Errorf("Error opening file %s: %v", fullPath, err)
	}
	defer file.Close()
	fileInfo, err := file.Stat()
	if err != nil {
		return pcStats, fmt.Errorf("Error getting file stat %s: %v", fullPath, err)
	}
	fileSize := fileInfo.Size()
	if fileSize == 0 {
		return pcStats, nil
	}
	pcStats, err = getPagecacheStats(int(file.Fd()), fileInfo.Size(), pagesize)
	if err != nil {
		return pcStats, fmt.Errorf("Getting pagecache stats for %s failed: %v", fullPath, err)
	}
	return pcStats, nil
}
