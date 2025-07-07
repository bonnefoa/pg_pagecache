package app

import (
	"maps"
	"slices"

	"github.com/bonnefoa/pg_pagecache/relation"
)

func (p *PgPagecache) sortTableToRelinfos(t relation.TableToRelinfos) []relation.TableInfo {
	tableInfos := slices.Collect(maps.Keys(t))
	p.sortTableInfos(tableInfos)
	return tableInfos
}

// formatAgggregated prints relations with their children
func (p *PgPagecache) formatAggregatedTables() (outputInfos []relation.OutputInfo, err error) {
	i := 0

	tableToRelinfos := make(map[relation.TableInfo][]*relation.RelInfo)
	// We don't care about partitions, flatten TableInfo -> []Relinfo map
	for _, tableMap := range p.partitionToTables {
		maps.Copy(tableToRelinfos, tableMap)
	}
	total := relation.TotalInfo

	sortedTableInfos := p.sortTableToRelinfos(tableToRelinfos)
	for _, tableInfo := range sortedTableInfos {
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

		// Add relinfo children
		relinfos := tableToRelinfos[tableInfo]
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

	total := relation.TotalInfo

	i := 0
	for _, partInfo := range partInfos {
		if p.Limit > 0 && i >= p.Limit {
			break
		}
		i++

		outputInfos = append(outputInfos, &partInfo)
		total.PcStats.Add(partInfo.PcStats)
		if p.Aggregation == AggPartitionOnly {
			continue
		}

		tableToRelinfos := p.partitionToTables[partInfo]
		sortedTableInfos := p.sortTableToRelinfos(tableToRelinfos)
		for _, tableInfo := range sortedTableInfos {
			outputInfos = append(outputInfos, &tableInfo)

			// Add relinfo children
			relinfos := tableToRelinfos[tableInfo]
			p.sortRelInfos(relinfos)
			for _, child := range relinfos {
				outputInfos = append(outputInfos, child)
			}
		}

	}
	outputInfos = append(outputInfos, &total)
	return
}
