package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"log/slog"

	"github.com/bonnefoa/pg_pagecache/memory"
	"github.com/bonnefoa/pg_pagecache/pagecache"
	"github.com/bonnefoa/pg_pagecache/relation"
	"github.com/bonnefoa/pg_pagecache/utils"
	"github.com/jackc/pgx/v5"
)

// PgPageCache stores the args and state of the pg_pagecache app
type PgPageCache struct {
	CliArgs
	conn *pgx.Conn

	dbid           uint32
	database       string
	pageSize       int64
	fileMemory     int64 // File backed memory in KB
	partitions     map[string]relation.PartInfo
	pageCacheState pagecache.State
}

func (p *PgPageCache) fillRelinfo(relinfo *relation.RelInfo) (err error) {
	baseDir := path.Join(p.PgData, "base", fmt.Sprintf("%d", p.dbid))
	_, err = os.Stat(baseDir)
	if err != nil {
		err = fmt.Errorf("Incorrect pg_data path: %v", err)
		return
	}

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
			if errors.Is(err, os.ErrNotExist) && segno > 0 {
				// Last segment was processed, exit
				return nil
			}
			return
		}

		segmentPcStats, err := p.pageCacheState.GetPageCacheInfo(fullPath, p.pageSize)
		if err != nil {
			return err
		}
		relinfo.Add(segmentPcStats)
	}
}

func (p *PgPageCache) fillTableStats(table *relation.TableInfo) error {
	var filteredRelinfo []relation.RelInfo

	for _, relinfo := range table.RelInfos {
		err := p.fillRelinfo(&relinfo)
		if err != nil {
			return err
		}
		if relinfo.PageCached >= p.CachedPageThreshold {
			filteredRelinfo = append(filteredRelinfo, relinfo)
		}
		table.Add(relinfo.PageStats)
	}
	table.RelInfos = filteredRelinfo

	return nil
}

// fillPartitionStats iterate over tableToRelinfos and fetch page cache stats
func (p *PgPageCache) fillPartitionStats() error {
	for partName, partInfo := range p.partitions {
		for tableName, tableInfo := range partInfo.TableInfos {
			err := p.fillTableStats(&tableInfo)
			if err != nil {
				return err
			}
			partInfo.Add(tableInfo.PageStats)
			partInfo.TableInfos[tableName] = tableInfo
		}
		p.partitions[partName] = partInfo
	}
	return nil
}

// getWalPageStats fetches page cache usage of WAL files
func (p *PgPageCache) getWalPageStats() (walPageStats pagecache.PageStats, err error) {
	baseDir := path.Join(p.PgData, "pg_wal")
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		err = fmt.Errorf("Error listing file: %v", err)
		return
	}

	for _, entry := range entries {
		var pageStats pagecache.PageStats
		var fsInfo os.FileInfo
		fullPath := path.Join(baseDir, entry.Name())
		fsInfo, err = os.Stat(fullPath)
		if err != nil {
			return
		}
		if fsInfo.IsDir() {
			continue
		}
		pageStats, err = p.pageCacheState.GetPageCacheInfo(fullPath, p.pageSize)
		if err != nil {
			return
		}
		walPageStats.Add(pageStats)
	}
	return
}

// NewPgPagecache fetches the active database id and name and creates the pgPageCache instance
func NewPgPagecache(conn *pgx.Conn, cliArgs CliArgs) (pgPagecache PgPageCache) {
	pgPagecache.conn = conn
	pgPagecache.CliArgs = cliArgs
	pgPagecache.pageCacheState = pagecache.NewPageCacheState(cliArgs.RawFlags)
	return
}

func (p *PgPageCache) getOutputInfos() (outputInfos []relation.OutputInfo) {
	if p.GroupPartition {
		outputInfos = p.getAggregatedPartitions()
	} else if p.GroupTable {
		outputInfos = p.getAggregatedTables()
	} else {
		outputInfos = p.getNoAggregations()
	}
	if p.ScanWal {
		outputInfos = append(outputInfos, &relation.WalInfo)
	}
	outputInfos = append(outputInfos, &relation.TotalInfo)
	return
}

// Run executes the pg_pagecache. It will fetch database and relation
// informations from the running postgres, then fetch page cache stats
// on those relations
func (p *PgPageCache) Run(ctx context.Context) (err error) {
	// Fetch dbid and database
	err = p.conn.QueryRow(ctx, "select oid, datname from pg_database where datname=current_database()").Scan(&p.dbid, &p.database)
	if err != nil {
		err = fmt.Errorf("error getting current database: %v", err)
		return
	}
	slog.Info("Fetched database details", "database", p.database, "dbid", p.dbid)

	// Fill the partition -> []Table map
	p.partitions, err = relation.GetPartitionToTables(ctx, p.conn, p.Relations, p.PageThreshold)
	if err != nil {
		err = fmt.Errorf("error getting table to relinfos mapping: %v", err)
		return
	}

	// Detect page size
	p.pageSize = pagecache.GetPageSize()
	slog.Info("Detected Page size", "pageSize", p.pageSize)

	// Go through all tables and fill their pagecache
	err = p.fillPartitionStats()
	if err != nil {
		return
	}

	if p.ScanWal {
		// Get pagecache usage of wal files
		relation.WalInfo.PageStats, err = p.getWalPageStats()
		if err != nil {
			return
		}
	}

	p.fileMemory, err = memory.GetCachedMemory(p.pageSize)
	if err != nil {
		return fmt.Errorf("Couldn't get cached_memory: %v", err)
	}
	slog.Info("Detected cached memory usage", "cache_memory", utils.FormatKBValue(p.fileMemory, utils.UnitGB))

	// Filter partitions under the threshold
	filteredPartInfos := make(map[string]relation.PartInfo, 0)
	for partName, partInfo := range p.partitions {
		if partInfo.PageCached > p.CachedPageThreshold {
			filteredPartInfos[partName] = partInfo
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

	outputInfos := p.getOutputInfos()
	return p.outputResults(outputInfos)
}
