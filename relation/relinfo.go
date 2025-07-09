package relation

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bonnefoa/pg_pagecache/pcstats"
)

type AggregationType int

const (
	AggNone AggregationType = iota
	AggTable
	AggTableOnly
	AggPartition
	AggPartitionOnly
)

type OutputInfo interface {
	ToStringArray(unit FormatUnit, page_size int64, file_memory int64) []string
	ToFlagDetails() [][]string
}

type BaseInfo struct {
	Name string
	pcstats.PageCacheInfo
	Kind rune
}

type PartInfo struct {
	BaseInfo
	TableInfos map[string]TableInfo
}

type TableInfo struct {
	BaseInfo
	Partition string
	RelInfos  []RelInfo
}

type RelInfo struct {
	BaseInfo
	Partition   string
	Table       string
	Relfilenode uint32
}

var (
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
	UnitPage FormatUnit = iota
	UnitKB
	UnitMB
	UnitGB

	kebibyte = float64(1 << 10)
	mebibyte = float64(1 << 20)
	gebibyte = float64(1 << 30)
)

type FormatUnit int

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

func ParseAggregation(s string) (AggregationType, error) {
	agg, ok := formatAggregationMap[strings.ToLower(s)]
	if !ok {
		err := fmt.Errorf("Unknown aggregation: %v\n", s)
		return agg, err
	}
	return agg, nil
}

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

func formatValue(value int, unit FormatUnit, page_size int64) (valueStr string) {
	switch unit {
	case UnitPage:
		valueStr = strconv.FormatInt(int64(value), 10)
	case UnitKB:
		valueStr = strconv.FormatFloat(float64(int64(value)*page_size)/kebibyte, 'f', -1, 64)
	case UnitMB:
		valueStr = strconv.FormatFloat(float64(int64(value)*page_size)/mebibyte, 'f', 2, 64)
	case UnitGB:
		valueStr = strconv.FormatFloat(float64(int64(value)*page_size)/gebibyte, 'f', 2, 64)
	}
	return fmt.Sprintf("%s %s", valueStr, unitToString(unit))
}

func (r *BaseInfo) ToStringArray(unit FormatUnit, page_size int64, file_memory int64) []string {
	return []string{"", "", r.Name, kindToString(r.Kind), "",
		formatValue(r.PageCached, unit, page_size),
		formatValue(r.PageCount, unit, page_size),
		r.GetCachedPct(),
		r.GetTotalCachedPct(file_memory)}
}

func (r *RelInfo) ToStringArray(unit FormatUnit, page_size int64, file_memory int64) []string {
	res := []string{r.Partition, r.Table, r.Name, kindToString(r.Kind), fmt.Sprintf("%d", r.Relfilenode),
		formatValue(r.PageCached, unit, page_size),
		formatValue(r.PageCount, unit, page_size),
		r.GetCachedPct(),
		r.GetTotalCachedPct(file_memory)}
	return res
}

func (t *TableInfo) ToStringArray(unit FormatUnit, page_size int64, file_memory int64) []string {
	return []string{t.Partition, t.Name, "", kindToString(t.Kind), "",
		formatValue(t.PageCached, unit, page_size),
		formatValue(t.PageCount, unit, page_size),
		t.GetCachedPct(),
		t.GetTotalCachedPct(file_memory)}
}

func (p *PartInfo) ToStringArray(unit FormatUnit, page_size int64, file_memory int64) []string {
	return []string{p.Name, "", "", kindToString(p.Kind), "",
		formatValue(p.PageCached, unit, page_size),
		formatValue(p.PageCount, unit, page_size),
		p.GetCachedPct(),
		p.GetTotalCachedPct(file_memory)}
}

func (r *BaseInfo) ToFlagDetails() [][]string {
	return nil
}

func (r *RelInfo) ToFlagDetails() [][]string {
	if r.PageCacheInfo.PageFlags == nil {
		return nil
	}

	var res [][]string
	for k, count := range r.PageCacheInfo.PageFlags {
		// TODO: Add flag description list
		res = append(res, []string{fmt.Sprintf("0x%x", k), fmt.Sprintf("%d", count)})
	}

	return nil
}

func (r *TableInfo) ToFlagDetails() [][]string {
	return nil
}

func (r *PartInfo) ToFlagDetails() [][]string {
	return nil
}
