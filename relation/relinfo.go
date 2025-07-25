package relation

import (
	"cmp"
	"fmt"
	"maps"
	"slices"

	"github.com/bonnefoa/pg_pagecache/pagecache"
	"github.com/bonnefoa/pg_pagecache/utils"
)

// OutputInfo represents an element that can will generate an output
type OutputInfo interface {
	ToStringArray(unit utils.Unit, pageSize int64, fileMemory int64) []string
	GetPagestats() pagecache.PageStats
	ToFlagDetails() [][]string
}

// BaseInfo contains informations shared by everyone (relation, partition, table...)
// with page cache stats, a name and a kind
type BaseInfo struct {
	pagecache.PageStats
	Name string
	Kind rune
}

// PartInfo represents the parent partition with its children
type PartInfo struct {
	BaseInfo
	TableInfos map[string]TableInfo
}

// TableInfo represents a relation with its indexes, toast table and toast table index
type TableInfo struct {
	BaseInfo
	Partition string
	RelInfos  []RelInfo
}

// RelInfo represents a relation, which can be anything from pg_class: a relation, an index, a toast table...
type RelInfo struct {
	BaseInfo
	Partition   string
	Table       string
	Relfilenode uint32
}

var (
	// TotalInfo stores the sum of all page stats. Used to display the last sum line.
	TotalInfo = BaseInfo{Name: "Total", Kind: 'S'}
)

// GetPagestats returns the page stats
func (r *BaseInfo) GetPagestats() pagecache.PageStats {
	return r.PageStats
}

// ToStringArray outputs baseInfo's information
func (r *BaseInfo) ToStringArray(unit utils.Unit, pageSize int64, fileMemory int64) []string {
	return []string{"", "", r.Name, "", kindToString(r.Kind),
		utils.FormatPageValue(r.PageCached, unit, pageSize),
		utils.FormatPageValue(r.PageCount, unit, pageSize),
		r.GetCachedPct(),
		r.GetTotalCachedPct(pageSize, fileMemory)}
}

// ToStringArray outputs relInfo's information
func (r *RelInfo) ToStringArray(unit utils.Unit, pageSize int64, fileMemory int64) []string {
	return []string{r.Partition, r.Table, r.Name, fmt.Sprintf("%d", r.Relfilenode),
		kindToString(r.Kind), utils.FormatPageValue(r.PageCached, unit, pageSize),
		utils.FormatPageValue(r.PageCount, unit, pageSize), r.GetCachedPct(),
		r.GetTotalCachedPct(pageSize, fileMemory)}
}

// ToStringArray outputs tableInfo's information
func (t *TableInfo) ToStringArray(unit utils.Unit, pageSize int64, fileMemory int64) []string {
	return []string{t.Partition, t.Name, "", "", kindToString(t.Kind),
		utils.FormatPageValue(t.PageCached, unit, pageSize),
		utils.FormatPageValue(t.PageCount, unit, pageSize),
		t.GetCachedPct(),
		t.GetTotalCachedPct(pageSize, fileMemory)}
}

// ToStringArray outputs partInfo's information
func (p *PartInfo) ToStringArray(unit utils.Unit, pageSize int64, fileMemory int64) []string {
	return []string{p.Name, "", "", "", kindToString(p.Kind),
		utils.FormatPageValue(p.PageCached, unit, pageSize),
		utils.FormatPageValue(p.PageCount, unit, pageSize),
		p.GetCachedPct(),
		p.GetTotalCachedPct(pageSize, fileMemory)}
}

// ToFlagDetails outputs page cache flags details
func (r *BaseInfo) ToFlagDetails() [][]string {
	return nil
}

// ToFlagDetails outputs page cache flags details
func (r *RelInfo) ToFlagDetails() [][]string {
	if r.PageStats.PageFlagsMap == nil {
		return nil
	}

	pageFlagsValues := slices.Collect(maps.Values(r.PageStats.PageFlagsMap))
	slices.SortFunc(pageFlagsValues, func(a, b pagecache.PageFlags) int {
		return cmp.Compare(b.Count, a.Count)
	})

	var res [][]string
	for _, pfs := range pageFlagsValues {
		res = append(res, []string{
			r.Name, fmt.Sprintf("%d", pfs.Count), fmt.Sprintf("0x%016x", pfs.Flags),
			pagecache.PageFlagShortName(pfs.Flags), pagecache.PageFlagLongName(pfs.Flags)})
	}

	return res
}

// ToFlagDetails outputs page cache flags details
func (t *TableInfo) ToFlagDetails() [][]string {
	return nil
}

// ToFlagDetails outputs page cache flags details
func (p *PartInfo) ToFlagDetails() [][]string {
	return nil
}
