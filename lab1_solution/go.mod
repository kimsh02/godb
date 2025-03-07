module main

go 1.21

toolchain go1.23.1

replace github.com/srmadden/godb => ./godb

require (
	github.com/chzyer/readline v1.5.1
	github.com/srmadden/godb v0.0.0-00010101000000-000000000000
)

require (
	github.com/tylertreat/BoomFilters v0.0.0-20210315201527-1a82519a3e43 // indirect
	github.com/xwb1989/sqlparser v0.0.0-20180606152119-120387863bf2 // indirect
	golang.org/x/sys v0.0.0-20220310020820-b874c991c1a5 // indirect
)
