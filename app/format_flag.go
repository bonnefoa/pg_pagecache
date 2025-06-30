package app

import (
	"flag"
	"fmt"
	"strings"
)

type OutputAggregation int
type OutputSort int
type FormatUnit int

const (
	SortRelation OutputSort = iota
	SortPageCached
	SortPageCount
)

const (
	AggRelation OutputAggregation = iota
	AggOnlyParent
	AggParentWithChildren

	UnitPage FormatUnit = iota
	UnitKB
	UnitMB
	UnitGB
)

var (
	sortOutputMap = map[string]OutputSort{
		"relation":   SortRelation,
		"pagecached": SortPageCached,
		"pagecount":  SortPageCount,
	}

	formatAggregationMap = map[string]OutputAggregation{
		"relation":             AggRelation,
		"parent_only":          AggOnlyParent,
		"parent_with_children": AggParentWithChildren,
	}

	formatUnitMap = map[string]FormatUnit{
		"page": UnitPage,
		"kb":   UnitKB,
		"mb":   UnitMB,
		"gb":   UnitGB,
	}

	formatOptions   OutputOptions
	unitFlag        string
	sortFlag        string
	aggregationFlag string
)

func init() {
	flag.IntVar(&formatOptions.Threshold, "threshold", 0, "Don't format file if cached pages are under the provided threshold. -1 to format everything")
	flag.IntVar(&formatOptions.Limit, "limit", -1, "Maximum number of results to format. -1 to format everything.")
	flag.StringVar(&unitFlag, "unit", "page", "Unit to use for paeg count and page cached. Can be page, kb or MB")
	flag.StringVar(&sortFlag, "sort", "pagecached", "Field to use for sort. Can be relation, pagecount or pagecached")
	flag.StringVar(&aggregationFlag, "aggregation", "relation", "How to aggregate results. relation, parent_only, parent_with_children")
}

type OutputOptions struct {
	Threshold   int
	Limit       int
	Sort        OutputSort
	Aggregation OutputAggregation
}

func parseSortOutput(s string) (OutputSort, error) {
	sortOutput, ok := sortOutputMap[strings.ToLower(s)]
	if !ok {
		err := fmt.Errorf("Unknown sort: %v\n", s)
		return sortOutput, err
	}
	return sortOutput, nil
}

func parseOutputAggregation(s string) (OutputAggregation, error) {
	agg, ok := formatAggregationMap[strings.ToLower(s)]
	if !ok {
		err := fmt.Errorf("Unknown aggregation: %v\n", s)
		return agg, err
	}
	return agg, nil
}

func parseUnitFlag(s string) (FormatUnit, error) {
	res, ok := formatUnitMap[strings.ToLower(s)]
	if !ok {
		err := fmt.Errorf("Unknown unit: %v\n", s)
		return res, err
	}
	return res, nil
}

func ParseOutputOptions() (OutputOptions, error) {
	var err error
	formatOptions.Sort, err = parseSortOutput(sortFlag)
	if err != nil {
		return formatOptions, err
	}
	formatOptions.Aggregation, err = parseOutputAggregation(aggregationFlag)
	if err != nil {
		return formatOptions, err
	}
	formatOptions.Unit, err = parseUnitFlag(unitFlag)
	if err != nil {
		return formatOptions, err
	}
	return formatOptions, err
}
