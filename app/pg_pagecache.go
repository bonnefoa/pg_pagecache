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

	dbid          uint32
	database      string
	page_size     int64
	cached_memory int64
	partitions    []relation.PartInfo
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

		segmentPcStats, err := pcstats.GetPcStats(fullPath, p.page_size)
		if err != nil {
			return err
		}
		relinfo.PcStats.Add(segmentPcStats)
	}
}

func (p *PgPagecache) fillTableStats(table *relation.TableInfo) error {
	var filteredRelinfo []relation.RelInfo

	for _, relinfo := range table.RelInfos {
		err := p.fillRelinfoPcStats(&relinfo)
		if err != nil {
			return err
		}
		if relinfo.PcStats.PageCached >= p.CachedPageThreshold {
			filteredRelinfo = append(filteredRelinfo, relinfo)
		}
		table.PcStats.Add(relinfo.PcStats)
	}
	table.RelInfos = filteredRelinfo

	return nil
}

// fillPartitionPcStats iterate over tableToRelinfos and fetch page cache stats
func (p *PgPagecache) fillPartitionPcStats() error {
	for partName, partInfo := range p.partitions {
		for tableName, tableInfo := range partInfo.TableInfos {
			err := p.fillTableStats(&tableInfo)
			if err != nil {
				return err
			}
			partInfo.PcStats.Add(tableInfo.PcStats)
			partInfo.TableInfos[tableName] = tableInfo
		}
		p.partitions[partName] = partInfo
	}
	slog.Info("Pagestats finished")
	return nil
}

// NewPgPagecache fetches the active database id and name and creates the PgPagecache instance
func NewPgPagecache(conn *pgx.Conn, cliArgs CliArgs) (pgPagecache PgPagecache, err error) {
	pgPagecache.conn = conn
	pgPagecache.CliArgs = cliArgs
	return
}

func (p *PgPagecache) getOutputInfos() ([]relation.OutputInfo, error) {
	switch p.Aggregation {
	case AggNone:
		return p.formatNoAggregation()

	case AggPartition:
		fallthrough
	case AggPartitionOnly:
		return p.formatAggregatePartitions()

	case AggTable:
		fallthrough
	case AggTableOnly:
		return p.formatAggregatedTables()
	}
	panic("Unreachable code")
}

func (p *PgPagecache) Run(ctx context.Context) (err error) {
	// Fetch dbid and database
	err = p.conn.QueryRow(ctx, "select oid, datname from pg_database where datname=current_database()").Scan(&p.dbid, &p.database)
	if err != nil {
		err = fmt.Errorf("Error getting current database: %v\n", err)
		return
	}
	slog.Info("Fetched database details", "database", p.database, "dbid", p.dbid)

	// Fill the partition -> []Table map
	p.partitions, err = relation.GetPartitionToTables(ctx, p.conn, p.Relations, p.PageThreshold)
	if err != nil {
		err = fmt.Errorf("Error getting table to relinfos mapping: %v\n", err)
		return
	}

	// Detect page size
	p.page_size = pcstats.GetPageSize()
	slog.Info("Detected Page size", "page_size", p.page_size)

	// Go through all tables and fill their PcStats
	err = p.fillPartitionPcStats()
	if err != nil {
		return
	}

	p.cached_memory, err = meminfo.GetCachedMemory(p.page_size)
	if err != nil {
		slog.Warn("Couldn't get cached_memory", "error", err)
	} else {
		slog.Info("Detected cached memory usage", "cached_memory", p.cached_memory)
	}

	// Filter partitions under the threshold
	var filteredPartInfos []relation.PartInfo
	for _, partInfo := range p.partitions {
		if partInfo.PcStats.PageCached > p.CachedPageThreshold {
			filteredPartInfos = append(filteredPartInfos, partInfo)
			continue
		}
	}
	p.partitions = filteredPartInfos

	for _, partInfo := range p.partitions {
		for tableName, tableInfo := range partInfo.TableInfos {
			// Filter table under the threshold
			if tableInfo.PcStats.PageCached <= p.CachedPageThreshold {
				delete(partInfo.TableInfos, tableName)
				continue
			}

			// Filter relinfos under the threshold
			var filteredRelinfos []relation.RelInfo
			for _, relinfo := range tableInfo.RelInfos {
				if relinfo.PcStats.PageCached > p.CachedPageThreshold {
					filteredRelinfos = append(filteredRelinfos, relinfo)
				}
			}
			tableInfo.RelInfos = filteredRelinfos
			partInfo.TableInfos[tableName] = tableInfo
		}
	}

	outputInfos, err := p.getOutputInfos()
	if err != nil {
		return err
	}
	return p.outputResults(outputInfos)
}
