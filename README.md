## pg_pagecache

pg_pagecache shows the pagecache usage of PostgreSQL's relation files.

It will:
- Connect to the database to fetch relation informations from `pg_class` and `pg_index`
- Iterate over `$PGDATA/base/$DBID` files with mincore to get pagecache usage

## Usage

Calling `pg_pagecache` without arguments will attempt to connect to the database using the standard `PGHOST`, `PGUSER` and `PGDATABASE` env vars. `PGDATA` will also be used to locate the relation files to scan.

```
pg_pagecache

Relation                        Kind          PageCached    PageCount     %Cached       %Total
pgbench_accounts_34             Relation      1124          12858         8.74          0.05
pgbench_accounts_35             Relation      1043          12858         8.11          0.05
pgbench_accounts_36             Relation      115           12858         0.89          0.01
pg_class                        Relation      104           1258          8.27          0.00
pg_attribute                    Relation      19            288           6.60          0.00
pgbench_accounts_32             Relation      12            12858         0.09          0.00
pg_amproc_fam_proc_index        Index         6             10            60.00         0.00
pg_index_indexrelid_index       Index         6             8             75.00         0.00
```

### Connect string 

It's possible to override the connection string and pgdata location with `connect_str` and `pg_data` flags:

```
pg_pagecache -connect_str "database=postgres" -pg_data ~/var/lib/postgresql/pg_data
```

### Limit Scan

On large databases, calling mincore can take time while not being very useful as you're likely mostly interested in top relations. You can use the `page_threshold` flag to only scan relations that have more pages (as reported by `pg_class`) than the provided threshold.

```
pg_pagecache -page_threshold 1000

Relation            Kind          PageCached    PageCount     %Cached       %Total
pgbench_accounts_34 Relation      1124          12858         8.74          0.05
pgbench_accounts_35 Relation      1043          12858         8.11          0.05
pgbench_accounts_36 Relation      115           12858         0.89          0.01
pgbench_accounts_32 Relation      12            12858         0.09          0.00
```
