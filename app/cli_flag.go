package app

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

var (
	cliArgs CliArgs

	relationsFlag string
)

type CliArgs struct {
	PgData        string
	Database      string
	ConnectString string
	Relations     []string
	PageThreshold int
	Limit         int

	OutputOptions OutputOptions
}

func init() {
	flag.StringVar(&cliArgs.PgData, "pgData", "", "Location of pgdata, uses PGDATA env var if not defined")
	flag.StringVar(&cliArgs.ConnectString, "connect_str", "", "Connection string to PostgreSQL")
	flag.IntVar(&cliArgs.PageThreshold, "page_threshold", 10, "Exclude relations with less pages than the threshold")
	flag.StringVar(&relationsFlag, "relations", "", "Filter on a specific relations (separated with commas)")
}

func ParseCliArgs() (CliArgs, error) {
	flag.Parse()
	err := SetLogLevel()
	if err != nil {
		return cliArgs, err
	}
	cliArgs.OutputOptions, err = ParseOutputOptions()
	if err != nil {
		return cliArgs, err
	}

	if cliArgs.PgData == "" {
		// Fallback to PGDATA env var
		var found bool
		cliArgs.PgData, found = os.LookupEnv("PGDATA")
		if !found {
			return cliArgs, fmt.Errorf("pgdata is mandatory")
		}
	}

	if relationsFlag != "" {
		cliArgs.Relations = strings.Split(relationsFlag, ",")
	}

	return cliArgs, err
}
