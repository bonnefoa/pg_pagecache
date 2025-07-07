package relation

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/lib/pq"
)

type TableToRelinfos map[TableInfo][]*RelInfo
type PartitionToTables map[PartInfo]TableToRelinfos

// GetPartitionToTables returns the mapping between a parent partition and its children
// Child includes toast table, toast table index and all indexes of the parent relation
func GetPartitionToTables(ctx context.Context, conn *pgx.Conn, tables []string, pageThreshold int) (partitionToTables PartitionToTables, err error) {
	rows, err := conn.Query(ctx, `SELECT COALESCE(parent_idx.relname, parent.relname, 'No Partition'), COALESCE(PPTI.relname, PT.relname, PI.relname, C.relname) as t, C.relname, C.relkind, COALESCE(NULLIF(C.relfilenode, 0), C.oid)
		FROM pg_class C
		LEFT JOIN pg_index ON pg_index.indexrelid = C.oid
		-- index to parent table
		LEFT JOIN pg_class PI ON pg_index.indrelid = PI.oid AND PI.relkind='r'
		-- toast to parent table
		LEFT JOIN pg_class PT ON C.oid = PT.reltoastrelid

    -- Parent partition
    LEFT JOIN pg_inherits inh ON inh.inhrelid = C.oid
    LEFT JOIN pg_class parent ON inh.inhparent = parent.oid

    -- Parent partition from indexes
    LEFT JOIN pg_inherits inh_idx ON inh_idx.inhrelid = PI.oid
    LEFT JOIN pg_class parent_idx ON inh_idx.inhparent = parent_idx.oid

		-- toast index to toast table
		LEFT JOIN pg_class PTI ON pg_index.indrelid = PTI.oid AND PTI.relkind='t'
		LEFT JOIN pg_class PPTI ON PPTI.reltoastrelid = PTI.oid
		WHERE ($1 OR COALESCE(PPTI.relname, PT.relname, PI.relname, C.relname)=ANY($2)) AND C.relpages > $3 AND C.relkind = ANY('{r,i,t,m,p,I}')
`, len(tables) == 0, pq.Array(tables), pageThreshold)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting list of relfilenode from pg_class: %v\n", err)
		return
	}

	partitionToTables = make(PartitionToTables, 0)
	for rows.Next() {
		partInfo := PartInfo{BaseInfo: BaseInfo{Kind: 'P'}}
		tableInfo := TableInfo{BaseInfo: BaseInfo{Kind: 'T'}}
		var relinfo RelInfo
		err = rows.Scan(&partInfo.Name, &tableInfo.Name, &relinfo.Name, &relinfo.Kind, &relinfo.Relfilenode)
		if err != nil {
			return nil, fmt.Errorf("Error getting table to relation from pg_class: %v", err)
		}
		tableToRelinfos, ok := partitionToTables[partInfo]
		if !ok {
			tableToRelinfos = make(TableToRelinfos, 0)
		}
		tableToRelinfos[tableInfo] = append(tableToRelinfos[tableInfo], &relinfo)
		partitionToTables[partInfo] = tableToRelinfos
	}
	return
}

func kindToString(kind rune) string {
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
	// Artificial kinds for our own types
	case 'S':
		return "Total"
	case 'P':
		return "Partition"
	case 'T':
		return "Table"
	}
	return "Unkown"
}
