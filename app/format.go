package app

import (
	"cmp"
	"slices"

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

func (p *PgPageCache) formatNoAggregation() (outputInfos []relation.OutputInfo, err error) {
	var relinfos []relation.RelInfo
	for _, partInfo := range p.partitions {
		for _, tableInfo := range partInfo.TableInfos {
			relinfos = append(relinfos, tableInfo.RelInfos...)
		}
	}
	p.sortRelInfos(relinfos)

	total := relation.TotalInfo
	for i, relinfo := range relinfos {
		if p.Limit > 0 && i >= p.Limit {
			break
		}
		outputInfos = append(outputInfos, &relinfo)
		total.Add(relinfo.PageStats)
	}
	outputInfos = append(outputInfos, &total)
	return
}
