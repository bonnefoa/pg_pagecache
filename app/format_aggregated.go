package app

import (
	"fmt"

	"github.com/bonnefoa/pg_pagecache/relation"
)

type RelToRelinfo map[string]relation.RelInfo

// For a specific relinfo, fetch the list of children
func (p *PgPagecache) fetchChildren(table *relation.RelInfo, relToRelinfo RelToRelinfo) (relinfos []relation.RelInfo) {
	relinfos = make([]relation.RelInfo, 0)
	for _, childRelname := range table.Children {
		childRelinfo, present := relToRelinfo[childRelname]
		if present {
			relinfos = append(relinfos, childRelinfo)
		}
	}
	return
}

func (p *PgPagecache) fetchPartitions(table *relation.RelInfo, relToRelinfo RelToRelinfo) (relinfos []relation.RelInfo) {
	relinfos = make([]relation.RelInfo, 0)
	for _, partition := range table.Partitions {
		childRelinfo, present := relToRelinfo[partition]
		if present {
			relinfos = append(relinfos, childRelinfo)
		}
	}
	return
}

func (p *PgPagecache) sumChildrenPcstats(relToRelinfo RelToRelinfo, table relation.RelInfo) {
	if len(table.Children) == 0 {
		// No children
		return
	}

	for _, childRelname := range table.Children {
		if table.Relname == childRelname {
			// table page stats is already included in the base relation
			continue
		}

		childRelinfo, present := relToRelinfo[childRelname]
		if !present {
			continue
		}
		table.PcStats.Add(childRelinfo.PcStats)
	}
}

func (p *PgPagecache) sumPartitionsPcstats(relToRelinfo RelToRelinfo, parent relation.RelInfo) {
	if len(parent.Partitions) == 0 {
		// No children
		return
	}

	for _, partitionRelname := range parent.Partitions {
		partition, present := relToRelinfo[partitionRelname]
		if !present {
			continue
		}
		p.sumChildrenPcstats(relToRelinfo, partition)
		parent.PcStats.Add(partition.PcStats)
	}
}

func (p *PgPagecache) getAggregatedTables(relToRelinfo RelToRelinfo) (relinfos []relation.RelInfo) {
	relinfos = make([]relation.RelInfo, 0)
	for _, table := range p.fileToRelinfo {
		p.sumChildrenPcstats(relToRelinfo, table)
		relinfos = append(relinfos, table)
	}
	return relinfos
}

func (p *PgPagecache) getAggregatedPartitions(relToRelinfo RelToRelinfo) (relinfos []relation.RelInfo) {
	relinfos = make([]relation.RelInfo, 0)
	for _, table := range p.fileToRelinfo {
		p.sumPartitionsPcstats(relToRelinfo, table)
		relinfos = append(relinfos, table)
	}
	return relinfos
}

// outputTables prints relations with their children
func (p *PgPagecache) outputTables(relinfos []relation.RelInfo, relToRelinfo RelToRelinfo) error {
	strValues := make([][]string, 0)
	strValues = append(strValues, []string{"Table", "Relation", "Kind",
		fmt.Sprintf("PageCached (%s)", relation.UnitToString(p.Unit)),
		fmt.Sprintf("PageCount (%s)", relation.UnitToString(p.Unit)),
		"%Cached", "%Total"})
	for i, table := range relinfos {
		if p.Limit > 0 && i >= p.Limit {
			break
		}

		children := p.fetchChildren(&table, relToRelinfo)
		if len(children) > 1 {
			// Only show table when it has multiple children
			tableRelinfo := relation.RelInfo{Relname: "-", Relkind: '-', PcStats: table.PcStats}
			strValues = append(strValues, tableRelinfo.ToStringArrayTable(table.Relname, p.Unit, p.page_size, p.cached_memory))
		}

		if p.Aggregation == AggTableOnly {
			// Skip printing children
			continue
		}
		p.sortRelInfos(children)
		for _, child := range children {
			strValues = append(strValues, child.ToStringArrayTable(table.Relname, p.Unit, p.page_size, p.cached_memory))
		}
	}
	return p.outputValues(strValues)
}

func (p *PgPagecache) outputPartitions(relinfos []relation.RelInfo, relToRelinfo RelToRelinfo) error {
	strValues := make([][]string, 0)
	strValues = append(strValues, []string{"Partition", "Table", "Relation", "Kind",
		fmt.Sprintf("PageCached (%s)", relation.UnitToString(p.Unit)),
		fmt.Sprintf("PageCount (%s)", relation.UnitToString(p.Unit)),
		"%Cached", "%Total"})
	for i, table := range relinfos {
		if p.Limit > 0 && i >= p.Limit {
			break
		}

		if len(table.Partitions) == 0 && !table.IsPartition {
			// Not a partition nor in a partition, fallback to normal output
			tableRelinfo := relation.RelInfo{Relname: "-", Relkind: '-', PcStats: table.PcStats}
			strValues = append(strValues, tableRelinfo.ToStringArrayPartition("-", table.Relname, p.Unit, p.page_size, p.cached_memory))

		}

		partitions := p.fetchPartitions(&table, relToRelinfo)
		if len(children) > 1 {
			// Only show table when it has multiple children
			tableRelinfo := relation.RelInfo{Relname: "-", Relkind: '-', PcStats: table.PcStats}
			strValues = append(strValues, tableRelinfo.ToStringArrayTable(table.Relname, p.Unit, p.page_size, p.cached_memory))
		}

		if p.Aggregation == AggTableOnly {
			// Skip printing children
			continue
		}
		p.sortRelInfos(children)
		for _, child := range children {
			strValues = append(strValues, child.ToStringArrayTable(table.Relname, p.Unit, p.page_size, p.cached_memory))
		}
	}
	return p.outputValues(strValues)
}

func (p *PgPagecache) formatAggregated() error {
	// Build the relname -> relinfo map
	relToRelinfo := make(RelToRelinfo, 0)
	for _, v := range p.fileToRelinfo {
		relToRelinfo[v.Relname] = v
	}

	if p.Aggregation == AggPartition || p.Aggregation == AggPartitionOnly {
		// Get relinfos list
		tables := p.getAggregatedTables(relToRelinfo)
		// sort it
		p.sortRelInfos(tables)
		return p.outputTablesAggregated(tables, relToRelinfo)
	}

	// Get relinfos list
	tables := p.getAggregatedTables(relToRelinfo)
	// sort it
	p.sortRelInfos(tables)
	return p.outputTablesAggregated(tables, relToRelinfo)
}
