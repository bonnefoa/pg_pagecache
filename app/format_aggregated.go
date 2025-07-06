package app

import (
	"fmt"
	"maps"
	"slices"

	"github.com/bonnefoa/pg_pagecache/relation"
)

type RelToRelinfo map[string]relation.RelInfo

// formatAgggregated prints relations with their children
func (p *PgPagecache) formatAggregatedTables() error {
	strValues := make([][]string, 0)
	// Header with table
	strValues = append(strValues, []string{"Table", "Relation", "Kind",
		fmt.Sprintf("PageCached (%s)", relation.UnitToString(p.Unit)),
		fmt.Sprintf("PageCount (%s)", relation.UnitToString(p.Unit)),
		"%Cached", "%Total"})

	i := 0

	tableToRelinfos := make(map[relation.TableInfo][]*relation.RelInfo)

	// We don't care about partitions, build a TableInfo -> []Relinfo map
	for _, tableMap := range p.partitionToTables {
		maps.Copy(tableToRelinfos, tableMap)
	}

	// Flatten the tableInfos to sort them
	tableInfos := slices.Collect(maps.Keys(tableToRelinfos))
	p.sortTableInfos(tableInfos)
	total := relation.RelInfo{Relkind: 'T'}

	for _, tableInfo := range tableInfos {
		relinfos := tableToRelinfos[tableInfo]

		if p.Limit > 0 && i >= p.Limit {
			break
		}
		i++

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
	}

	strValues = append(strValues, total.ToStringArrayParent("Total", p.Unit, p.page_size, p.cached_memory))
	return p.outputValues(strValues)
}

func (p *PgPagecache) formatAggregatePartitions() error {
	strValues := make([][]string, 0)
	// Header with partitons
	strValues = append(strValues, []string{"Partition", "Table", "Relation", "Kind",
		fmt.Sprintf("PageCached (%s)", relation.UnitToString(p.Unit)),
		fmt.Sprintf("PageCount (%s)", relation.UnitToString(p.Unit)),
		"%Cached", "%Total"})

	i := 0

	partInfos := slices.Collect(maps.Keys(p.partitionToTables))
	p.sortPartInfos(partInfos)

	total := relation.RelInfo{Relkind: 'T'}

	if p.Aggregation == AggPartitionOnly {
		for _, partInfo := range partInfos {
			if p.Limit > 0 && i >= p.Limit {
				break
			}
			i++

			strValues = append(strValues, partInfo.ToStringArray(p.Unit, p.page_size, p.cached_memory))
			total.PcStats.Add(partInfo.PcStats)
		}
		strValues = append(strValues, total.ToStringArrayParent("Total", p.Unit, p.page_size, p.cached_memory))
	}

	// // Flatten the tableInfos to sort them
	// tableInfos := slices.Collect(maps.Keys(tableToRelinfos))
	// p.sortTableInfos(tableInfos)
	// total := relation.RelInfo{Relname: "", Relkind: 'T'}
	//
	//	for _, tableInfo := range tableInfos {
	//		relinfos := tableToRelinfos[tableInfo]
	//
	//		if p.Limit > 0 && i >= p.Limit {
	//			break
	//		}
	//
	//		strValues = append(strValues, tableInfo.ToStringArray(p.Unit, p.page_size, p.cached_memory))
	//		total.PcStats.Add(tableInfo.PcStats)
	//
	//		if p.Aggregation == AggTableOnly {
	//			// Skip printing children
	//			continue
	//		}
	//		p.sortRelInfos(relinfos)
	//		for _, child := range relinfos {
	//			strValues = append(strValues, child.ToStringArrayParent("", p.Unit, p.page_size, p.cached_memory))
	//		}
	//		i++
	//	}
	//
	// strValues = append(strValues, total.ToStringArrayParent("Total", p.Unit, p.page_size, p.cached_memory))
	return p.outputValues(strValues)
}
