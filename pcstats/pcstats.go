package pcstats

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

const (
	KPF_REFERENCED          = 1 << 2
	KPF_UPTODATE            = 1 << 3
	KPF_DIRTY               = 1 << 4
	KPF_LRU                 = 1 << 5
	KPF_ACTIVE              = 1 << 6
	KPF_WRITEBACK           = 1 << 8
	KPF_HACKERS_BITS uint64 = 0xffff << 32
)

type PageCacheInfo struct {
	PageCached int
	PageCount  int

	PageFlags map[uint64]int
}

type PcState struct {
	pagemapFile *os.File
	kpageFlags  *os.File
}

func (p *PageCacheInfo) Add(b PageCacheInfo) {
	p.PageCount += b.PageCount
	p.PageCached += b.PageCached

	if p.PageFlags == nil {
		p.PageFlags = make(map[uint64]int)
	}
	for k, v := range b.PageFlags {
		p.PageFlags[k] += v
	}
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

func (p *PcState) getActivePages(pageCacheInfo *PageCacheInfo, mmapPtr uintptr, fileSizePtr uintptr, vec []byte, pageSize int64) (err error) {
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
		flags := binary.LittleEndian.Uint64(kbuf) & ^KPF_HACKERS_BITS
		pageCacheInfo.PageFlags[flags]++
	}

	return nil
}

func (p *PcState) getPagecacheStats(fd int, fileSize int64, pageSize int64) (PageCacheInfo, error) {
	var mmap []byte
	pageCacheInfo := PageCacheInfo{0, 0, make(map[uint64]int, 0)}
	// void *mmap(void addr[.length], size_t length, int prot, int flags, int fd, off_t offset);
	mmap, err := unix.Mmap(fd, 0, int(fileSize), unix.PROT_READ, unix.MAP_SHARED)
	if err != nil {
		return pageCacheInfo, fmt.Errorf("Error while mmaping: %v", err)
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
		return pageCacheInfo, fmt.Errorf("syscall SYS_MINCORE failed: %v", err)
	}

	pageCacheInfo.PageCount = len(vec)
	pageCacheInfo.PageCached = 0
	for _, v := range vec {
		// On return, the least significant bit of each byte will be set if the corresponding page is currently resident in memory, and be clear otherwise
		if v&0x1 > 0 {
			pageCacheInfo.PageCached = pageCacheInfo.PageCached + 1
		}
	}

	if runtime.GOOS == "linux" {
		err = p.getActivePages(&pageCacheInfo, mmapPtr, fileSizePtr, vec, pageSize)
		if err != nil {
			return pageCacheInfo, err
		}
	}

	return pageCacheInfo, nil
}

func NewPcState() (pcState PcState, err error) {
	if runtime.GOOS != "linux" {
		// Nothing to do
		return
	}

	mode := os.FileMode(0600)
	pcState.pagemapFile, err = os.OpenFile("/proc/self/pagemap", os.O_RDONLY, mode)
	if err != nil {
		return
	}
	pcState.kpageFlags, err = os.OpenFile("/proc/kpageflags", os.O_RDONLY, mode)
	if err != nil {
		return
	}
	return
}

func (p *PcState) GetPcStats(fullPath string, pagesize int64) (PageCacheInfo, error) {
	pageCacheInfo := PageCacheInfo{0, 0, make(map[uint64]int, 0)}
	file, err := os.Open(fullPath)
	if err != nil {
		return pageCacheInfo, fmt.Errorf("Error opening file %s: %v", fullPath, err)
	}
	defer file.Close()
	fileInfo, err := file.Stat()
	if err != nil {
		return pageCacheInfo, fmt.Errorf("Error getting file stat %s: %v", fullPath, err)
	}
	fileSize := fileInfo.Size()
	if fileSize == 0 {
		return pageCacheInfo, nil
	}
	pageCacheInfo, err = p.getPagecacheStats(int(file.Fd()), fileInfo.Size(), pagesize)
	if err != nil {
		return pageCacheInfo, fmt.Errorf("Getting pagecache stats for %s failed: %v", fullPath, err)
	}
	return pageCacheInfo, nil
}
