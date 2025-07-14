package relation

import (
	"fmt"
	"strconv"

	"github.com/bonnefoa/pg_pagecache/pagecache"
)

// OutputInfo represents an element that can will generate an output
type OutputInfo interface {
	ToStringArray(unit FormatUnit, pageSize int64, fileMemory int64) []string
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

// FormatUnit represents the unit used for output
type FormatUnit int

var (
	// TotalInfo stores the sum of all page stats. Used to display the last sum line.
	TotalInfo = BaseInfo{Name: "Total", Kind: 'S'}
)

const (
	// UnitPage outputs values in pages
	UnitPage FormatUnit = iota
	// UnitKB outputs values in KB
	UnitKB
	// UnitMB outputs values in MB
	UnitMB
	// UnitGB outputs values in GB
	UnitGB

	kebibyte = float64(1 << 10)
	mebibyte = float64(1 << 20)
	gebibyte = float64(1 << 30)
)

func unitToString(u FormatUnit) string {
	switch u {
	case UnitPage:
		return "Pgs"
	case UnitKB:
		return "KB"
	case UnitMB:
		return "MB"
	case UnitGB:
		return "GB"
	}
	return "?"
}

func formatValue(value int, unit FormatUnit, pageSize int64) (valueStr string) {
	switch unit {
	case UnitPage:
		valueStr = strconv.FormatInt(int64(value), 10)
	case UnitKB:
		valueStr = strconv.FormatFloat(float64(int64(value)*pageSize)/kebibyte, 'f', -1, 64)
	case UnitMB:
		valueStr = strconv.FormatFloat(float64(int64(value)*pageSize)/mebibyte, 'f', 2, 64)
	case UnitGB:
		valueStr = strconv.FormatFloat(float64(int64(value)*pageSize)/gebibyte, 'f', 2, 64)
	}
	return fmt.Sprintf("%s %s", valueStr, unitToString(unit))
}

// GetPagestats returns the page stats
func (r *BaseInfo) GetPagestats() pagecache.PageStats {
	return r.PageStats
}

// ToStringArray outputs baseInfo's information
func (r *BaseInfo) ToStringArray(unit FormatUnit, pageSize int64, fileMemory int64) []string {
	return []string{"", "", r.Name, "", kindToString(r.Kind),
		formatValue(r.PageCached, unit, pageSize),
		formatValue(r.PageCount, unit, pageSize),
		r.GetCachedPct(),
		r.GetTotalCachedPct(fileMemory)}
}

// ToStringArray outputs relInfo's information
func (r *RelInfo) ToStringArray(unit FormatUnit, pageSize int64, fileMemory int64) []string {
	return []string{r.Partition, r.Table, r.Name, fmt.Sprintf("%d", r.Relfilenode),
		kindToString(r.Kind), formatValue(r.PageCached, unit, pageSize),
		formatValue(r.PageCount, unit, pageSize), r.GetCachedPct(),
		r.GetTotalCachedPct(fileMemory)}
}

// ToStringArray outputs tableInfo's information
func (t *TableInfo) ToStringArray(unit FormatUnit, pageSize int64, fileMemory int64) []string {
	return []string{t.Partition, t.Name, "", "", kindToString(t.Kind),
		formatValue(t.PageCached, unit, pageSize),
		formatValue(t.PageCount, unit, pageSize),
		t.GetCachedPct(),
		t.GetTotalCachedPct(fileMemory)}
}

// ToStringArray outputs partInfo's information
func (p *PartInfo) ToStringArray(unit FormatUnit, pageSize int64, fileMemory int64) []string {
	return []string{p.Name, "", "", "", kindToString(p.Kind),
		formatValue(p.PageCached, unit, pageSize),
		formatValue(p.PageCount, unit, pageSize),
		p.GetCachedPct(),
		p.GetTotalCachedPct(fileMemory)}
}

// ToFlagDetails outputs page cache flags details
func (r *BaseInfo) ToFlagDetails() [][]string {
	return nil
}

// ToFlagDetails outputs page cache flags details
func (r *RelInfo) ToFlagDetails() [][]string {
	if r.PageStats.PageFlags == nil {
		return nil
	}

	var res [][]string
	for flags, count := range r.PageStats.PageFlags {
		res = append(res, []string{
			r.Name, fmt.Sprintf("%d", count), fmt.Sprintf("0x%016x", flags),
			pagecache.PageFlagShortName(flags), pagecache.PageFlagLongName(flags)})
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
