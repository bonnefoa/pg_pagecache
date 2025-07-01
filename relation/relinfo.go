package relation

import (
	"strconv"

	"github.com/bonnefoa/pg_pagecache/pcstats"
)

type RelInfo struct {
	Relfilenode uint32
	Relname     string
	Relkind     rune
	PcStats     pcstats.PcStats
	Children    []string
}

const (
	UnitPage FormatUnit = iota
	UnitKB
	UnitMB
	UnitGB
)

type FileToRelinfo map[uint32]RelInfo
type FormatUnit int

func formatValue(value int, unit FormatUnit, page_size int64) string {
	kb := float64(1024)
	mb := float64(1024 * 1024)
	gb := float64(1024 * 1024 * 1024)
	switch unit {
	case UnitPage:
		return strconv.FormatInt(int64(value), 10)
	case UnitKB:
		return strconv.FormatFloat(float64(int64(value)*page_size)/kb, 'f', -1, 64)
	case UnitMB:
		return strconv.FormatFloat(float64(int64(value)*page_size)/mb, 'f', 2, 64)
	case UnitGB:
		return strconv.FormatFloat(float64(int64(value)*page_size)/gb, 'f', 2, 64)
	}
	panic("Unreachable code")
}

func (r *RelInfo) ToStringArray(unit FormatUnit, page_size int64, cached_memory int64) []string {
	return []string{r.Relname, KindToString(r.Relkind),
		formatValue(r.PcStats.PageCached, unit, page_size),
		formatValue(r.PcStats.PageCount, unit, page_size),
		r.PcStats.GetCachedPct(),
		r.PcStats.GetTotalCachedPct(cached_memory)}
}

func (r *RelInfo) ToStringArrayParent(parent string, unit FormatUnit, page_size int64, cached_memory int64) []string {
	return []string{parent, r.Relname, KindToString(r.Relkind),
		formatValue(r.PcStats.PageCached, unit, page_size),
		formatValue(r.PcStats.PageCount, unit, page_size),
		r.PcStats.GetCachedPct(),
		r.PcStats.GetTotalCachedPct(cached_memory)}
}
