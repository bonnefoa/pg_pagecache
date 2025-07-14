package pagecache

import (
	"strings"
)

// kpf is the enum of kernel page flags
type kpf int

// Kernel Page cache flags
const (
	kpfLocked     = 0
	kpfError      = 1
	kpfReferenced = 2
	kpfUptodate   = 3
	kpfDirty      = 4
	kpfLru        = 5
	kpfActive     = 6
	kpfSlab       = 7
	kpfWriteback  = 8
	kpfReclaim    = 9
	kpfBuddy      = 10

	kpfMmap         = 11
	kpfAnon         = 12
	kpfSwapcache    = 13
	kpfSwapbacked   = 14
	kpfCompoundHead = 15
	kpfCompoundTail = 16
	kpfHuge         = 17
	kpfUnevictable  = 18
	kpfHwpoison     = 19
	kpfNopage       = 20

	kpfKsm      = 21
	kpfThp      = 22
	kpfOffline  = 23
	kpfZeroPage = 24
	kpfIdle     = 25
	kpfPgtable  = 26

	kpfReserved     = 32
	kpfMlocked      = 33
	kpfOwner2       = 34
	kpfPrivate      = 35
	kpfPrivate2     = 36
	kpfOwnerPrivate = 37
	kpfArch         = 38
	kpfUncached     = 39
	kpfSoftdirty    = 40
	kpfArch2        = 41

	kpfAnonExclusive = 47
	kpfReadahead     = 48
	kpfSlubFrozen    = 50
	kpfSlubDebug     = 51
	kpfFile          = 61
	kpfSwap          = 62
	kpfMmapExclusive = 63

	pmSoftDirty     = 1 << 55
	pmMmapExclusive = 1 << 56
	pmFile          = 1 << 61
	pmSwap          = 1 << 62
	pmPresent       = 1 << 63
)

var (
	kpfMap = [64][]string{
		kpfLocked:     {"L", "locked"},
		kpfError:      {"E", "error"},
		kpfReferenced: {"R", "referenced"},
		kpfUptodate:   {"U", "uptodate"},
		kpfDirty:      {"D", "dirty"},
		kpfLru:        {"l", "lru"},
		kpfActive:     {"A", "active"},
		kpfSlab:       {"S", "slab"},
		kpfWriteback:  {"W", "writeback"},
		kpfReclaim:    {"I", "reclaim"},
		kpfBuddy:      {"B", "buddy"},

		kpfMmap:         {"M", "mmap"},
		kpfAnon:         {"a", "anonymous"},
		kpfSwapcache:    {"s", "swapcache"},
		kpfSwapbacked:   {"b", "swapbacked"},
		kpfCompoundHead: {"H", "compound_head"},
		kpfCompoundTail: {"T", "compound_tail"},
		kpfHuge:         {"G", "huge"},
		kpfUnevictable:  {"u", "unevictable"},
		kpfHwpoison:     {"X", "hwpoison"},
		kpfNopage:       {"n", "nopage"},
		kpfKsm:          {"x", "ksm"},
		kpfThp:          {"t", "thp"},
		kpfOffline:      {"o", "offline"},
		kpfPgtable:      {"g", "pgtable"},
		kpfZeroPage:     {"z", "zero_page"},
		kpfIdle:         {"i", "idle_page"},

		kpfReserved:     {"r", "reserved"},
		kpfMlocked:      {"m", "mlocked"},
		kpfOwner2:       {"d", "owner_2"},
		kpfPrivate:      {"P", "private"},
		kpfPrivate2:     {"p", "private_2"},
		kpfOwnerPrivate: {"O", "owner_private"},
		kpfArch:         {"h", "arch"},
		kpfSoftdirty:    {"f", "softdirty"},
		kpfArch2:        {"H", "arch_2"},

		kpfAnonExclusive: {"d", "anon_exclusive"},
		kpfReadahead:     {"I", "readahead"},
		kpfSlubFrozen:    {"A", "slub_frozen"},
		kpfSlubDebug:     {"E", "slub_debug"},

		kpfFile:          {"F", "file"},
		kpfSwap:          {"w", "swap"},
		kpfMmapExclusive: {"1", "mmap_exclusive"},
	}
)

func expandOverloadedFlags(flags uint64, pme uint64) uint64 {
	/* Anonymous pages use PG_owner_2 for anon_exclusive */
	if (flags&(1<<kpfAnon)) > 0 && (flags&(1<<kpfOwner2)) > 0 {
		flags ^= ((1 << kpfOwner2) | (1 << kpfAnonExclusive))
	}

	/* SLUB overloads several page flags */
	if (flags & (1 << kpfSlab)) > 0 {
		if (flags & (1 << kpfActive)) > 0 {
			flags ^= (1 << kpfActive) | (1 << kpfSlubFrozen)
		}
		if (flags & (1 << kpfError)) > 0 {
			flags ^= (1 << kpfError) | (1 << kpfSlubFrozen)
		}
	}

	/* PG_reclaim is overloaded as PG_readahead in the read path */
	if (flags & ((1 << kpfReclaim) | (1 << kpfWriteback))) == (1 << kpfReclaim) {
		flags ^= (1 << kpfReclaim) | (1 << kpfReadahead)
	}

	if (pme & pmSoftDirty) > 0 {
		flags |= (1 << kpfSoftdirty)
	}
	if (pme & pmFile) > 0 {
		flags |= (1 << kpfFile)
	}
	if (pme & pmSwap) > 0 {
		flags |= (1 << kpfSwap)
	}
	if (pme & pmMmapExclusive) > 0 {
		flags |= (1 << kpfMmapExclusive)
	}

	return flags
}

// PageFlagShortName returns flag values as one character per flag, with '_' if not set
func PageFlagShortName(flags uint64) string {
	res := strings.Builder{}

	for i, flagName := range kpfMap {
		present := (flags >> i) & 1
		if present > 0 {
			res.WriteString(flagName[0])
		} else {
			res.WriteString("_")
		}
	}
	return res.String()
}

// PageFlagLongName returns flag values as long names, separated by ','
func PageFlagLongName(flags uint64) string {
	res := strings.Builder{}

	for i, flagName := range kpfMap {
		present := (flags >> i) & 1
		if present > 0 {
			res.WriteString(flagName[1])
			res.WriteRune(',')
		}
	}
	return strings.Trim(res.String(), ",")
}
