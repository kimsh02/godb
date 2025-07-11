# GoDB

![Demo](assets/demo.gif)

## About

* GoDB is a fully functional relational database written purely in Go.

## Features

* Core modules support disk access, query processing (filter, join, aggregate,
  insertion/deletion, projection, order-by, and limit), transactions handling,
  and concurrency control.

* Integrates rigorous two-phase locking mechanism with page-level concurrency
  control (spinlocks, mutexes), deadlock detection for recovery, and *no
  steal/force* buffer management policy to handle disk/memory I/O.

* *No steal* requires that dirty pages are not written to the disk before a
  transaction commits, and *force* requires that all changes made by a
  transaction once it commits are immediatey written to the disk. This
  combination simplifies database recovery protocol for crashes but leads to
  higher memory usage and slower performance as a trade-off.

## Build

* To build the database, make sure the Go toolchain is installed on your machine
  and then run `go build` in the root directory.

* The executable will be compiled as `main` which can be invoked with `./main`.

## Usage

Typing `\h` will give a list of commands you can input; for example, `\d` lists
the tables and their schemas.  Tables are, by default, loaded from the file
`catalog.txt`, but you can point to another catalog file.  Note that each table
in the catalog is stored in a file called `<tablename>.dat`, where tablename is
the name of the table. From this terminal, you can run `DROP`, `CREATE`,
`INSERT`, `BEGIN`, `COMMIT/ROLLBACK`, and `SELECT` statements.  You can also
load a CSV file into a table using the `\l` command.

The parser supports most of SQL with some limitations, including:

* No CTEs, window functions, recursive queries, or other SQL99 or later features
  (arbitrarily nested subqueries are fully supported)
* No OUTER joins (all joins are INNER)
* No USING clause for join predicates (you should write this to ON)
* No correlated subqueries
* No UPDATEs


When you first run the console, it will load a small test catalog containing two
identical tables of people and ages.  You can see the schemas of these tables
using the `\d` command.

## MBTA Example

As an example, you can load the famous MBTA dataset into GoDB format. You can
download it from
[here](https://www.dropbox.com/scl/fi/l27l17fg6mo3d4jjihmls/transitdb.zip?rlkey=890c1omvwevm6n4us10d7m11j). Note
that all columns are either strings or ints; floats have been cast to ints in
this database.

If you download and unzip this file in your top level lab, you can connect to
over the console using the `\c` command:

```
> \c transitdb/transitdb.catalog
Loaded transitdb/transitdb.catalog
gated_station_entries (service_date string, time string, station_id string, line_id string, gated_entries int)
lines (line_id string, line_name string)
routes (route_id int, line_id string, first_station_id string, last_station_id string, direction int, direction_desc string, route_name string)
stations (station_id string, station_name string)
rail_ridership (season string, line_id string, direction int, time_period_id string, station_id string, total_ons int, total_offs int, number_service_days int, average_ons int, average_offs int, average_flow int)
station_orders (route_id int, station_id string, stop_order int, distance_from_last_station_miles int)
time_periods (time_period_id string, day_type string, time_period string, period_start_time string, period_end_time string)
```

Once it is loaded, you should be able to run a query. For example, to find the
first and last station of each line, you can write:

```
> SELECT line_name,
>        direction_desc,
>        s1.station_name AS first_station,
>        s2.station_name AS last_station
> FROM routes
> JOIN lines ON lines.line_id = routes.line_id
> JOIN stations s1 ON first_station_id = s1.station_id
> JOIN stations s2 ON last_station_id = s2.station_id
> ORDER BY line_name ASC, direction_desc ASC, first_station ASC, last_station ASC;
          line_name          |        direction_desc       |        first_station        |         last_station        |
         "Blue Line"         |             East            |           Bowdoin           |          Wonderland         |
         "Blue Line"         |             West            |          Wonderland         |           Bowdoin           |
         "Green Line"        |             East            |       "Boston College"      |     "Government Center"     |
         "Green Line"        |             East            |      "Cleveland Circle"     |     "Government Center"     |
         "Green Line"        |             East            |        "Heath Street"       |           Lechmere          |
         "Green Line"        |             East            |          Riverside          |       "North Station"       |
         "Green Line"        |             West            |     "Government Center"     |       "Boston College"      |
         "Green Line"        |             West            |     "Government Center"     |      "Cleveland Circle"     |
         "Green Line"        |             West            |       "North Station"       |          Riverside          |
         "Green Line"        |             West            |           Lechmere          |        "Heath Street"       |
      "Mattapan Trolley"     |           Inbound           |           Mattapan          |           Ashmont           |
      "Mattapan Trolley"     |           Outbound          |           Ashmont           |           Mattapan          |
        "Orange Line"        |            North            |        "Forest Hills"       |         "Oak Grove"         |
        "Orange Line"        |            South            |         "Oak Grove"         |        "Forest Hills"       |
          "Red Line"         |            North            |           Ashmont           |           Alewife           |
          "Red Line"         |            North            |          Braintree          |           Alewife           |
          "Red Line"         |            South            |           Alewife           |           Ashmont           |
          "Red Line"         |            South            |           Alewife           |          Braintree          |
(18 results)
57.01075ms
```

You can also view the query plan generated for the query by appending the
"EXPLAIN" keyword to a query, e.g.:

```
> explain SELECT line_name,
>        direction_desc,
>        s1.station_name AS first_station,
>        s2.station_name AS last_station
> FROM routes
> JOIN lines ON lines.line_id = routes.line_id
> JOIN stations s1 ON first_station_id = s1.station_id
> JOIN stations s2 ON last_station_id = s2.station_id
> ORDER BY line_name ASC, direction_desc ASC, first_station ASC, last_station ASC;

Order By line_name,direction_desc,first_station,last_station,
    Project lines.line_name,routes.direction_desc,s1.station_name,s2.station_name, -> [line_name direction_desc first_station last_station]
        Join, routes.last_station_id == s2.station_id
            Join, routes.first_station_id == s1.station_id
                Join, lines.line_id == routes.line_id
                    Heap Scan transitdb/lines.dat
                    Heap Scan transitdb/routes.dat
                Heap Scan transitdb/stations.dat
            Heap Scan transitdb/stations.dat
```
