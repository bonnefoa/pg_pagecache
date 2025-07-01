package app

import (
	"fmt"
	"maps"
	"slices"
	"sort"
	"strconv"

	"github.com/bonnefoa/pg_pagecache/relation"
)

func (p *PgPagecache) sortRelInfos(r []relation.RelInfo) {
	sort.Slice(r, func(i, j int) bool {
		switch p.OutputOptions.Sort {
		case SortRelation:
			return r[i].Relname > r[j].Relname
		case SortPageCached:
			return r[i].PcStats.PageCached > r[j].PcStats.PageCached
		default:
			return r[i].PcStats.PageCount > r[j].PcStats.PageCount
		}
	})
}

func (p *PgPagecache) formatValue(value int) string {
	kb := float64(1024)
	mb := float64(1024 * 1024)
	gb := float64(1024 * 1024 * 1024)
	switch p.OutputOptions.Unit {
	case UnitPage:
		return strconv.FormatInt(int64(value), 10)
	case UnitKB:
		return strconv.FormatFloat(float64(int64(value)*p.page_size)/kb, 'f', -1, 64)
	case UnitMB:
		return strconv.FormatFloat(float64(int64(value)*p.page_size)/mb, 'f', 2, 64)
	case UnitGB:
		return strconv.FormatFloat(float64(int64(value)*p.page_size)/gb, 'f', 2, 64)
	}
	panic("Unreachable code")
}

// outputRelinfos prints one line per relation
func (p *PgPagecache) outputRelinfos(relinfos []relation.RelInfo) {
	fmt.Print("Relation,Kind,PageCached,PageCount,PercentCached,PercentTotal\n")
	for i, relinfo := range relinfos {
		if p.OutputOptions.Limit > 0 && i >= p.OutputOptions.Limit {
			return
		}
		fmt.Printf("%s,%s,%s,%s,%s,%s\n",
			relinfo.Relname,
			relation.KindToString(relinfo.Relkind),
			p.formatValue(relinfo.PcStats.PageCached),
			p.formatValue(relinfo.PcStats.PageCount),
			relinfo.PcStats.GetCachedPct(),
			relinfo.PcStats.GetTotalCachedPct(p.cached_memory))
	}
}

func (p *PgPagecache) formatNoAggregation() {
	// No aggregation
	relinfos := slices.Collect(maps.Values(p.fileToRelinfo))
	p.sortRelInfos(relinfos)
	p.outputRelinfos(relinfos)
}
