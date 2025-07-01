package app

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"slices"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/bonnefoa/pg_pagecache/relation"
)

func (p *PgPagecache) sortRelInfos(r []relation.RelInfo) {
	sort.Slice(r, func(i, j int) bool {
		switch p.Sort {
		case SortRelation:
			return r[i].Relname > r[j].Relname
		case SortPageCached:
			return r[i].PcStats.PageCached > r[j].PcStats.PageCached
		default:
			return r[i].PcStats.PageCount > r[j].PcStats.PageCount
		}
	})
}

func (p *PgPagecache) formatValue(value int) string {
	kb := float64(1024)
	mb := float64(1024 * 1024)
	gb := float64(1024 * 1024 * 1024)
	switch p.Unit {
	case UnitPage:
		return strconv.FormatInt(int64(value), 10)
	case UnitKB:
		return strconv.FormatFloat(float64(int64(value)*p.page_size)/kb, 'f', -1, 64)
	case UnitMB:
		return strconv.FormatFloat(float64(int64(value)*p.page_size)/mb, 'f', 2, 64)
	case UnitGB:
		return strconv.FormatFloat(float64(int64(value)*p.page_size)/gb, 'f', 2, 64)
	}
	panic("Unreachable code")
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
func (p *PgPagecache) outputRelinfos(relinfos []relation.RelInfo) error {
	strValues := make([][]string, 0)
	strValues = append(strValues, []string{"Relation", "Kind", "PageCached", "PageCount", "%Cached", "%Total"})
	for i, relinfo := range relinfos {
		if p.Limit > 0 && i >= p.Limit {
			break
		}
		strValues = append(strValues, []string{relinfo.Relname, relation.KindToString(relinfo.Relkind),
			p.formatValue(relinfo.PcStats.PageCached),
			p.formatValue(relinfo.PcStats.PageCount),
			relinfo.PcStats.GetCachedPct(),
			relinfo.PcStats.GetTotalCachedPct(p.cached_memory)})
	}
	return p.outputValues(strValues)
}

func (p *PgPagecache) formatNoAggregation() error {
	// No aggregation
	relinfos := slices.Collect(maps.Values(p.fileToRelinfo))
	p.sortRelInfos(relinfos)
	return p.outputRelinfos(relinfos)
}
