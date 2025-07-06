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

func (p *PgPagecache) outputValues(valuesWithHeader [][]string) error {
	values := valuesWithHeader
	if p.NoHeader {
		values = valuesWithHeader[1:]
	}
	switch p.Type {
	case FormatCSV:
		w := csv.NewWriter(os.Stdout)
		w.WriteAll(values)
		return w.Error()
	case FormatJson:
		m := make([]map[string]string, 0)
		for _, line := range valuesWithHeader[1:] {
			o := make(map[string]string, 0)
			for i, k := range valuesWithHeader[0] {
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

// outputRelinfos prints one line per relation
func (p *PgPagecache) outputRelinfos(relinfos []*relation.RelInfo) error {
	total := relation.RelInfo{BaseInfo: relation.BaseInfo{Name: "Total"}, Relkind: 'T'}
	strValues := make([][]string, 0)
	strValues = append(strValues, []string{"Relation", "Kind",
		fmt.Sprintf("PageCached (%s)", relation.UnitToString(p.Unit)),
		fmt.Sprintf("PageCount (%s)", relation.UnitToString(p.Unit)),
		"%Cached", "%Total"})
	for i, relinfo := range relinfos {
		if p.Limit > 0 && i >= p.Limit {
			break
		}
		strValues = append(strValues, relinfo.ToStringArray(p.Unit, p.page_size, p.cached_memory))
		total.PcStats.Add(relinfo.PcStats)
	}
	strValues = append(strValues, total.ToStringArray(p.Unit, p.page_size, p.cached_memory))
	return p.outputValues(strValues)
}

func (p *PgPagecache) formatNoAggregation() error {
	// No aggregation
	var relinfos []*relation.RelInfo
	for _, tables := range p.partitionToTables {
		for _, r := range tables {
			relinfos = append(relinfos, r...)
		}
	}
	p.sortRelInfos(relinfos)
	return p.outputRelinfos(relinfos)
}
