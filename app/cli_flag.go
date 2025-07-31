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

// CliArgs stores cli flag values
type CliArgs struct {
	PgData              string
	Database            string
	ConnectString       string
	Relations           []string
	PageThreshold       int
	CachedPageThreshold int
	Cpuprofile          string
	RawFlags            bool
	ScanWal             bool

	FormatFlags
}

func init() {
	flag.StringVar(&cliArgs.PgData, "pg_data", "", "Location of pgdata, uses PGDATA env var if not defined")
	flag.StringVar(&cliArgs.ConnectString, "connect_str", "", "Connection string to PostgreSQL")
	flag.IntVar(&cliArgs.PageThreshold, "page_threshold", 0, "Exclude relations pages under the threshold. -1 to display everything")
	flag.IntVar(&cliArgs.CachedPageThreshold, "cached_page_threshold", 0, "Exclude relations with cached pages under the threshold. -1 to display everything")
	flag.StringVar(&cliArgs.Cpuprofile, "cpuprofile", "", "write cpu profile to `file`")
	flag.StringVar(&relationsFlag, "relations", "", "Filter on a specific relations (separated with commas)")
	flag.BoolVar(&cliArgs.RawFlags, "raw_flags", false, "Raw flag mode")
	flag.BoolVar(&cliArgs.ScanWal, "scan_wal", true, "Scan pagecache usage of WAL files")
}

// ParseCliArgs returns a CliArgs with parsed values
func ParseCliArgs() (CliArgs, error) {
	flag.Parse()
	err := SetLogLevel()
	if err != nil {
		return cliArgs, err
	}
	cliArgs.FormatFlags, err = ParseFormatOptions()
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
