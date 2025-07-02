package app

import (
	"fmt"
	"log/slog"

	"github.com/bonnefoa/pg_pagecache/relation"
)

type RelToRelinfo map[string]relation.RelInfo

// For a specific relinfo, fetch the list of children sorted
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

func (p *PgPagecache) getAggregatedRelinfos(relToRelinfo RelToRelinfo) (relinfos []relation.RelInfo) {
	relinfos = make([]relation.RelInfo, 0)
	for _, table := range p.fileToRelinfo {

		if len(table.Children) == 0 {
			// It's a child, skip it
			continue
		}

		slog.Debug("Processing table", "Relation", table.Relname)
		for _, childRelname := range table.Children {
			if table.Relname == childRelname {
				// table page stats is already included in the base relation
				continue
			}

			childRelinfo, present := relToRelinfo[childRelname]
			if !present {
				continue
			}
			slog.Debug("Processing children", "Relation", childRelname, "PageCount", childRelinfo.PcStats.PageCount)
			table.PcStats.Add(childRelinfo.PcStats)
		}
		// Add it to the list
		relinfos = append(relinfos, table)
	}
	return relinfos
}

// outputRelinfosAggregated prints relations with their children
func (p *PgPagecache) outputRelinfosAggregated(relinfos []relation.RelInfo, relToRelinfo RelToRelinfo) error {
	strValues := make([][]string, 0)
	// Table With Children
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
			parentRelinfo := relation.RelInfo{Relname: "-", Relkind: '-', PcStats: table.PcStats}
			strValues = append(strValues, parentRelinfo.ToStringArrayParent(table.Relname, p.Unit, p.page_size, p.cached_memory))
		}

		if p.Aggregation == AggTableOnly {
			// Skip printing children
			continue
		}
		p.sortRelInfos(children)
		for _, child := range children {
			strValues = append(strValues, child.ToStringArrayParent(table.Relname, p.Unit, p.page_size, p.cached_memory))
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

	// Get relinfos list
	relinfos := p.getAggregatedRelinfos(relToRelinfo)

	// sort it
	p.sortRelInfos(relinfos)

	return p.outputRelinfosAggregated(relinfos, relToRelinfo)
}
