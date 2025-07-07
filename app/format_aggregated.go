package app

import (
	"fmt"
	"maps"
	"slices"

	"github.com/bonnefoa/pg_pagecache/relation"
)

type RelToRelinfo map[string]relation.RelInfo

func (p *PgPagecache) getHeader() []string {
	var res []string

	switch p.Aggregation {
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

	res = append(res, []string{"Relation", "Kind", fmt.Sprintf("PageCached (%s)",
		relation.UnitToString(p.Unit)), fmt.Sprintf("PageCount (%s)",
		relation.UnitToString(p.Unit)), "%Cached", "%Total"}...)

	return res
}

// formatAgggregated prints relations with their children
func (p *PgPagecache) formatAggregatedTables() (outputInfos []relation.OutputInfo, err error) {
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

		outputInfos = append(outputInfos, &tableInfo)
		total.PcStats.Add(tableInfo.PcStats)

		if p.Aggregation == AggTableOnly {
			// Skip printing children
			continue
		}
		p.sortRelInfos(relinfos)
		for _, child := range relinfos {
			outputInfos = append(outputInfos, child)
		}
	}

	outputInfos = append(outputInfos, &total)
	return
}

func (p *PgPagecache) formatAggregatePartitions() (outputInfos []relation.OutputInfo, err error) {
	partInfos := slices.Collect(maps.Keys(p.partitionToTables))
	p.sortPartInfos(partInfos)

	total := relation.RelInfo{Relkind: 'T'}

	i := 0
	if p.Aggregation == AggPartitionOnly {
		for _, partInfo := range partInfos {
			if p.Limit > 0 && i >= p.Limit {
				break
			}
			i++

			outputInfos = append(outputInfos, &partInfo)
			total.PcStats.Add(partInfo.PcStats)
		}
		outputInfos = append(outputInfos, &total)
		return
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
	return
}
