package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"log/slog"

	"github.com/bonnefoa/pg_pagecache/meminfo"
	"github.com/bonnefoa/pg_pagecache/pcstats"
	"github.com/bonnefoa/pg_pagecache/relation"
	"github.com/jackc/pgx/v5"
)

type PgPagecache struct {
	CliArgs
	conn *pgx.Conn

	dbid            uint32
	database        string
	page_size       int64
	cached_memory   int64
	tableToRelinfos relation.TableToRelinfos
}

func (p *PgPagecache) fillRelinfoPcStats(relinfo *relation.RelInfo) (err error) {
	baseDir := fmt.Sprintf("%s/base/%d", p.PgData, p.dbid)
	segno := 0

	for {
		filename := fmt.Sprintf("%d", relinfo.Relfilenode)
		if segno > 0 {
			filename = fmt.Sprintf("%d.%d", relinfo.Relfilenode, segno)
		}
		segno++
		fullPath := filepath.Join(baseDir, filename)
		_, err = os.Stat(fullPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				// Last segment was processed, exit
				return nil
			}
			return
		}

		relinfo.PcStats, err = pcstats.GetPcStats(fullPath, p.page_size)
		if err != nil {
			return err
		}
	}
}

// fillTablesPcStats iterate over tableToRelinfos and fetch page cache stats
func (p *PgPagecache) fillTablesPcStats() error {
	newTableToRelinfos := make(relation.TableToRelinfos, 0)

	for t, relinfos := range p.tableToRelinfos {
		var filteredRelinfo []*relation.RelInfo

		for _, relinfo := range relinfos {
			err := p.fillRelinfoPcStats(relinfo)
			if err != nil {
				return err
			}
			if relinfo.PcStats.PageCached >= p.CachedPageThreshold {
				filteredRelinfo = append(filteredRelinfo, relinfo)
			}
			t.PcStats.Add(relinfo.PcStats)
		}

		newTableToRelinfos[t] = filteredRelinfo
	}

	slog.Info("Pagestats finished")
	p.tableToRelinfos = newTableToRelinfos
	return nil
}

// NewPgPagecache fetches the active database id and name and creates the PgPagecache instance
func NewPgPagecache(conn *pgx.Conn, cliArgs CliArgs) (pgPagecache PgPagecache, err error) {
	pgPagecache.conn = conn
	pgPagecache.CliArgs = cliArgs
	return
}

func (p *PgPagecache) Run(ctx context.Context) (err error) {
	// Fetch dbid and database
	err = p.conn.QueryRow(ctx, "select oid, datname from pg_database where datname=current_database()").Scan(&p.dbid, &p.database)
	if err != nil {
		err = fmt.Errorf("Error getting current database: %v\n", err)
		return
	}
	slog.Info("Fetched database details", "database", p.database, "dbid", p.dbid)

	// Fill the table -> Relinfos map
	p.tableToRelinfos, err = relation.GetTableToRelinfo(ctx, p.conn, p.Relations, p.PageThreshold)
	if err != nil {
		err = fmt.Errorf("Error getting table to relinfos mapping: %v\n", err)
		return
	}
	slog.Info("Found relations matching page threshold", "numbers", len(p.tableToRelinfos), "page_threshold", p.PageThreshold)

	// Detect page size
	p.page_size = pcstats.GetPageSize()
	slog.Info("Detected Page size", "page_size", p.page_size)

	// Go through all tables and fill their PcStats
	err = p.fillTablesPcStats()
	if err != nil {
		return
	}

	p.cached_memory, err = meminfo.GetCachedMemory(p.page_size)
	if err != nil {
		slog.Warn("Couldn't get cached_memory", "error", err)
	} else {
		slog.Info("Detected cached memory usage", "cached_memory", p.cached_memory)
	}

	// Filter out relations under the cached threshold
	for k, relinfos := range p.tableToRelinfos {
		for _, relinfo := range relinfos {
			if relinfo.PcStats.PageCached <= p.CachedPageThreshold {
				delete(p.tableToRelinfos, k)
			}

		}
	}

	if p.Aggregation == AggNone {
		return p.formatNoAggregation()
	} else {
		return p.formatAggregated()
	}
}
