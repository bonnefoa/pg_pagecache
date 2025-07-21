## pg_pagecache

`pg_pagecache` shows page cache statistics of PostgreSQL's relation files.

It will:
- Connect to the database to fetch relation informations from `pg_class` and `pg_index`
- Iterate over relation files with `mincore` to get how many pages are cached
- If readable, it reads `/proc/kpageflages` to get page flags and display them (similar to `page-types`)

## Installation

Get the latest binary for your architecture:
```
tag=$(curl -s 'https://api.github.com/repos/bonnefoa/pg_pagecache/releases/latest' | sed -n 's/ *"name": "\(v[0-9]*.[0-9]*.[0-9]*\).*/\1/p')
file="pg_pagecache-${tag}-$(uname)-$(uname -m)"
curl -L "https://github.com/bonnefoa/pg_pagecache/releases/download/${tag}/${file}" -o pg_pagecache
chmod a+x pg_pagecache
```

## Usage

Calling `pg_pagecache` without arguments will attempt to connect to the database using the standard `PGHOST`, `PGUSER` and `PGDATABASE` env vars.
`PGDATA` will also be used to locate the relation files to scan.

```
./pg_pagecache

Partition     Table            Relation              Relfilenode   Kind          PageCached    PageCount     %Cached       %Total
No partition  pgbench_accounts pgbench_accounts_pkey 33628         Index         552 Pgs       552 Pgs       100.00        0.03
No partition  pgbench_branches pgbench_branches_pkey 33624         Index         4 Pgs         4 Pgs         100.00        0.00
No partition  pgbench_tellers  pgbench_tellers_pkey  33626         Index         4 Pgs         4 Pgs         100.00        0.00
No partition  pgbench_branches pgbench_branches      33621         Relation      2 Pgs         2 Pgs         100.00        0.00
No partition  pgbench_tellers  pgbench_tellers       33623         Relation      2 Pgs         2 Pgs         100.00        0.00
                               Total                               Total         564 Pgs       564 Pgs       100.00        0.03
```

### Page Flags

If `/proc/kpageflages` is readable, page flags details will be displayed

```
sudo ./pg_pagecache -connect_str "user=postgres database=postgres host=localhost" -pg_data ~/pg_data
Partition     Table            Relation                  Relfilenode   Kind          PageCached    PageCount     %Cached       %Total
No partition  pgbench_accounts pgbench_accounts_pkey     33628         Index         552 Pgs       552 Pgs       100.00        0.03
No partition  pg_type          pg_type_oid_index         2703          Index         10 Pgs        18 Pgs        55.56         0.00
No partition  pg_type          pg_type_typname_nsp_index 2704          Index         10 Pgs        26 Pgs        38.46         0.00
No partition  pgbench_branches pgbench_branches_pkey     33624         Index         4 Pgs         4 Pgs         100.00        0.00
No partition  pgbench_tellers  pgbench_tellers_pkey      33626         Index         4 Pgs         4 Pgs         100.00        0.00
No partition  pg_statistic     pg_toast_2619             2840          TOAST         2 Pgs         16 Pgs        12.50         0.00
No partition  pgbench_branches pgbench_branches          33621         Relation      2 Pgs         2 Pgs         100.00        0.00
No partition  pgbench_tellers  pgbench_tellers           33623         Relation      2 Pgs         2 Pgs         100.00        0.00
                               Total                                   Total         586 Pgs       624 Pgs       93.91         0.03

Page Flags
Relation                  Page Count    Flags              Symbolic Flags                                                   Long Symbolic Flags
pgbench_accounts_pkey     550           0x0000000000000028 ___U_l__________________________________________________________ uptodate,lru
pgbench_accounts_pkey     2             0x000000000000002c __RU_l__________________________________________________________ referenced,uptodate,lru
pg_type_oid_index         10            0x000000000000002c __RU_l__________________________________________________________ referenced,uptodate,lru
pg_type_typname_nsp_index 10            0x000000000000002c __RU_l__________________________________________________________ referenced,uptodate,lru
pgbench_branches_pkey     4             0x0000000000000028 ___U_l__________________________________________________________ uptodate,lru
pgbench_tellers_pkey      4             0x0000000000000028 ___U_l__________________________________________________________ uptodate,lru
pg_toast_2619             2             0x000000000000002c __RU_l__________________________________________________________ referenced,uptodate,lru
pgbench_branches          2             0x0000000000000028 ___U_l__________________________________________________________ uptodate,lru
pgbench_tellers           2             0x0000000000000028 ___U_l__________________________________________________________ uptodate,lru
```

## %Total Memory

`%Total` column reports the total usage of the cached memory. Cache memory is extracted from `/proc/meminfo`.
`PostgreSQL` own `shared_buffers` is removed from this total as it is reported in the cache memory and can't be used by the page cache. 
This way, `%Total` shows the relation's memory usage of the page cache memory.
