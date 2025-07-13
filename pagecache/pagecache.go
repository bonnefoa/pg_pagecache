package pagecache

import (
	"encoding/binary"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"unsafe"

	"syscall"

	"golang.org/x/sys/unix"
)

// Kernel Page cache flags
const (
	KpfReferenced        = 1 << 2
	KpfUptodate          = 1 << 3
	KpfDirty             = 1 << 4
	KpfLRU               = 1 << 5
	KpfActive            = 1 << 6
	KpfWriteback         = 1 << 8
	KpfHackerBits uint64 = 0xffff << 32
)

// PageStats stores page cache information
type PageStats struct {
	PageCached int
	PageCount  int

	PageFlags map[uint64]int
}

// State stores state for page cache related functions
type State struct {
	pagemapFile *os.File
	kpageFlags  *os.File
}

func (p *PageStats) Add(b PageStats) {
	p.PageCount += b.PageCount
	p.PageCached += b.PageCached

	if p.PageFlags == nil {
		p.PageFlags = make(map[uint64]int)
	}
	for k, v := range b.PageFlags {
		p.PageFlags[k] += v
	}
}

func (p *PageStats) GetCachedPct() string {
	if p.PageCached > 0 {
		value := 100 * float64(p.PageCached) / float64(p.PageCount)
		return strconv.FormatFloat(value, 'f', 2, 64)
	}
	return "0"
}

func (p *PageStats) GetTotalCachedPct(totalCached int64) string {
	if p.PageCached > 0 && totalCached > 0 {
		value := 100 * float64(p.PageCached) / float64(totalCached)
		return strconv.FormatFloat(value, 'f', 2, 64)
	}
	return "0"
}

func GetPageSize() int64 {
	return int64(os.Getpagesize())
}

func (p *State) getActivePages(pageStats *PageStats, mmapPtr uintptr, fileSizePtr uintptr, vec []byte, pageSize int64) (err error) {
	ret, _, err := syscall.Syscall(syscall.SYS_MADVISE, mmapPtr, fileSizePtr, unix.MADV_RANDOM)
	if ret != 0 {
		return fmt.Errorf("syscall MADVISE failed: %v", err)
	}

	// Populate PTE
	for i, v := range vec {
		if v&0x1 > 0 {
			_ = *(*byte)(unsafe.Pointer(mmapPtr + uintptr(int64(i)*pageSize)))
		}
	}

	ret, _, err = syscall.Syscall(syscall.SYS_MADVISE, mmapPtr, fileSizePtr, unix.MADV_SEQUENTIAL)
	if ret != 0 {
		return fmt.Errorf("syscall MADVISE failed: %v", err)
	}

	numPages := len(vec)

	// One int64 per page
	buf := make([]byte, numPages*8)
	offset := (int64(mmapPtr) / pageSize) * 8
	n, err := p.pagemapFile.ReadAt(buf, offset)
	if n != len(buf) || err != nil {
		return fmt.Errorf("Error reading pagemap: %v", err)
	}

	const i64Size = int(unsafe.Sizeof(int64(0)))
	i64Ptr := (*int64)(unsafe.Pointer(unsafe.SliceData(buf)))
	i64Len := len(buf) / i64Size
	i64 := unsafe.Slice(i64Ptr, i64Len)

	for _, f := range i64 {
		pfn := f & 0x7FFFFFFFFFFFFF
		if pfn == 0 {
			continue
		}

		kbuf := make([]byte, 8)
		_, err = p.kpageFlags.ReadAt(kbuf, int64(pfn)*8)
		if err != nil {
			return err
		}
		flags := binary.LittleEndian.Uint64(kbuf) & ^KpfHackerBits
		pageStats.PageFlags[flags]++
	}

	return nil
}

func (p *State) getPagecacheStats(fd int, fileSize int64, pageSize int64) (PageStats, error) {
	var mmap []byte
	pageStats := PageStats{0, 0, make(map[uint64]int, 0)}
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
	pageNumber := (fileSize + pageSize - 1) / pageSize
	vec := make([]byte, pageNumber)

	mmapPtr := uintptr(unsafe.Pointer(&mmap[0]))
	fileSizePtr := uintptr(fileSize)
	vecPtr := uintptr(unsafe.Pointer(&vec[0]))

	ret, _, err := syscall.Syscall(syscall.SYS_MINCORE, mmapPtr, fileSizePtr, vecPtr)
	if ret != 0 {
		return pageStats, fmt.Errorf("syscall SYS_MINCORE failed: %v", err)
	}

	pageStats.PageCount = len(vec)
	pageStats.PageCached = 0
	for _, v := range vec {
		// On return, the least significant bit of each byte will be set if the corresponding page is currently resident in memory, and be clear otherwise
		if v&0x1 > 0 {
			pageStats.PageCached = pageStats.PageCached + 1
		}
	}

	if runtime.GOOS == "linux" {
		err = p.getActivePages(&pageStats, mmapPtr, fileSizePtr, vec, pageSize)
		if err != nil {
			return pageStats, err
		}
	}

	return pageStats, nil
}

func NewPageCacheState() (state State, err error) {
	if runtime.GOOS != "linux" {
		// Nothing to do
		return
	}

	mode := os.FileMode(0600)
	state.pagemapFile, err = os.OpenFile("/proc/self/pagemap", os.O_RDONLY, mode)
	if err != nil {
		return
	}
	state.kpageFlags, err = os.OpenFile("/proc/kpageflags", os.O_RDONLY, mode)
	if err != nil {
		return
	}
	return
}

func (p *State) GetPageCacheInfo(fullPath string, pagesize int64) (PageStats, error) {
	pageStats := PageStats{0, 0, make(map[uint64]int, 0)}
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
	pageStats, err = p.getPagecacheStats(int(file.Fd()), fileInfo.Size(), pagesize)
	if err != nil {
		return pageStats, fmt.Errorf("Getting pagecache stats for %s failed: %v", fullPath, err)
	}
	return pageStats, nil
}
