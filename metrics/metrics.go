package metrics

import (
	"context"
	"time"

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

	DeadlineIndex, _ = tag.NewKey("deadline_index")

	TaskType, _ = tag.NewKey("task_type")

	LuckyValueDay, _ = tag.NewKey("lucky_value_day") //1day, 7day, 30day
)

// Measures
var (
	Info = stats.Int64("info", "Arbitrary counter to tag monitor info to", stats.UnitDimensionless)

	Balance = stats.Float64("balance", "actor balance", "FIL")

	MinerFaults     = stats.Int64("miner/faults", "faulty sectors", stats.UnitDimensionless)
	MinerRecoveries = stats.Int64("miner/recoveries", "recovery sectors", stats.UnitDimensionless)

	DeadlineCost = stats.Int64("deadline/cost", "proven cost of current deadline", "epoch")

	JobsTimeout = stats.Int64("miner/jobs", "the number of jobs that timed out", stats.UnitDimensionless)

	LuckyValue = stats.Float64("lucky_value", "lucky value of miner", stats.UnitDimensionless)
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
		Measure:     Balance,
		Aggregation: view.LastValue(),
		TagKeys:     []tag.Key{MinerID, ActorAddress, AddressType},
	}
	MinerFaultsView = &view.View{
		Aggregation: view.LastValue(),
		Measure:     MinerFaults,
		TagKeys:     []tag.Key{MinerID},
	}
	MinerRecoveriesView = &view.View{
		Aggregation: view.LastValue(),
		Measure:     MinerRecoveries,
		TagKeys:     []tag.Key{MinerID},
	}
	DeadlineCostView = &view.View{
		Aggregation: view.LastValue(),
		Measure:     DeadlineCost,
		TagKeys:     []tag.Key{MinerID, DeadlineIndex},
	}
	JobsTimeoutView = &view.View{
		Aggregation: view.LastValue(),
		Measure:     JobsTimeout,
		TagKeys:     []tag.Key{MinerID, TaskType},
	}
	LuckyValueView = &view.View{
		Aggregation: view.LastValue(),
		Measure:     LuckyValue,
		TagKeys:     []tag.Key{MinerID, LuckyValueDay},
	}
)

var Views = []*view.View{
	InfoView,
	BalanceView,
	MinerFaultsView,
	MinerRecoveriesView,
	DeadlineCostView,
	JobsTimeoutView,
	LuckyValueView,
}

// SinceInMilliseconds returns the duration of time since the provide time as a float64.
func SinceInMilliseconds(startTime time.Time) float64 {
	return float64(time.Since(startTime).Nanoseconds()) / 1e6
}

// Timer is a function stopwatch, calling it starts the timer,
// calling the returned function will record the duration.
func Timer(ctx context.Context, m *stats.Float64Measure) func() time.Duration {
	start := time.Now()
	return func() time.Duration {
		stats.Record(ctx, m.M(SinceInMilliseconds(start)))
		return time.Since(start)
	}
}
