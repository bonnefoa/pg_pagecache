package main

import (
	"context"
	"flag"
	"os"

	"log/slog"

	"github.com/bonnefoa/pg_pagecache/app"
	"github.com/jackc/pgx/v5"
)

func main() {
	ctx := context.Background()
	flag.Parse()

	cliArgs, err := app.ParseCliArgs()
	if err != nil {
		slog.Error("Error while parsing arguments", "error", err)
		flag.Usage()
		os.Exit(1)
	}

	// Get the db connection
	conn, err := pgx.Connect(ctx, cliArgs.ConnectString)
	if err != nil {
		slog.Error("Unable to connect to database", "error", err)
		os.Exit(1)
	}
	defer conn.Close(ctx)

	// Build PgPagecache struct
	pgPagecache, err := app.NewPgPagecache(ctx, conn, cliArgs)
	if err != nil {
		slog.Any("error", err)
		os.Exit(1)
	}

	// Run it
	err = pgPagecache.Run()
	if err != nil {
		slog.Any("error", err)
		os.Exit(1)
	}
}
