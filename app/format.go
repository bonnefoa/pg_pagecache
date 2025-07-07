package app

import (
	"sort"

	"github.com/bonnefoa/pg_pagecache/relation"
)

func (p *PgPagecache) sortPartInfos(r []relation.PartInfo) {
	sort.Slice(r, func(i, j int) bool {
		switch p.Sort {
		case SortName:
			return r[i].Name > r[j].Name
		case SortPageCached:
			return r[i].PcStats.PageCached > r[j].PcStats.PageCached
		default:
			return r[i].PcStats.PageCount > r[j].PcStats.PageCount
		}
	})
}

func (p *PgPagecache) sortTableInfos(r []relation.TableInfo) {
	sort.Slice(r, func(i, j int) bool {
		switch p.Sort {
		case SortName:
			return r[i].Name > r[j].Name
		case SortPageCached:
			return r[i].PcStats.PageCached > r[j].PcStats.PageCached
		default:
			return r[i].PcStats.PageCount > r[j].PcStats.PageCount
		}
	})
}

func (p *PgPagecache) sortRelInfos(r []relation.RelInfo) {
	sort.Slice(r, func(i, j int) bool {
		switch p.Sort {
		case SortName:
			return r[i].Name > r[j].Name
		case SortPageCached:
			return r[i].PcStats.PageCached > r[j].PcStats.PageCached
		default:
			return r[i].PcStats.PageCount > r[j].PcStats.PageCount
		}
	})
}

func (p *PgPagecache) formatNoAggregation() (outputInfos []relation.OutputInfo, err error) {
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
		total.PcStats.Add(relinfo.PcStats)
	}
	outputInfos = append(outputInfos, &total)
	return
}
