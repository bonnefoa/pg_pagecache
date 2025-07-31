package utils

import (
	"fmt"
	"strconv"
)

// Unit represents the unit used for output
type Unit int

const (
	// UnitPage outputs values in pages
	UnitPage Unit = iota
	// UnitKB outputs values in KB
	UnitKB
	// UnitMB outputs values in MB
	UnitMB
	// UnitGB outputs values in GB
	UnitGB

	kebibyte = float64(1 << 10)
	mebibyte = float64(1 << 20)
	gebibyte = float64(1 << 30)
)

func unitToString(u Unit) string {
	switch u {
	case UnitPage:
		return "Pgs"
	case UnitKB:
		return "KB"
	case UnitMB:
		return "MB"
	case UnitGB:
		return "GB"
	}
	return "?"
}

func FormatPageValue(value int, unit Unit, pageSize int64) (valueStr string) {
	switch unit {
	case UnitPage:
		valueStr = strconv.FormatInt(int64(value), 10)
	case UnitKB:
		valueStr = strconv.FormatFloat(float64(int64(value)*pageSize)/kebibyte, 'f', -1, 64)
	case UnitMB:
		valueStr = strconv.FormatFloat(float64(int64(value)*pageSize)/mebibyte, 'f', 2, 64)
	case UnitGB:
		valueStr = strconv.FormatFloat(float64(int64(value)*pageSize)/gebibyte, 'f', 2, 64)
	}
	return fmt.Sprintf("%s%s", valueStr, unitToString(unit))
}

func FormatKBValue(value int64, unit Unit) (valueStr string) {
	valueBytes := float64(value) * kebibyte
	switch unit {
	case UnitPage:
		valueStr = strconv.FormatInt(value, 10)
	case UnitKB:
		valueStr = strconv.FormatFloat(valueBytes/kebibyte, 'f', -1, 64)
	case UnitMB:
		valueStr = strconv.FormatFloat(valueBytes/mebibyte, 'f', 2, 64)
	case UnitGB:
		valueStr = strconv.FormatFloat(valueBytes/gebibyte, 'f', 2, 64)
	}
	return fmt.Sprintf("%s%s", valueStr, unitToString(unit))
}
