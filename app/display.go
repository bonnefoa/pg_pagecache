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

func (p *PgPagecache) outputResults(outputInfos []relation.OutputInfo) error {
	var values [][]string

	header := p.getHeader()
	if !p.NoHeader && p.Type != FormatJson {
		values = append(values, header)
	}

	for _, v := range outputInfos {
		line := v.ToStringArray(p.Unit, p.page_size, p.file_memory)
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
