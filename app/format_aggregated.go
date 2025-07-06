package app

import (
	"fmt"
	"maps"
	"slices"

	"github.com/bonnefoa/pg_pagecache/relation"
)

type RelToRelinfo map[string]relation.RelInfo

// formatAgggregated prints relations with their children
func (p *PgPagecache) formatAggregated() error {
	strValues := make([][]string, 0)
	// Table With Children
	strValues = append(strValues, []string{"Table", "Relation", "Kind",
		fmt.Sprintf("PageCached (%s)", relation.UnitToString(p.Unit)),
		fmt.Sprintf("PageCount (%s)", relation.UnitToString(p.Unit)),
		"%Cached", "%Total"})

	i := 0

	tableInfos := slices.Collect(maps.Keys(p.tableToRelinfos))
	p.sortTableInfos(tableInfos)
	total := relation.RelInfo{Relname: "", Relkind: 'T'}

	for _, tableInfo := range tableInfos {
		relinfos := p.tableToRelinfos[tableInfo]

		if p.Limit > 0 && i >= p.Limit {
			break
		}

		strValues = append(strValues, tableInfo.ToStringArray(p.Unit, p.page_size, p.cached_memory))
		total.PcStats.Add(tableInfo.PcStats)

		if p.Aggregation == AggTableOnly {
			// Skip printing children
			continue
		}
		p.sortRelInfos(relinfos)
		for _, child := range relinfos {
			strValues = append(strValues, child.ToStringArrayParent("", p.Unit, p.page_size, p.cached_memory))
		}
		i++
	}

	strValues = append(strValues, total.ToStringArrayParent("Total", p.Unit, p.page_size, p.cached_memory))
	return p.outputValues(strValues)
}
