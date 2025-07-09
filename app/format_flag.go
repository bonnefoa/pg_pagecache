package app

import (
	"flag"
	"fmt"
	"strings"

	"github.com/bonnefoa/pg_pagecache/relation"
)

type FormatSort int
type FormatType int

type FormatOptions struct {
	Unit        relation.FormatUnit
	Limit       int
	Type        FormatType
	Sort        FormatSort
	NoHeader    bool
	Aggregation relation.AggregationType
}

const (
	SortName FormatSort = iota
	SortPageCached
	SortPageCount

	FormatCSV = iota
	FormatColumn
	FormatJson
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
		"json":   FormatJson,
	}

	formatOptions   FormatOptions
	typeFlag        string
	unitFlag        string
	sortFlag        string
	aggregationFlag string
)

func init() {
	flag.IntVar(&formatOptions.Limit, "limit", -1, "Maximum number of results to format. -1 to format everything.")
	flag.BoolVar(&formatOptions.NoHeader, "no_header", false, "Don't print header if true.")
	flag.StringVar(&typeFlag, "format", "column", "Output format to use. Can be csv, column or json")
	flag.StringVar(&unitFlag, "unit", "page", "Unit to use for paeg count and page cached. Can be page, kb or MB")
	flag.StringVar(&sortFlag, "sort", "pagecached", "Field to use for sort. Can be relation, pagecount or pagecached")
	flag.StringVar(&aggregationFlag, "aggregation", "none", "How to aggregate results. relation, table, table_only")
}

func parseSort(s string) (FormatSort, error) {
	sortOutput, ok := formatSortMap[strings.ToLower(s)]
	if !ok {
		err := fmt.Errorf("Unknown sort: %v\n", s)
		return sortOutput, err
	}
	return sortOutput, nil
}

func parseUnitFlag(s string) (relation.FormatUnit, error) {
	res, ok := formatUnitMap[strings.ToLower(s)]
	if !ok {
		err := fmt.Errorf("Unknown unit: %v\n", s)
		return res, err
	}
	return res, nil
}

func parseTypeFlag(s string) (FormatType, error) {
	res, ok := formatTypeMap[strings.ToLower(s)]
	if !ok {
		err := fmt.Errorf("Unknown format type: %v\n", s)
		return res, err
	}
	return res, nil
}

func ParseFormatOptions() (FormatOptions, error) {
	var err error
	formatOptions.Sort, err = parseSort(sortFlag)
	if err != nil {
		return formatOptions, err
	}
	formatOptions.Aggregation, err = relation.ParseAggregation(aggregationFlag)
	if err != nil {
		return formatOptions, err
	}
	formatOptions.Unit, err = parseUnitFlag(unitFlag)
	if err != nil {
		return formatOptions, err
	}
	formatOptions.Type, err = parseTypeFlag(typeFlag)
	if err != nil {
		return formatOptions, err
	}
	return formatOptions, err
}
