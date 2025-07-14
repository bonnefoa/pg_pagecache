package pagecache

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

func (s *State) populatePTE(mmapPtr uintptr, fileSizePtr uintptr, cachedPages []byte, pageSize int64) (err error) {
	ret, _, err := syscall.Syscall(syscall.SYS_MADVISE, mmapPtr, fileSizePtr, unix.MADV_RANDOM)
	if ret != 0 {
		return fmt.Errorf("syscall MADVISE failed: %v", err)
	}

	// Populate PTE
	for i, v := range cachedPages {
		if v&0x1 > 0 {
			_ = *(*byte)(unsafe.Pointer(mmapPtr + uintptr(int64(i)*pageSize)))
		}
	}

	ret, _, err = syscall.Syscall(syscall.SYS_MADVISE, mmapPtr, fileSizePtr, unix.MADV_SEQUENTIAL)
	if ret != 0 {
		return fmt.Errorf("syscall MADVISE failed: %v", err)
	}
	return nil
}

func (s *State) readPageMap(mmapPtr uintptr, numPages int, pageSize int64) (pagemapFlags []uint64, err error) {
	indexPages := (int64(mmapPtr) / pageSize)
	pagemapFlags, err = readInt64SliceFromFile(s.pagemapFile, numPages, indexPages)
	if err != nil {
		err = fmt.Errorf("error reading pagemap flags: %v", err)
		return
	}
	return
}

func (s *State) readKpageFlags(pagemapFlags []uint64) (pageFlags map[uint64]int, err error) {
	pageFlags = make(map[uint64]int, 0)
	for _, pme := range pagemapFlags {
		pfn := pme & 0x7FFFFFFFFFFFFF
		if pfn == 0 {
			continue
		}

		var flagSlice []uint64
		flagSlice, err = readInt64SliceFromFile(s.kpageFlagsFile, 1, int64(pfn))
		if err != nil {
			return
		}
		var flags uint64
		if s.rawFlags {
			flags = expandOverloadedFlags(flagSlice[0], pme)
		} else {
			flags = wellKnownFlags(flagSlice[0])
		}
		pageFlags[flags]++
	}

	return
}
