package relation

import "github.com/bonnefoa/pg_pagecache/pcstats"

type RelInfo struct {
	Relfilenode uint32
	Relname     string
	Relkind     rune
	PcStats     pcstats.PcStats
	Children    []string
}

type FileToRelinfo map[uint32]RelInfo
