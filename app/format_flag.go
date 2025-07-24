package app

import (
	"flag"
	"fmt"
	"strings"

	"github.com/bonnefoa/pg_pagecache/relation"
)

// FormatSort represents the different sort options
type FormatSort int

// FormatType represents the different type options
type FormatType int

// FormatFlags stores all format related flags
type FormatFlags struct {
	Unit           relation.FormatUnit
	Limit          int
	Type           FormatType
	Sort           FormatSort
	NoHeader       bool
	GroupTable     bool
	GroupPartition bool
}

const (
	// SortName sorts by name
	SortName FormatSort = iota
	// SortPageCached sorts by number of cached pages
	SortPageCached
	// SortPageCount sorts by number of pages
	SortPageCount

	// FormatCSV outputs the result using CSV
	FormatCSV = iota
	// FormatColumn outputs the result using aligned column
	FormatColumn
	// FormatJSON outputs the result using JSON
	FormatJSON
)

var (
	formatSortMap = map[string]FormatSort{
		"name":       SortName,
		"pagecached": SortPageCached,
		"pagecount":  SortPageCount,
	}

	formatUnitMap = map[string]relation.FormatUnit{
		"page": relation.UnitPage,
		"kb":   relation.UnitKB,
		"mb":   relation.UnitMB,
		"gb":   relation.UnitGB,
	}

	formatTypeMap = map[string]FormatType{
		"csv":    FormatCSV,
		"column": FormatColumn,
		"json":   FormatJSON,
	}

	formatFlags     FormatFlags
	typeFlag        string
	unitFlag        string
	sortFlag        string
	aggregationFlag string
)

func init() {
	flag.IntVar(&formatFlags.Limit, "limit", -1, "Maximum number of results to format. -1 to format everything.")
	flag.BoolVar(&formatFlags.NoHeader, "no_header", false, "Don't print header.")
	flag.BoolVar(&formatFlags.GroupPartition, "group_partition", false, "Group partition.")
	flag.BoolVar(&formatFlags.GroupTable, "group_table", false, "Group indexes, toast with owning relation.")
	flag.StringVar(&typeFlag, "format", "column", "Output format to use. Can be csv, column or json")
	flag.StringVar(&unitFlag, "unit", "mb", "Unit to use for paeg count and page cached. Can be page, kb, mb or gb")
	flag.StringVar(&sortFlag, "sort", "pagecached", "Field to use for sort. Can be relation, pagecount or pagecached")
	flag.StringVar(&aggregationFlag, "aggregation", "none", "How to aggregate results. relation, table, table_only")
}

func parseSort(s string) (FormatSort, error) {
	sortOutput, ok := formatSortMap[strings.ToLower(s)]
	if !ok {
		err := fmt.Errorf("unknown sort: %v", s)
		return sortOutput, err
	}
	return sortOutput, nil
}

func parseUnitFlag(s string) (relation.FormatUnit, error) {
	res, ok := formatUnitMap[strings.ToLower(s)]
	if !ok {
		err := fmt.Errorf("unknown unit: %v", s)
		return res, err
	}
	return res, nil
}

func parseTypeFlag(s string) (FormatType, error) {
	res, ok := formatTypeMap[strings.ToLower(s)]
	if !ok {
		err := fmt.Errorf("unknown format type: %v", s)
		return res, err
	}
	return res, nil
}

// ParseFormatOptions parses and return FormatFlags with parsed values
func ParseFormatOptions() (FormatFlags, error) {
	var err error
	formatFlags.Sort, err = parseSort(sortFlag)
	if err != nil {
		return formatFlags, err
	}
	formatFlags.Unit, err = parseUnitFlag(unitFlag)
	if err != nil {
		return formatFlags, err
	}
	formatFlags.Type, err = parseTypeFlag(typeFlag)
	if err != nil {
		return formatFlags, err
	}
	return formatFlags, err
}
