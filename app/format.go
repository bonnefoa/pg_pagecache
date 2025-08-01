package app

import (
	"cmp"
	"slices"

	"github.com/bonnefoa/pg_pagecache/pagecache"
	"github.com/bonnefoa/pg_pagecache/relation"
)

func (p *PgPageCache) sortPartInfos(r []relation.PartInfo) {
	slices.SortFunc(r, func(a, b relation.PartInfo) int {
		switch p.Sort {
		case SortPageCount:
			return cmp.Or(cmp.Compare(b.PageCount, a.PageCount), cmp.Compare(a.Name, b.Name))
		case SortPageCached:
			return cmp.Or(cmp.Compare(b.PageCached, a.PageCached), cmp.Compare(a.Name, b.Name))
		}
		return cmp.Compare(a.Name, b.Name)
	})
}

func (p *PgPageCache) sortPageFlags(r []pagecache.PageFlags) {
	slices.SortFunc(r, func(a, b pagecache.PageFlags) int {
		return cmp.Compare(a.Count, b.Count)
	})
}

func (p *PgPageCache) sortTableInfos(r []relation.TableInfo) {
	slices.SortFunc(r, func(a, b relation.TableInfo) int {
		switch p.Sort {
		case SortPageCount:
			return cmp.Or(cmp.Compare(b.PageCount, a.PageCount), cmp.Compare(a.Name, b.Name))
		case SortPageCached:
			return cmp.Or(cmp.Compare(b.PageCached, a.PageCached), cmp.Compare(a.Name, b.Name))
		}
		return cmp.Compare(a.Name, b.Name)
	})
}

func (p *PgPageCache) sortRelInfos(r []relation.RelInfo) {
	slices.SortFunc(r, func(a, b relation.RelInfo) int {
		switch p.Sort {
		case SortPageCount:
			return cmp.Or(cmp.Compare(b.PageCount, a.PageCount), cmp.Compare(a.Name, b.Name))
		case SortPageCached:
			return cmp.Or(cmp.Compare(b.PageCached, a.PageCached), cmp.Compare(a.Name, b.Name))
		}
		return cmp.Compare(a.Name, b.Name)
	})
}

func (p *PgPageCache) getNoAggregations() (outputInfos []relation.OutputInfo) {
	var relinfos []relation.RelInfo
	for _, partInfo := range p.partitions {
		for _, tableInfo := range partInfo.TableInfos {
			relinfos = append(relinfos, tableInfo.RelInfos...)
		}
	}
	p.sortRelInfos(relinfos)

	for i, relinfo := range relinfos {
		if p.Limit > 0 && i >= p.Limit {
			break
		}
		outputInfos = append(outputInfos, &relinfo)
		relation.TotalInfo.Add(relinfo.PageStats)
	}
	return
}
