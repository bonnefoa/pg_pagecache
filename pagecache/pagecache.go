package pagecache

import (
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"strconv"
	"unsafe"

	"syscall"

	"golang.org/x/sys/unix"
)

type PageFlags struct {
	Flags uint64
	Count int
}

// PageStats stores page cache information
type PageStats struct {
	PageCached int
	PageCount  int

	PageFlagsMap map[uint64]PageFlags
}

// State stores state for page cache related functions
type State struct {
	rawFlags         bool
	pagemapFile      *os.File
	kpageFlagsFile   *os.File
	CanReadPageFlags bool
}

// Add adds stats from provided pageStats
func (p *PageStats) Add(b PageStats) {
	p.PageCount += b.PageCount
	p.PageCached += b.PageCached

	if p.PageFlagsMap == nil {
		p.PageFlagsMap = make(map[uint64]PageFlags)
	}
	for flags, v := range b.PageFlagsMap {
		pfs, ok := p.PageFlagsMap[flags]
		if !ok {
			pfs.Flags = flags
		}
		pfs.Count += v.Count
		p.PageFlagsMap[flags] = pfs
	}
}

// GetCachedPct returns the percent of cached pages as a string
func (p *PageStats) GetCachedPct() string {
	if p.PageCached > 0 {
		value := 100 * float64(p.PageCached) / float64(p.PageCount)
		return strconv.FormatFloat(value, 'f', 2, 64)
	}
	return "0"
}

// GetTotalCachedPct returns the percent of total cached pages as a string
// totalCached is in KB
func (p *PageStats) GetTotalCachedPct(pageSize int64, totalCached int64) string {
	if p.PageCached > 0 && totalCached > 0 {
		value := 100 * (float64(p.PageCached) * float64(pageSize) / 1024) / float64(totalCached)
		return strconv.FormatFloat(value, 'f', 2, 64)
	}
	return "0"
}

// GetPageSize returns the os page size
func GetPageSize() int64 {
	return int64(os.Getpagesize())
}

// readInt64SliceFromFile reads int64 elements from a file. Size and index are in int64 elements, not in bytes
func readInt64SliceFromFile(f *os.File, size int, index int64) ([]uint64, error) {
	buf := make([]byte, 8*size)
	n, err := f.ReadAt(buf, index*8)
	if n != len(buf) || err != nil {
		return nil, fmt.Errorf("Error reading pagemap: %v", err)
	}

	// Convert []byte to []uint64
	const ui64Size = int(unsafe.Sizeof(uint64(0)))
	ui64Ptr := (*uint64)(unsafe.Pointer(unsafe.SliceData(buf)))
	ui64Len := len(buf) / ui64Size
	return unsafe.Slice(ui64Ptr, ui64Len), nil
}

func (s *State) getPagecacheStats(fd int, fileSize int64, pageSize int64) (PageStats, error) {
	var mmap []byte
	pageStats := PageStats{0, 0, make(map[uint64]PageFlags, 0)}
	// void *mmap(void addr[.length], size_t length, int prot, int flags, int fd, off_t offset);
	mmap, err := unix.Mmap(fd, 0, int(fileSize), unix.PROT_READ, unix.MAP_SHARED)
	if err != nil {
		return pageStats, fmt.Errorf("Error while mmaping: %v", err)
	}
	defer unix.Munmap(mmap)

	// Mincore signature:
	// int mincore(void addr[.length], size_t length, unsigned char *vec);
	// Build the result vec. From mincore doc: The vec argument must point to an
	// array containing at least (length+PAGE_SIZE-1) / PAGE_SIZE bytes
	numPages := (fileSize + pageSize - 1) / pageSize
	vec := make([]byte, numPages)

	mmapPtr := uintptr(unsafe.Pointer(&mmap[0]))
	fileSizePtr := uintptr(fileSize)
	vecPtr := uintptr(unsafe.Pointer(&vec[0]))

	ret, _, err := syscall.Syscall(syscall.SYS_MINCORE, mmapPtr, fileSizePtr, vecPtr)
	if ret != 0 {
		return pageStats, fmt.Errorf("syscall SYS_MINCORE failed: %v", err)
	}

	pageStats.PageCount = len(vec)
	pageStats.PageCached = 0
	cachedPageIndex := 0
	for i, v := range vec {
		// On return, the least significant bit of each byte will be set if the corresponding page is currently resident in memory, and be clear otherwise
		if v&0x1 > 0 {
			pageStats.PageCached = pageStats.PageCached + 1
			cachedPageIndex = i
		}
	}

	if s.CanReadPageFlags && pageStats.PageCached > 0 {
		err = s.populatePTE(mmapPtr, fileSizePtr, vec, pageSize)
		if err != nil {
			return pageStats, err
		}
		pagemapFlags, err := s.readPageMap(mmapPtr, int(numPages), pageSize)
		if err != nil {
			return pageStats, err
		}
		if pagemapFlags[cachedPageIndex]&PFN_MASK == 0 {
			slog.Info("Can't read Page Frame Numbers, CAP_SYS_ADMIN may be missing. Page Flags won't be displayed.")
			s.CanReadPageFlags = false
			return pageStats, err
		}
		// Make sure to unmap before reading kpageflags
		unix.Munmap(mmap)
		flagsCount, err := s.readKpageFlags(pagemapFlags)
		if err != nil {
			return pageStats, err
		}
		for flags, flagsCount := range flagsCount {
			pfs, ok := pageStats.PageFlagsMap[flags]
			if !ok {
				pfs = PageFlags{flags, 0}
			}
			pfs.Count += flagsCount
			pageStats.PageFlagsMap[flags] = pfs
		}
	}

	return pageStats, nil
}

// NewPageCacheState creates a new pagecache state
func NewPageCacheState(rawFlags bool) (state State) {
	state.rawFlags = rawFlags
	state.CanReadPageFlags = false
	if runtime.GOOS != "linux" {
		// Nothing to do
		return
	}

	var err error
	mode := os.FileMode(0600)
	state.pagemapFile, err = os.OpenFile("/proc/self/pagemap", os.O_RDONLY, mode)
	if err != nil {
		slog.Info("Error opening /proc/self/pagemap, page flags won't be available", "err", err)
		return
	}
	state.kpageFlagsFile, err = os.OpenFile("/proc/kpageflags", os.O_RDONLY, mode)
	if err != nil {
		state.pagemapFile = nil
		slog.Info("Error opening /proc/kpageflags, page flags won't be available", "err", err)
		return
	}

	// Assume true at this point. This may be switched to false if pfn are all 0
	// which means we don't have CAP_SYS_ADMIN. It could be replaced by a capabilities
	// check but this requires dedicated linkage options. In the end, it's simpler to
	// try and check pfn's values
	state.CanReadPageFlags = true
	return
}

// GetPageCacheInfo returns the page cache stats for the provided file
func (s *State) GetPageCacheInfo(fullPath string, pagesize int64) (PageStats, error) {
	pageStats := PageStats{0, 0, make(map[uint64]PageFlags, 0)}
	file, err := os.Open(fullPath)
	if err != nil {
		return pageStats, fmt.Errorf("Error opening file %s: %v", fullPath, err)
	}
	defer file.Close()
	fileInfo, err := file.Stat()
	if err != nil {
		return pageStats, fmt.Errorf("Error getting file stat %s: %v", fullPath, err)
	}
	fileSize := fileInfo.Size()
	if fileSize == 0 {
		return pageStats, nil
	}
	pageStats, err = s.getPagecacheStats(int(file.Fd()), fileInfo.Size(), pagesize)
	if err != nil {
		return pageStats, fmt.Errorf("Getting pagecache stats for %s failed: %v", fullPath, err)
	}
	return pageStats, nil
}
