package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
)

var (
	pgdata        string
	database      string
	connectString string
)

type FileToRelation map[uint32]string

func init() {
	flag.StringVar(&pgdata, "pgData", "", "Location of pgdata, uses PGDATA env var if not defined")
	flag.StringVar(&connectString, "", "", "Connection string to PostgreSQL")
}

func getFileToRelation(ctx context.Context, conn *pgx.Conn) (fileToRelation FileToRelation, err error) {
	rows, err := conn.Query(ctx, "SELECT relname, relfilenode::int FROM pg_class WHERE relfilenode > 0")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting list of relfilenode from pg_class: %v\n", err)
		return
	}

	for rows.Next() {
		var relname string
		var relfilenode uint32
		err = rows.Scan(&relname, relfilenode)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting list of relfilenode from pg_class: %v\n", err)
			return
		}
		fileToRelation[relfilenode] = relname
	}

	return
}

func main() {
	ctx := context.Background()
	flag.Parse()

	//	if pgdata == "" {
	//		pgdata, found := os.LookupEnv("PGDATA")
	//		if !found {
	//			flag.Usage()
	//			os.Exit(1)
	//		}
	//	}

	conn, err := pgx.Connect(ctx, connectString)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(ctx)

	var database string
	err = conn.QueryRow(ctx, "select current_database()").Scan(&database)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting current database: %v\n", err)
		os.Exit(1)
	}

	fileToRelation, err := getFileToRelation(ctx, conn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting file to relation mapping: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Working on database %s", database)
	fmt.Printf("Found %d files", len(fileToRelation))
}
