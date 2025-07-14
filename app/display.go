package app

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/bonnefoa/pg_pagecache/relation"
)

var (
	pageHeader = []string{
		"Partition", "Table", "Relation", "Relfilenode", "Kind", "PageCached",
		"PageCount", "%Cached", "%Total"}
	flagHeader = []string{"Relation", "Page Count", "Flags", "Symbolic Flags",
		"Long Symbolic Flags"}
)

func (p *PgPageCache) outputColumns(values [][]string, outputInfos []relation.OutputInfo) {
	w := tabwriter.NewWriter(os.Stdout, 14, 0, 1, ' ', 0)
	for _, v := range values {
		fmt.Fprintln(w, strings.Join(v, "\t"))
	}
	w.Flush()

	if p.pageCacheState.CanReadPageFlags() && !p.GroupTable {
		fmt.Printf("\nPage Flags\n")
		fmt.Fprintln(w, strings.Join(flagHeader, "\t"))
		for _, v := range outputInfos {
			for _, flag := range v.ToFlagDetails() {
				fmt.Fprintln(w, strings.Join(flag, "\t"))
			}
		}
		w.Flush()
	}
}

func (p *PgPageCache) outputJSON(header []string, values [][]string) error {
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
	return nil
}

// AdjustLine remove unecessary output lines
// When grouping table, relation and relfilenode will always be empty
func (p *PgPageCache) AdjustLine(line []string) []string {
	var res []string
	// Partition + table
	res = append(res, line[0:2]...)
	if !p.GroupTable {
		// Relation + relfilenode
		res = append(res, line[2:4]...)
	}
	res = append(res, line[4:]...)
	return res
}

func (p *PgPageCache) outputResults(outputInfos []relation.OutputInfo) error {
	var values [][]string

	header := p.AdjustLine(pageHeader)
	if !p.NoHeader && p.Type != FormatJSON {
		values = append(values, header)
	}

	for _, v := range outputInfos {
		line := v.ToStringArray(p.Unit, p.pageSize, p.fileMemory)
		values = append(values, p.AdjustLine(line))
	}

	switch p.Type {
	case FormatCSV:
		w := csv.NewWriter(os.Stdout)
		w.WriteAll(values)
		return w.Error()
	case FormatJSON:
		return p.outputJSON(header, values)
	case FormatColumn:
		p.outputColumns(values, outputInfos)
	}
	return nil
}
