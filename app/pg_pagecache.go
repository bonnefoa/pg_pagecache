package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"log/slog"

	"github.com/bonnefoa/pg_pagecache/pcstats"
	"github.com/bonnefoa/pg_pagecache/relation"
	"github.com/jackc/pgx/v5"
)

type PgPagecache struct {
	conn          *pgx.Conn
	pgData        string
	OutputOptions OutputOptions

	dbid          uint32
	database      string
	fileToRelinfo relation.FileToRelinfo
	relToRelinfo  relation.RelToRelinfo
}

func extractRelfilenode(filename string) (relfilenode uint32, err error) {
	// Remove possible segment number
	relid := strings.Split(filename, ".")[0]
	relfilenodeUint64, err := strconv.ParseUint(relid, 10, 32)
	relfilenode = uint32(relfilenodeUint64)
	if err != nil {
		return
	}
	if relfilenode == 0 {
		slog.Debug("Not a relation file, ignoring", "filename", filename)
		return
	}
	return
}

// fillPcStats iterate over fileToRelinfo and fetch page cache stats
func (p *PgPagecache) fillPcStats() error {
	baseDir := fmt.Sprintf("%s/base/%d", p.pgData, p.dbid)
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return fmt.Errorf("Error listing file: %v", err)
	}

	num_entries := len(entries)
	slog.Info("Listed relation files", "num_files", num_entries)
	for k, entrie := range entries {
		filename := entrie.Name()

		if strings.Contains(filename, "_") {
			// Ignore FSM and VM forks
			continue
		}

		relfilenode, err := extractRelfilenode(filename)
		if err != nil {
			return err
		}
		if relfilenode == 0 {
			continue
		}

		// Get the matching relinfo
		relinfo, ok := p.fileToRelinfo[relfilenode]
		if !ok {
			// relfilenode was filtered out, skip it
			continue
		}

		fullPath := filepath.Join(baseDir, filename)
		pcStats, err := pcstats.GetPcStats(fullPath)
		if err != nil {
			return err
		}
		if pcStats.PageCount == 0 {
			// No page at all, skip it
			continue
		}
		if k%1000 == 0 {
			slog.Info("Getting pagestas", "current_entry", k, "num_entries", num_entries)
		}

		relinfo.PcStats.Add(pcStats)
		p.fileToRelinfo[relfilenode] = relinfo
		slog.Debug("Adding relinfo", "Relation", relinfo.Relname, "filename", filename, "pagecached", pcStats.PageCached)
	}
	return nil
}

// NewPgPagecache fetches the active database id and name and creates the PgPagecache instance
func NewPgPagecache(ctx context.Context, conn *pgx.Conn, cliArgs CliArgs) (pgPagecache PgPagecache, err error) {
	pgPagecache.conn = conn
	pgPagecache.pgData = cliArgs.PgData
	pgPagecache.OutputOptions = cliArgs.OutputOptions

	// Fetch dbid and database
	err = conn.QueryRow(ctx, "select oid, datname from pg_database where datname=current_database()").Scan(&pgPagecache.dbid, &pgPagecache.database)
	if err != nil {
		err = fmt.Errorf("Error getting current database: %v\n", err)
		return
	}
	slog.Debug("Fetched database details", "database", pgPagecache.database, "dbid", pgPagecache.dbid)

	// Fill the file -> relinfo map
	pgPagecache.fileToRelinfo, err = relation.GetFileToRelinfo(ctx, conn, cliArgs.Relations, cliArgs.PageThreshold)
	if err != nil {
		err = fmt.Errorf("Error getting file to relation mapping: %v\n", err)
		return
	}
	slog.Debug("Fetched fileToRelinfo", "length", len(pgPagecache.fileToRelinfo))

	return
}

func (p *PgPagecache) Run() (err error) {
	// Go through all known file and fill their PcStats
	err = p.fillPcStats()
	if err != nil {
		return
	}

	// Build the relname -> relinfo map
	p.relToRelinfo = make(relation.RelToRelinfo, 0)
	for _, v := range p.fileToRelinfo {
		p.relToRelinfo[v.Relname] = v
	}

	p.OutputResults()

	return
}
