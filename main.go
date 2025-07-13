package main

import (
	"context"
	"flag"
	"os"
	"runtime/pprof"

	"log/slog"

	"github.com/bonnefoa/pg_pagecache/app"
	"github.com/jackc/pgx/v5"
)

type QueryTracer struct{}

func (q QueryTracer) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	slog.Debug("Running query", "SQL", data.SQL)
	return ctx
}

func (q QueryTracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
}

func main() {
	ctx := context.Background()
	flag.Parse()

	cliArgs, err := app.ParseCliArgs()
	if err != nil {
		slog.Error("Error while parsing arguments", "error", err)
		flag.Usage()
		os.Exit(1)
	}

	if cliArgs.Cpuprofile != "" {
		f, err := os.Create(cliArgs.Cpuprofile)
		if err != nil {
			slog.Error("could not create CPU profile", "error", err)
			os.Exit(1)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			slog.Error("could not start CPU profile", "error", err)
			os.Exit(1)
		}
		defer pprof.StopCPUProfile()
	}

	// Get the db connection
	config, err := pgx.ParseConfig(cliArgs.ConnectString)
	if err != nil {
		slog.Error("Error parsing connection string", "error", err)
		os.Exit(1)
	}
	config.Tracer = QueryTracer{}
	conn, err := pgx.ConnectConfig(ctx, config)
	if err != nil {
		slog.Error("Unable to connect to database", "error", err)
		os.Exit(1)
	}
	defer conn.Close(ctx)

	// Build PgPagecache struct
	pgPagecache, err := app.NewPgPagecache(conn, cliArgs)
	if err != nil {
		slog.Error("New PgPagecache error", "error", err)
		os.Exit(1)
	}

	// Run it
	err = pgPagecache.Run(ctx)
	if err != nil {
		slog.Error("Run error", "error", err)
		os.Exit(1)
	}
}
