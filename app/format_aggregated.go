package app

import (
	"maps"
	"slices"

	"github.com/bonnefoa/pg_pagecache/relation"
)

// formatAgggregated prints relations with their children
func (p *PgPagecache) formatAggregatedTables() (outputInfos []relation.OutputInfo, err error) {
	i := 0

	var tableInfos []relation.TableInfo
	// We don't care about partitions, flatten TableInfo -> []Relinfo map
	for _, partInfo := range p.partitions {
		tableInfos = append(tableInfos, slices.Collect(maps.Values(partInfo.TableInfos))...)
	}
	total := relation.TotalInfo

	p.sortTableInfos(tableInfos)
	for _, tableInfo := range tableInfos {
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
		p.sortRelInfos(tableInfo.RelInfos)
		for _, relinfo := range tableInfo.RelInfos {
			outputInfos = append(outputInfos, &relinfo)
		}
	}

	outputInfos = append(outputInfos, &total)
	return
}

func (p *PgPagecache) formatAggregatePartitions() (outputInfos []relation.OutputInfo, err error) {
	p.sortPartInfos(p.partitions)

	total := relation.TotalInfo

	i := 0
	for _, partInfo := range p.partitions {
		if p.Limit > 0 && i >= p.Limit {
			break
		}
		i++

		outputInfos = append(outputInfos, &partInfo)
		total.PcStats.Add(partInfo.PcStats)
		if p.Aggregation == AggPartitionOnly {
			continue
		}

		tableInfos := slices.Collect(maps.Values(partInfo.TableInfos))
		p.sortTableInfos(tableInfos)
		for _, tableInfo := range tableInfos {
			outputInfos = append(outputInfos, &tableInfo)

			// Add relinfo children
			p.sortRelInfos(tableInfo.RelInfos)
			for _, relInfo := range tableInfo.RelInfos {
				outputInfos = append(outputInfos, &relInfo)
			}
		}

	}
	outputInfos = append(outputInfos, &total)
	return
}
