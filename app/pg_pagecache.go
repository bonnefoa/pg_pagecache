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

	dbid        uint32
	database    string
	page_size   int64
	file_memory int64 // Cache memory without shared_buffers
	partitions  []relation.PartInfo
	pcState     pcstats.PcState
}

// getSharedBuffers returns the amount of shared_buffers memory in 4KB pages
func (p *PgPagecache) getSharedBuffers(ctx context.Context, conn *pgx.Conn) (shared_buffers int, err error) {
	row := conn.QueryRow(ctx, "SELECT setting::int FROM pg_settings where name='shared_buffers'")
	err = row.Scan(&shared_buffers)

	// pg_settings uses 8Kb blocks, we want 4KB pages
	return shared_buffers / 2, err
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

		segmentPcStats, err := p.pcState.GetPcStats(fullPath, p.page_size)
		if err != nil {
			return err
		}
		relinfo.Add(segmentPcStats)
	}
}

func (p *PgPagecache) fillTableStats(table *relation.TableInfo) error {
	var filteredRelinfo []relation.RelInfo

	for _, relinfo := range table.RelInfos {
		err := p.fillRelinfoPcStats(&relinfo)
		if err != nil {
			return err
		}
		if relinfo.PageCached >= p.CachedPageThreshold {
			filteredRelinfo = append(filteredRelinfo, relinfo)
		}
		table.Add(relinfo.PageCacheInfo)
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
			partInfo.Add(tableInfo.PageCacheInfo)
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
	pgPagecache.pcState, err = pcstats.NewPcState()
	return
}

func (p *PgPagecache) getOutputInfos() ([]relation.OutputInfo, error) {
	switch p.Aggregation {
	case relation.AggNone:
		return p.formatNoAggregation()

	case relation.AggPartition:
		fallthrough
	case relation.AggPartitionOnly:
		return p.formatAggregatePartitions()

	case relation.AggTable:
		fallthrough
	case relation.AggTableOnly:
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

	cached_memory, err := meminfo.GetCachedMemory(p.page_size)
	if err != nil {
		return fmt.Errorf("Couldn't get cached_memory: %v", err)
	}
	slog.Info("Detected cached memory usage", "cached_memory", cached_memory)

	shared_buffers, err := p.getSharedBuffers(ctx, p.conn)
	if err != nil {
		return fmt.Errorf("Couldn't get shared_buffers: %v", err)
	}
	slog.Info("Detected shared_buffers", "shared_buffers", shared_buffers)
	p.file_memory = cached_memory - int64(shared_buffers)

	// Filter partitions under the threshold
	var filteredPartInfos []relation.PartInfo
	for _, partInfo := range p.partitions {
		if partInfo.PageCached > p.CachedPageThreshold {
			filteredPartInfos = append(filteredPartInfos, partInfo)
			continue
		}
	}
	p.partitions = filteredPartInfos

	for _, partInfo := range p.partitions {
		for tableName, tableInfo := range partInfo.TableInfos {
			// Filter table under the threshold
			if tableInfo.PageCached <= p.CachedPageThreshold {
				delete(partInfo.TableInfos, tableName)
				continue
			}

			// Filter relinfos under the threshold
			var filteredRelinfos []relation.RelInfo
			for _, relinfo := range tableInfo.RelInfos {
				if relinfo.PageCached > p.CachedPageThreshold {
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
