package app

import (
	"flag"
	"fmt"
	"strings"
)

type OutputAggregation int
type OutputSort int

const (
	SortRelation OutputSort = iota
	SortPageCached
	SortPageCount
)

const (
	AggRelation OutputAggregation = iota
	AggOnlyParent
	AggParentWithChildren
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

	formatOptions   OutputOptions
	sortFlag        string
	aggregationFlag string
)

func init() {
	flag.IntVar(&formatOptions.Threshold, "threshold", 0, "Don't format file if cached pages are under the provided threshold. -1 to format everything")
	flag.IntVar(&formatOptions.Limit, "limit", -1, "Maximum number of results to format. -1 to format everything.")
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
	return formatOptions, err
}
