package app

import (
	"maps"
	"slices"

	"github.com/bonnefoa/pg_pagecache/relation"
)

func (p *PgPageCache) getAggregatedPartitions() (outputInfos []relation.OutputInfo) {
	i := 0
	total := relation.TotalInfo

	partitionSlice := slices.Collect(maps.Values(p.partitions))
	p.sortPartInfos(partitionSlice)
	for _, partition := range partitionSlice {
		if p.Limit > 0 && i >= p.Limit {
			break
		}
		i++

		// Add parent partition
		outputInfos = append(outputInfos, &partition)
		total.Add(partition.PageStats)

		tableInfos := slices.Collect(maps.Values(partition.TableInfos))
		p.sortTableInfos(tableInfos)
		for _, tableInfo := range tableInfos {
			// Add table
			outputInfos = append(outputInfos, &tableInfo)

			if !p.GroupTable {
				// Add relinfo children
				p.sortRelInfos(tableInfo.RelInfos)
				for _, relInfo := range tableInfo.RelInfos {
					outputInfos = append(outputInfos, &relInfo)
				}
			}
		}
	}
	outputInfos = append(outputInfos, &relation.WalInfo)
	outputInfos = append(outputInfos, &total)
	return
}

func (p *PgPageCache) getAggregatedTables() (outputInfos []relation.OutputInfo) {
	i := 0
	total := relation.TotalInfo

	var tableInfos []relation.TableInfo
	// We don't care about partitions, flatten TableInfo -> []Relinfo map
	for _, partInfo := range p.partitions {
		tableInfos = append(tableInfos, slices.Collect(maps.Values(partInfo.TableInfos))...)
	}
	p.sortTableInfos(tableInfos)

	for _, tableInfo := range tableInfos {
		if p.Limit > 0 && i >= p.Limit {
			break
		}
		i++

		outputInfos = append(outputInfos, &tableInfo)
		total.Add(tableInfo.PageStats)

		if p.GroupTable {
			// Skip printing children
			continue
		}

		// Add relinfo children
		p.sortRelInfos(tableInfo.RelInfos)
		for _, relinfo := range tableInfo.RelInfos {
			outputInfos = append(outputInfos, &relinfo)
		}
	}

	outputInfos = append(outputInfos, &relation.WalInfo)
	outputInfos = append(outputInfos, &total)
	return
}
