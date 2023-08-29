package metrics

import (
	"context"
	"time"

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

// Distribution
var defaultMillisecondsDistribution = view.Distribution(0.01, 0.05, 0.1, 0.3, 0.6, 0.8, 1, 2, 3, 4, 5, 6, 8, 10, 13, 16, 20, 25, 30, 40, 50, 65, 80, 100, 130, 160, 200, 250, 300, 400, 500, 650, 800, 1000, 2000, 3000, 4000, 5000, 7500, 10000, 20000, 50000, 100_000, 250_000, 500_000, 1000_000)
var blockTookDurationDistribution = view.Distribution(0, 1, 2, 3, 5, 7, 10, 30, 60, 120) //seconds

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

	BlockCID, _    = tag.NewKey("block_cid")
	BlockHeight, _ = tag.NewKey("block_height")
	ErrorType, _   = tag.NewKey("error_type")
	RecordType, _  = tag.NewKey("record_type")
)

// Measures
var (
	Info = stats.Int64("info", "Arbitrary counter to tag monitor info to", stats.UnitDimensionless)

	Balance = stats.Float64("balance", "actor balance (FIL)", "FIL")

	MinerFaults          = stats.Int64("miner/faults", "faulty sectors", stats.UnitDimensionless)
	MinerRecoveries      = stats.Int64("miner/recoveries", "recovery sectors", stats.UnitDimensionless)
	MinerRawBytePower    = stats.Int64("miner/raw_byte_power", "miner raw byte power", stats.UnitBytes)
	MinerQualityAdjPower = stats.Int64("miner/quality_adj_power", "miner quality adj power", stats.UnitBytes)

	DeadlineCost = stats.Int64("deadline/cost", "proven cost of current deadline (epoch)", "epoch")

	JobsTimeout = stats.Int64("miner/jobs", "the number of jobs that timed out", stats.UnitDimensionless)
	JobsNumber  = stats.Int64("miner/jobs_number", "total number of sealing jobs", stats.UnitDimensionless)

	LuckyValue = stats.Float64("lucky_value", "lucky value of miner", stats.UnitDimensionless)

	BlockOnchain      = stats.Int64("block/on_chain", "counter for block on chain", stats.UnitDimensionless)
	BlockOrphanCount  = stats.Int64("block/orphan_count", "counter for orphan block", stats.UnitDimensionless)
	BlockOrphan       = stats.Int64("block/orphan", "mined orphan block", stats.UnitDimensionless)
	BlockTookDuration = stats.Float64("block/took", "duration of mined a block", stats.UnitSeconds)

	SelfError          = stats.Int64("self/error", "couter for monitor error", stats.UnitDimensionless)
	SelfRecordDuration = stats.Float64("self/record", "duration of every record", stats.UnitMilliseconds)
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
	MinerRawBytePowerView = &view.View{
		Aggregation: view.LastValue(),
		Measure:     MinerRawBytePower,
		TagKeys:     []tag.Key{MinerID},
	}
	MinerQualityAdjPowerView = &view.View{
		Aggregation: view.LastValue(),
		Measure:     MinerQualityAdjPower,
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
	JobsNumberView = &view.View{
		Aggregation: view.LastValue(),
		Measure:     JobsNumber,
		TagKeys:     []tag.Key{MinerID},
	}
	LuckyValueView = &view.View{
		Aggregation: view.LastValue(),
		Measure:     LuckyValue,
		TagKeys:     []tag.Key{MinerID, LuckyValueDay},
	}
	BlockOnchainView = &view.View{
		Measure:     BlockOnchain,
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{MinerID},
	}
	BlockOrphanCountView = &view.View{
		Measure:     BlockOrphanCount,
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{MinerID},
	}
	BlockOrphanView = &view.View{
		Measure:     BlockOrphan,
		Aggregation: view.LastValue(),
		TagKeys:     []tag.Key{MinerID, BlockCID, BlockHeight},
	}
	BlockTookDurationView = &view.View{
		Measure:     BlockTookDuration,
		Aggregation: blockTookDurationDistribution,
		TagKeys:     []tag.Key{MinerID},
	}
	SelfErrorView = &view.View{
		Measure:     SelfError,
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{ErrorType},
	}
	SelfRecordDurationView = &view.View{
		Measure:     SelfRecordDuration,
		Aggregation: defaultMillisecondsDistribution,
		TagKeys:     []tag.Key{RecordType},
	}
)

var Views = []*view.View{
	InfoView,
	BalanceView,
	MinerFaultsView,
	MinerRecoveriesView,
	MinerRawBytePowerView,
	MinerQualityAdjPowerView,
	DeadlineCostView,
	JobsTimeoutView,
	JobsNumberView,
	LuckyValueView,
	BlockOnchainView,
	BlockOrphanCountView,
	BlockOrphanView,
	BlockTookDurationView,
	SelfErrorView,
	SelfRecordDurationView,
}

// SinceInMilliseconds returns the duration of time since the provide time as a float64.
func SinceInMilliseconds(startTime time.Time) float64 {
	return float64(time.Since(startTime).Nanoseconds()) / 1e6
}

// Timer is a function stopwatch, calling it starts the timer,
// calling the returned function will record the duration.
func Timer(ctx context.Context, recordType string) func() time.Duration {
	ctx, _ = tag.New(ctx,
		tag.Upsert(RecordType, recordType),
	)
	start := time.Now()
	return func() time.Duration {
		stats.Record(ctx, SelfRecordDuration.M(SinceInMilliseconds(start)))
		return time.Since(start)
	}
}

func RecordError(ctx context.Context, errType string) {
	ctx, _ = tag.New(ctx,
		tag.Upsert(ErrorType, errType),
	)

	stats.Record(ctx, SelfError.M(1))
}
