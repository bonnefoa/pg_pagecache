package pcstats

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

type PageCacheInfo struct {
	PageCached int
	PageCount  int
}

type PcState struct {
	pagemapFile *os.File
}

func (p *PageCacheInfo) Add(b PageCacheInfo) {
	p.PageCount += b.PageCount
	p.PageCached += b.PageCached
}

func (p *PageCacheInfo) GetCachedPct() string {
	if p.PageCached > 0 {
		value := 100 * float64(p.PageCached) / float64(p.PageCount)
		return strconv.FormatFloat(value, 'f', 2, 64)
	}
	return "0"
}

func (p *PageCacheInfo) GetTotalCachedPct(totalCached int64) string {
	if p.PageCached > 0 && totalCached > 0 {
		value := 100 * float64(p.PageCached) / float64(totalCached)
		return strconv.FormatFloat(value, 'f', 2, 64)
	}
	return "0"
}

func GetPageSize() int64 {
	return int64(os.Getpagesize())
}

func (p *PcState) getActivePages(mmapPtr uintptr, fileSizePtr uintptr, vec []byte, pageSize int64) (err error) {
	ret, _, err := syscall.Syscall(syscall.SYS_MADVISE, mmapPtr, fileSizePtr, unix.MADV_RANDOM)
	if ret != 0 {
		return fmt.Errorf("syscall MADVISE failed: %v", err)
	}

	// Populate PTE
	for i, v := range vec {
		if v&0x1 > 0 {
			_ = (*byte)(unsafe.Pointer(mmapPtr + uintptr(i)*uintptr(pageSize)))
		}
	}

	ret, _, err = syscall.Syscall(syscall.SYS_MADVISE, mmapPtr, fileSizePtr, unix.MADV_SEQUENTIAL)
	if ret != 0 {
		return fmt.Errorf("syscall MADVISE failed: %v", err)
	}

	// One int64 per page
	buf := make([]byte, len(vec)*8)
	offset := int64(mmapPtr) / pageSize
	n, err := p.pagemapFile.ReadAt(buf, offset*8)
	if n != len(buf) || err != nil {
		return fmt.Errorf("Error reading pagemap: %v", err)
	}

	const i64Size = int(unsafe.Sizeof(int64(0)))
	i64Ptr := (*int64)(unsafe.Pointer(unsafe.SliceData(buf)))
	i64Len := len(buf) / i64Size
	i64 := unsafe.Slice(i64Ptr, i64Len)

	slog.Info("buf", "buf", buf)
	for i, v := range vec {
		if v&0x1 > 0 {
			slog.Info("I64 value", "offset", offset, "i", i, "i64", i64[i])
		}
	}

	return nil
}

func (p *PcState) getPagecacheStats(fd int, fileSize int64, pageSize int64) (PageCacheInfo, error) {
	var mmap []byte
	pcStats := PageCacheInfo{}
	// void *mmap(void addr[.length], size_t length, int prot, int flags, int fd, off_t offset);
	mmap, err := unix.Mmap(fd, 0, int(fileSize), unix.PROT_READ, unix.MAP_SHARED)
	if err != nil {
		return pcStats, fmt.Errorf("Error while mmaping: %v", err)
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

	if runtime.GOOS == "linux" {
		err = p.getActivePages(mmapPtr, fileSizePtr, vec, pageSize)
	}

	return pcStats, err
}

func NewPcState() (pcState PcState, err error) {
	if runtime.GOOS != "linux" {
		// Nothing to do
		return
	}
	c := cap.GetProc()
	if on, err := c.GetFlag(cap.Permitted, cap.SETUID); err != nil {
		fmt.Printf("unable to determine cap_setuid permitted flag value: %v\n", err)
		return
	} else if !on {
		fmt.Println("no permitted capability, try: sudo setcap cap_setuid=p program")
		return
	}

	pcState.pagemapFile, err = os.Open("/proc/self/pagemap")
	return
}

func (p *PcState) GetPcStats(fullPath string, pagesize int64) (PageCacheInfo, error) {
	pcStats := PageCacheInfo{}
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
	pcStats, err = p.getPagecacheStats(int(file.Fd()), fileInfo.Size(), pagesize)
	if err != nil {
		return pcStats, fmt.Errorf("Getting pagecache stats for %s failed: %v", fullPath, err)
	}
	return pcStats, nil
}
