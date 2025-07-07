package app

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

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

func (p *PgPagecache) sortRelInfos(r []*relation.RelInfo) {
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

func (p *PgPagecache) outputValues(outputInfos []relation.OutputInfo) error {
	var values [][]string

	header := p.getHeader()
	if !p.NoHeader {
		values = append(values, header)
	}

	for _, v := range outputInfos {
		line := v.ToStringArray(p.Unit, p.page_size, p.cached_memory)
		switch p.Aggregation {
		case AggNone:
			line = line[2:]
		case AggTable:
			fallthrough
		case AggTableOnly:
			line = line[1:]
		}
		values = append(values, line)
	}

	switch p.Type {
	case FormatCSV:
		w := csv.NewWriter(os.Stdout)
		w.WriteAll(values)
		return w.Error()
	case FormatJson:
		m := make([]map[string]string, 0)
		for _, line := range values {
			o := make(map[string]string, 0)
			for i, k := range header {
				o[k] = line[i]
			}
			m = append(m, o)
		}
		res, err := json.Marshal(m)
		if err != nil {
			return err
		}
		fmt.Print(string(res))
	case FormatColumn:
		w := tabwriter.NewWriter(os.Stdout, 14, 0, 1, ' ', 0)
		for _, v := range values {
			fmt.Fprintln(w, strings.Join(v, "\t"))
		}
		w.Flush()
	}
	return nil
}

func (p *PgPagecache) formatNoAggregation() (outputInfos []relation.OutputInfo, err error) {
	var relinfos []*relation.RelInfo
	for _, tables := range p.partitionToTables {
		for _, r := range tables {
			relinfos = append(relinfos, r...)
		}
	}
	p.sortRelInfos(relinfos)

	total := relation.RelInfo{BaseInfo: relation.BaseInfo{Name: "Total"}, Relkind: 'T'}
	for i, relinfo := range relinfos {
		if p.Limit > 0 && i >= p.Limit {
			break
		}
		outputInfos = append(outputInfos, relinfo)
		total.PcStats.Add(relinfo.PcStats)
	}
	outputInfos = append(outputInfos, &total)
	return
}
