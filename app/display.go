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

func (p *PgPageCache) outputResults(outputInfos []relation.OutputInfo) error {
	var values [][]string

	header := relation.GetHeader(p.Aggregation)
	if !p.NoHeader && p.Type != FormatJSON {
		values = append(values, header)
	}

	for _, v := range outputInfos {
		line := v.ToStringArray(p.Aggregation, p.Unit, p.pageSize, p.fileMemory)
		values = append(values, line)
	}

	switch p.Type {
	case FormatCSV:
		w := csv.NewWriter(os.Stdout)
		w.WriteAll(values)
		return w.Error()
	case FormatJSON:
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

		fmt.Printf("\nPage Flags\n")
		fmt.Fprintln(w, strings.Join([]string{"Relation", "Flags", "Symbolic Flags", "Count"}, "\t"))
		for _, v := range outputInfos {
			for _, flag := range v.ToFlagDetails() {
				fmt.Fprintln(w, strings.Join(flag, "\t"))
			}
		}
		w.Flush()

	}
	return nil
}
