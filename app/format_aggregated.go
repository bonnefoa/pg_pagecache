package app

import (
	"log/slog"

	"github.com/bonnefoa/pg_pagecache/relation"
)

type RelToRelinfo map[string]relation.RelInfo

// For a specific relinfo, fetch the list of children sorted
func (p *PgPagecache) fetchChildren(parent *relation.RelInfo, relToRelinfo RelToRelinfo) (relinfos []relation.RelInfo) {
	relinfos = make([]relation.RelInfo, 0)
	for _, childRelname := range parent.Children {
		childRelinfo, present := relToRelinfo[childRelname]
		if present {
			relinfos = append(relinfos, childRelinfo)
		}
	}
	return
}

func (p *PgPagecache) getAggregatedRelinfos(relToRelinfo RelToRelinfo) (relinfos []relation.RelInfo) {
	relinfos = make([]relation.RelInfo, 0)
	for _, parent := range p.fileToRelinfo {

		if len(parent.Children) == 0 {
			// It's a child, skip it
			continue
		}

		slog.Debug("Processing parent", "Relation", parent.Relname)
		for _, childRelname := range parent.Children {
			if parent.Relname == childRelname {
				// Parent page stats is already included in the base relation
				continue
			}

			childRelinfo, present := relToRelinfo[childRelname]
			if !present {
				continue
			}
			slog.Debug("Processing children", "Relation", childRelname, "PageCount", childRelinfo.PcStats.PageCount)
			parent.PcStats.Add(childRelinfo.PcStats)
		}
		// Add it to the list
		relinfos = append(relinfos, parent)
	}
	return relinfos
}

// outputRelinfosAggregated prints relations with their children
func (p *PgPagecache) outputRelinfosAggregated(relinfos []relation.RelInfo, relToRelinfo RelToRelinfo) error {
	strValues := make([][]string, 0)
	// Parent With Children
	strValues = append(strValues, []string{"Parent", "Relation", "Kind", "PageCached", "PageCount", "%Cached", "%Total"})
	for i, parent := range relinfos {
		if p.Limit > 0 && i >= p.Limit {
			break
		}

		children := p.fetchChildren(&parent, relToRelinfo)
		if len(children) > 1 {
			// Only show parent when it has multiple children
			strValues = append(strValues, []string{parent.Relname, "-", "-",
				p.formatValue(parent.PcStats.PageCached),
				p.formatValue(parent.PcStats.PageCount),
				parent.PcStats.GetCachedPct(),
				parent.PcStats.GetTotalCachedPct(p.cached_memory)})
		}

		p.sortRelInfos(children)
		for _, child := range children {
			strValues = append(strValues, []string{parent.Relname, child.Relname,
				relation.KindToString(child.Relkind),
				p.formatValue(child.PcStats.PageCached),
				p.formatValue(child.PcStats.PageCount),
				child.PcStats.GetCachedPct(), child.PcStats.GetTotalCachedPct(p.cached_memory)})
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
