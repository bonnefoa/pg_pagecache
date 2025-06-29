package relation

import "github.com/bonnefoa/pg_pagecache/pcstats"

type RelInfo struct {
	Relfilenode uint32
	Relname     string
	PcStats     pcstats.PcStats
	Children    []string
}

type RelToRelinfo map[string]RelInfo
type FileToRelinfo map[uint32]RelInfo
