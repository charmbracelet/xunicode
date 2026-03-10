module bench

go 1.25.5

replace charm.land/xunicode => ../../

require (
	charm.land/xunicode v0.0.0-00010101000000-000000000000
	github.com/SCKelemen/unicode v1.1.1
	github.com/clipperhouse/uax29/v2 v2.7.0
	github.com/go-text/typesetting v0.3.4
	github.com/rivo/uniseg v0.4.7
)

require golang.org/x/text v0.34.0 // indirect
