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
	switch p.displayOptions.Aggregation {
	case DisplayRelation:
		return slices.Collect(maps.Values(p.fileToRelinfo))

	case DisplayParentWithChildren:
		// For display group, we will create the same list of parent relinfos
		// During display, we will fetch the list of children
		fallthrough

	case DisplayOnlyParent:
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
		switch p.displayOptions.Sort {
		case SortRelation:
			return r[i].Relname > r[j].Relname
		case SortPageCached:
			return r[i].PcStats.PageCached > r[j].PcStats.PageCached
		default:
			return r[i].PcStats.PageCount > r[j].PcStats.PageCount
		}
	})
}

func (p *PgPagecache) displayResults() {
	relinfos := p.groupResults()
	p.sortRelInfos(relinfos)
	slog.Debug("Sorted relinfos", "Length", len(relinfos))

	if p.displayOptions.Aggregation != DisplayParentWithChildren {
		fmt.Print("Relation,PageCount,PageCached\n")
		for _, relinfo := range relinfos {
			slog.Debug("Relinfo display", "relname", relinfo.Relname, "cached", relinfo.PcStats.PageCached)
			if relinfo.PcStats.PageCached > p.displayOptions.Threshold {
				fmt.Printf("%s,%d,%d\n", relinfo.Relname, relinfo.PcStats.PageCount, relinfo.PcStats.PageCached)
			}
		}
		return
	}

	// Parent With Children
	fmt.Print("Parent,Relation,PageCount,PageCached\n")
	for _, parent := range relinfos {
		if parent.PcStats.PageCached > p.displayOptions.Threshold {
			// This is the parent
			fmt.Printf("%s,,%d,%d\n", parent.Relname, parent.PcStats.PageCount, parent.PcStats.PageCached)

			children := p.fetchChildren(&parent)
			p.sortRelInfos(children)
			for _, child := range children {
				fmt.Printf("%s,%s,%d,%d\n", parent.Relname, child.Relname, child.PcStats.PageCount, child.PcStats.PageCached)
			}
		}
	}
}
