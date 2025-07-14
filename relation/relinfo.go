package relation

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bonnefoa/pg_pagecache/pagecache"
)

// AggregationType represents the different types of aggregation
type AggregationType int

const (
	// AggNone outputs all relations (table, indexes, toast tables, toast indexes...) at the same level
	AggNone AggregationType = iota
	// AggTable groups relations with their indexes, toast and toast index
	AggTable
	// AggTableOnly aggregates results per parent relation with summed stats
	AggTableOnly
	// AggPartition aggregates partitions together
	AggPartition
	// AggPartitionOnly only display the parent partition with summed stats
	AggPartitionOnly
)

// OutputInfo represents an element that can will generate an output
type OutputInfo interface {
	ToStringArray(agg AggregationType, unit FormatUnit, pageSize int64, fileMemory int64) []string
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

	formatAggregationMap = map[string]AggregationType{
		"none":           AggNone,
		"table":          AggTable,
		"table_only":     AggTableOnly,
		"partition":      AggPartition,
		"partition_only": AggPartitionOnly,
	}
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

// ParseAggregation parses the aggregation flag
func ParseAggregation(s string) (AggregationType, error) {
	agg, ok := formatAggregationMap[strings.ToLower(s)]
	if !ok {
		err := fmt.Errorf("unknown aggregation: %v", s)
		return agg, err
	}
	return agg, nil
}

// GetHeader returns the output's header
func GetHeader(agg AggregationType) []string {
	var res []string

	switch agg {
	case AggPartition:
		fallthrough
	case AggPartitionOnly:
		res = append(res, "Partition")
		fallthrough
	case AggTable:
		fallthrough
	case AggTableOnly:
		res = append(res, "Table")
	case AggNone: // Nothing to do
	}

	res = append(res, []string{"Relation", "Kind", "Relfilenode",
		"PageCached", "PageCount",
		"%Cached", "%Total"}...)

	return res
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

func adjustLine(agg AggregationType, line []string) []string {
	switch agg {
	case AggNone:
		line = line[2:]
	case AggTable:
		fallthrough
	case AggTableOnly:
		line = line[1:]
	}
	return line
}

// ToStringArray outputs baseInfo's information
func (r *BaseInfo) ToStringArray(agg AggregationType, unit FormatUnit, pageSize int64, fileMemory int64) []string {
	res := []string{"", "", r.Name, kindToString(r.Kind), "",
		formatValue(r.PageCached, unit, pageSize),
		formatValue(r.PageCount, unit, pageSize),
		r.GetCachedPct(),
		r.GetTotalCachedPct(fileMemory)}
	return adjustLine(agg, res)
}

// ToStringArray outputs relInfo's information
func (r *RelInfo) ToStringArray(agg AggregationType, unit FormatUnit, pageSize int64, fileMemory int64) []string {
	res := []string{r.Partition, r.Table, r.Name, kindToString(r.Kind), fmt.Sprintf("%d", r.Relfilenode),
		formatValue(r.PageCached, unit, pageSize),
		formatValue(r.PageCount, unit, pageSize),
		r.GetCachedPct(),
		r.GetTotalCachedPct(fileMemory)}
	return adjustLine(agg, res)
}

// ToStringArray outputs tableInfo's information
func (t *TableInfo) ToStringArray(agg AggregationType, unit FormatUnit, pageSize int64, fileMemory int64) []string {
	res := []string{t.Partition, t.Name, "", kindToString(t.Kind), "",
		formatValue(t.PageCached, unit, pageSize),
		formatValue(t.PageCount, unit, pageSize),
		t.GetCachedPct(),
		t.GetTotalCachedPct(fileMemory)}
	return adjustLine(agg, res)
}

// ToStringArray outputs partInfo's information
func (p *PartInfo) ToStringArray(agg AggregationType, unit FormatUnit, pageSize int64, fileMemory int64) []string {
	res := []string{p.Name, "", "", kindToString(p.Kind), "",
		formatValue(p.PageCached, unit, pageSize),
		formatValue(p.PageCount, unit, pageSize),
		p.GetCachedPct(),
		p.GetTotalCachedPct(fileMemory)}
	return adjustLine(agg, res)
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
