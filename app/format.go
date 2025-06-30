package app

import (
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"sort"

	"github.com/bonnefoa/pg_pagecache/relation"
)

// For a specific relinfo, fetch the list of children sorted
func (p *PgPagecache) fetchChildren(parent *relation.RelInfo) (relinfos []relation.RelInfo) {
	relinfos = make([]relation.RelInfo, len(parent.Children))
	for i, childRelname := range parent.Children {
		childRelinfo := p.relToRelinfo[childRelname]
		slog.Debug("Child", "relname", childRelinfo.Relname, "pcstat", childRelinfo.PcStats.PageCount)
		relinfos[i] = childRelinfo
	}

	return
}

func (p *PgPagecache) groupResults() (relinfos []relation.RelInfo) {
	switch p.OutputOptions.Aggregation {
	case AggRelation:
		return slices.Collect(maps.Values(p.fileToRelinfo))

	case AggParentWithChildren:
		// For group, we will create the same list of parent relinfos
		// During output, we will fetch the list of children
		fallthrough

	case AggOnlyParent:
		relinfos = make([]relation.RelInfo, 0)
		for _, parent := range p.fileToRelinfo {
			if len(parent.Children) > 0 {
				// It's a parent, accumulate PcStats
				slog.Debug("Processing parent", "Relation", parent.Relname)
				for _, childRelname := range parent.Children {
					if parent.Relname == childRelname {
						// Parent relname is already counted
						continue
					}
					childRelinfo := p.relToRelinfo[childRelname]
					slog.Debug("Processing children", "Relation", childRelname, "PageCount", childRelinfo.PcStats.PageCount)
					parent.PcStats.Add(childRelinfo.PcStats)
				}
				// And add it to the list
				relinfos = append(relinfos, parent)
			}
		}
		return relinfos
	}
	return relinfos
}

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

// outputRelinfos prints one line per relation
func (p *PgPagecache) outputRelinfos(relinfos []relation.RelInfo) {
	fmt.Print("Relation,Relkind,PageCached,PageCount,PercentCached\n")
	for i, relinfo := range relinfos {
		if p.OutputOptions.Limit > 0 && i >= p.OutputOptions.Limit {
			return
		}
		if relinfo.PcStats.PageCached < p.OutputOptions.Threshold {
			return
		}
		pctCached := 100 * float32(relinfo.PcStats.PageCached) / float32(relinfo.PcStats.PageCount)
		fmt.Printf("%s,%s,%d,%d,%.2f\n", relinfo.Relname, relation.KindToString(relinfo.Relkind), relinfo.PcStats.PageCached, relinfo.PcStats.PageCount, pctCached)
	}
}

// outputRelinfosGroupedChildren prints relations with their children
func (p *PgPagecache) outputRelinfosGroupedChildren(relinfos []relation.RelInfo) {
	// Parent With Children
	fmt.Print("Parent,Relation,PageCached,PageCount,PercentCached\n")
	for i, parent := range relinfos {
		if p.OutputOptions.Limit > 0 && i >= p.OutputOptions.Limit {
			return
		}
		if parent.PcStats.PageCached < p.OutputOptions.Threshold {
			continue
		}

		// Print the parent
		pctCached := 100 * float32(parent.PcStats.PageCached) / float32(parent.PcStats.PageCount)
		fmt.Printf("%s,,%d,%d,%.2f\n", parent.Relname, parent.PcStats.PageCached, parent.PcStats.PageCount, pctCached)

		children := p.fetchChildren(&parent)
		p.sortRelInfos(children)
		for _, child := range children {
			pctCached := 100 * float32(child.PcStats.PageCached) / float32(child.PcStats.PageCount)
			fmt.Printf("%s,%s,%s,%d,%d,%.2f\n", parent.Relname, child.Relname, relation.KindToString(child.Relkind), child.PcStats.PageCached, child.PcStats.PageCount, pctCached)
		}

	}
}

func (p *PgPagecache) OutputResults() {
	relinfos := p.groupResults()
	p.sortRelInfos(relinfos)
	slog.Debug("Sorted relinfos", "Length", len(relinfos))

	if p.OutputOptions.Aggregation != AggParentWithChildren {
		p.outputRelinfos(relinfos)
	} else {
		p.outputRelinfosGroupedChildren(relinfos)
	}

}
