package app

import (
	"flag"
	"fmt"
	"strings"
)

type DisplayAggregation int
type DisplaySort int

const (
	SortRelation DisplaySort = iota
	SortPageCached
	SortPageCount
)

const (
	DisplayRelation DisplayAggregation = iota
	DisplayOnlyParent
	DisplayParentWithChildren
)

var (
	sortOutputMap = map[string]DisplaySort{
		"relation":   SortRelation,
		"pagecached": SortPageCached,
		"pagecount":  SortPageCount,
	}

	displayAggregationMap = map[string]DisplayAggregation{
		"relation":             DisplayRelation,
		"parent_only":          DisplayOnlyParent,
		"parent_with_children": DisplayParentWithChildren,
	}
)

var (
	sortFlag        string
	aggregationFlag string
	thresholdFlag   int
)

func init() {
	flag.IntVar(&thresholdFlag, "threshold", 0, "Don't display file if cached pages are under the provided threshold. -1 to display everything")
	flag.StringVar(&sortFlag, "sort", "pagecached", "Field to use for sort. Can be relation, pagecount or pagecached")
	flag.StringVar(&aggregationFlag, "aggregation", "relation", "How to aggregate results. relation, parent_only, parent_with_children")
}

type DisplayOptions struct {
	Group       bool
	Threshold   int
	Sort        DisplaySort
	Aggregation DisplayAggregation
}

func parseSortOutput(s string) (DisplaySort, error) {
	sortOutput, ok := sortOutputMap[strings.ToLower(s)]
	if !ok {
		err := fmt.Errorf("Unknown sort: %v\n", s)
		return sortOutput, err
	}
	return sortOutput, nil
}

func parseDisplayAggregation(s string) (DisplayAggregation, error) {
	agg, ok := displayAggregationMap[strings.ToLower(s)]
	if !ok {
		err := fmt.Errorf("Unknown aggregation: %v\n", s)
		return agg, err
	}
	return agg, nil
}

func ParseDisplayOptions() (displayOptions DisplayOptions, err error) {
	displayOptions.Sort, err = parseSortOutput(sortFlag)
	if err != nil {
		return
	}
	displayOptions.Aggregation, err = parseDisplayAggregation(aggregationFlag)
	if err != nil {
		return
	}
	displayOptions.Threshold = thresholdFlag
	return
}
