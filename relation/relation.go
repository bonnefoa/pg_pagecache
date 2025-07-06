package relation

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/lib/pq"
)

type TableToRelinfos map[TableInfo][]*RelInfo

// GetTableToRelinfo returns the mapping between a table parent and its child
// Child includes toast table, toast table index and all indexes of the parent relation
func GetTableToRelinfo(ctx context.Context, conn *pgx.Conn, tables []string, pageThreshold int) (tableToRelinfos TableToRelinfos, err error) {
	rows, err := conn.Query(ctx, `SELECT COALESCE(PPTI.relname, PT.relname, PI.relname, C.relname) as t, C.relname, C.relkind, COALESCE(NULLIF(C.relfilenode, 0), C.oid)
		FROM pg_class C
		LEFT JOIN pg_index ON pg_index.indexrelid = C.oid
		-- index to parent table
		LEFT JOIN pg_class PI ON pg_index.indrelid = PI.oid AND PI.relkind='r'
		-- toast to parent table
		LEFT JOIN pg_class PT ON C.oid = PT.reltoastrelid

		-- toast index to toast table
		LEFT JOIN pg_class PTI ON pg_index.indrelid = PTI.oid AND PTI.relkind='t'
		LEFT JOIN pg_class PPTI ON PPTI.reltoastrelid = PTI.oid
		WHERE ($1 OR COALESCE(PPTI.relname, PT.relname, PI.relname, C.relname)=ANY($2)) AND C.relpages > $3 AND C.relkind = ANY('{r,i,t,m,p,I}')
`, len(tables) == 0, pq.Array(tables), pageThreshold)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting list of relfilenode from pg_class: %v\n", err)
		return
	}

	tableToRelinfos = make(TableToRelinfos, 0)
	for rows.Next() {
		var tableInfo TableInfo
		var relinfo RelInfo
		err = rows.Scan(&tableInfo.Name, &relinfo.Relname, &relinfo.Relkind, &relinfo.Relfilenode)
		if err != nil {
			return nil, fmt.Errorf("Error getting table to relation from pg_class: %v", err)
		}
		tableToRelinfos[tableInfo] = append(tableToRelinfos[tableInfo], &relinfo)
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
		return ""
	case '-':
		return "-"
	}
	return "Unkown"
}
