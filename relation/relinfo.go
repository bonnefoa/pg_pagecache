package relation

import (
	"fmt"
	"strconv"

	"github.com/bonnefoa/pg_pagecache/pcstats"
)

type OutputInfo interface {
	ToStringArray(unit FormatUnit, page_size int64, file_memory int64) []string
}

type BaseInfo struct {
	Name    string
	PcStats pcstats.PcStats
	Kind    rune
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

var TotalInfo = BaseInfo{Name: "Total", Kind: 'S'}

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
		formatValue(r.PcStats.PageCached, unit, page_size),
		formatValue(r.PcStats.PageCount, unit, page_size),
		r.PcStats.GetCachedPct(),
		r.PcStats.GetTotalCachedPct(file_memory)}
}

func (r *RelInfo) ToStringArray(unit FormatUnit, page_size int64, file_memory int64) []string {
	return []string{r.Partition, r.Table, r.Name, kindToString(r.Kind), fmt.Sprintf("%d", r.Relfilenode),
		formatValue(r.PcStats.PageCached, unit, page_size),
		formatValue(r.PcStats.PageCount, unit, page_size),
		r.PcStats.GetCachedPct(),
		r.PcStats.GetTotalCachedPct(file_memory)}
}

func (t *TableInfo) ToStringArray(unit FormatUnit, page_size int64, file_memory int64) []string {
	return []string{t.Partition, t.Name, "", kindToString(t.Kind), "",
		formatValue(t.PcStats.PageCached, unit, page_size),
		formatValue(t.PcStats.PageCount, unit, page_size),
		t.PcStats.GetCachedPct(),
		t.PcStats.GetTotalCachedPct(file_memory)}
}

func (p *PartInfo) ToStringArray(unit FormatUnit, page_size int64, file_memory int64) []string {
	return []string{p.Name, "", "", kindToString(p.Kind), "",
		formatValue(p.PcStats.PageCached, unit, page_size),
		formatValue(p.PcStats.PageCount, unit, page_size),
		p.PcStats.GetCachedPct(),
		p.PcStats.GetTotalCachedPct(file_memory)}
}
