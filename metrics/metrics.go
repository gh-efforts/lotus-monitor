package metrics

import (
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

// Tags
var (
	Version, _ = tag.NewKey("version")
	Commit, _  = tag.NewKey("commit")
	MinerID, _ = tag.NewKey("miner_id")

	ActorAddress, _ = tag.NewKey("actor_address")
	AddressType, _  = tag.NewKey("address_type")
)

// Measures
var (
	Info    = stats.Int64("info", "Arbitrary counter to tag monitor info to", stats.UnitDimensionless)
	Balance = stats.Float64("balance", "actor balance", "FIL")
)

// Views
var (
	InfoView = &view.View{
		Name:        "info",
		Description: "Monitor information",
		Measure:     Info,
		Aggregation: view.LastValue(),
		TagKeys:     []tag.Key{Version, Commit},
	}
	BalanceView = &view.View{
		Name:        "balance",
		Description: "actor balance",
		Measure:     Balance,
		Aggregation: view.LastValue(),
		TagKeys:     []tag.Key{ActorAddress, AddressType},
	}
)

var Views = []*view.View{
	InfoView,
	BalanceView,
}
