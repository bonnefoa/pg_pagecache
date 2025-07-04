package relation

import (
	"context"
	"fmt"
	"os"

	"github.com/bonnefoa/pg_pagecache/pcstats"
	"github.com/jackc/pgx/v5"
	"github.com/lib/pq"
)

type TableToRelations map[string][]string
type TableToPartitions map[string][]string

// GetTableToRelations returns the mapping between a table parent and its child
// Child includes toast table, toast table index and all indexes of the parent relation
func getTableToRelations(ctx context.Context, conn *pgx.Conn, pageThreshold int) (tableToRelations TableToRelations, err error) {
	rows, err := conn.Query(ctx, `SELECT COALESCE(PPTI.relname, PT.relname, PI.relname, C.relname) as t, array_agg(C.relname)
		FROM pg_class C
		LEFT JOIN pg_index ON pg_index.indexrelid = C.oid
		-- index to parent table
		LEFT JOIN pg_class PI ON pg_index.indrelid = PI.oid AND PI.relkind='r'
		-- toast to parent table
		LEFT JOIN pg_class PT ON C.oid = PT.reltoastrelid

		-- toast index to toast table
		LEFT JOIN pg_class PTI ON pg_index.indrelid = PTI.oid AND PTI.relkind='t'
		LEFT JOIN pg_class PPTI ON PPTI.reltoastrelid = PTI.oid
		WHERE C.relpages > $1
		GROUP BY t
`, pageThreshold)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting list of relfilenode from pg_class: %v\n", err)
		return
	}

	tableToRelations = make(TableToRelations, 0)
	for rows.Next() {
		var table string
		var relations []string
		err = rows.Scan(&table, &relations)
		if err != nil {
			return nil, fmt.Errorf("Error getting table to relation from pg_class: %v", err)
		}
		tableToRelations[table] = relations
	}
	return
}

func getParentToPartitions(ctx context.Context, conn *pgx.Conn) (tableToPartitions TableToPartitions, err error) {
	rows, err := conn.Query(ctx, `SELECT C.relname, array_agg(child.relname)
		FROM pg_class C
		JOIN pg_inherits inh ON inh.inhparent = C.oid
		JOIN pg_class child ON inh.inhrelid = child.oid
		WHERE C.relkind = 'p'
		GROUP BY C.relname
		`)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting list of relfilenode from pg_class: %v\n", err)
		return
	}

	tableToPartitions = make(TableToPartitions, 0)
	for rows.Next() {
		var table string
		var partitions []string
		err = rows.Scan(&table, &partitions)
		if err != nil {
			return nil, fmt.Errorf("Error getting table to partitions from pg_class: %v", err)
		}
		tableToPartitions[table] = partitions
	}
	return
}

// GetFileToRelinfo returns the mapping between relfilenode and the relation
func GetFileToRelinfo(ctx context.Context, conn *pgx.Conn, relations []string, pageThreshold int) (fileToRelinfo FileToRelinfo, err error) {
	tableToRelations, err := getTableToRelations(ctx, conn, pageThreshold)
	if err != nil {
		return
	}

	var rows pgx.Rows
	if len(relations) > 0 {
		rows, err = conn.Query(ctx, `SELECT C.relname, C.relkind, COALESCE(NULLIF(C.relfilenode, 0), C.oid)
FROM pg_class C
WHERE relname=ANY($1) AND relpages > $2 AND relkind = ANY('{r,i,t,m,p,I}')`, pq.Array(relations), pageThreshold)
	} else {
		rows, err = conn.Query(ctx, `SELECT C.relname, C.relkind, COALESCE(NULLIF(C.relfilenode, 0), C.oid)
FROM pg_class C
WHERE relpages > $1 AND relkind = ANY('{r,i,t,m,p,I}')`, pageThreshold)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting list of relfilenode from pg_class: %v\n", err)
		return
	}

	fileToRelinfo = make(FileToRelinfo, 0)
	for rows.Next() {
		var relname string
		var relkind rune
		var relfilenode uint32
		err = rows.Scan(&relname, &relkind, &relfilenode)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting list of relfilenode from pg_class: %v\n", err)
			return
		}

		children := tableToRelations[relname]
		fileToRelinfo[relfilenode] = RelInfo{relfilenode, relname, relkind, pcstats.PcStats{}, children}
	}

	return
}

func KindToString(kind rune) string {
	switch kind {
	case 'r':
		return "Relation"
	case 'i':
		return "Index"
	case 'm':
		return "Materialised View"
	case 't':
		return "TOAST"
	case 'p':
		return "Partitioned Tabled"
	case 'I':
		return "Partitioned Index"
	// Artificial kind for our total line
	case 'T':
		return "Total"
	case '-':
		return "-"
	}
	return "Unkown"
}
