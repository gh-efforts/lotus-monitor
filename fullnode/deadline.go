package fullnode

import (
	"strconv"
	"sync"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/gh-efforts/lotus-monitor/metrics"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
)

func (n *FullNode) deadlineRecords() {
	stop := metrics.Timer(n.ctx, "fullnode/deadlineRecords")
	defer stop()

	wg := sync.WaitGroup{}
	wg.Add(len(n.miners))

	for _, maddr := range n.miners {
		go func(maddr address.Address) {
			defer wg.Done()
			if err := n.deadlineRecord(maddr); err != nil {
				log.Errorw("deadlineRecord failed", "miner", maddr, "err", err)
				metrics.RecordError(n.ctx, "fullnode/deadlineRecord")
			}
		}(maddr)
	}
	wg.Wait()
}

func (n *FullNode) deadlineRecord(maddr address.Address) error {
	ctx, _ := tag.New(n.ctx,
		tag.Upsert(metrics.MinerID, maddr.String()),
	)

	di, err := n.API.StateMinerProvingDeadline(ctx, maddr, types.EmptyTSK)
	if err != nil {
		return err
	}

	deadlines, err := n.API.StateMinerDeadlines(ctx, maddr, types.EmptyTSK)
	if err != nil {
		return err
	}

	dlIdx := di.Index
	//当前 deadline 已经提交的 partition 数量
	provenPartitions, err := deadlines[dlIdx].PostSubmissions.Count()
	if err != nil {
		return err
	}

	partitions, err := n.API.StateMinerPartitions(ctx, maddr, uint64(dlIdx), types.EmptyTSK)
	if err != nil {
		return err
	}

	//以 LiveSectors 不等于0的 partition 数量来判断应该要提交几个 partition
	liveSectorsPartitions := uint64(0)
	for _, partition := range partitions {
		live, err := partition.LiveSectors.Count()
		if err != nil {
			return err
		}

		if live > 0 {
			liveSectorsPartitions += 1
		}
	}

	currentCost := int64(0)
	if provenPartitions < liveSectorsPartitions {
		currentCost = int64(di.CurrentEpoch - di.Open)
	} else {
		currentCost = -1
	}

	ctx, _ = tag.New(ctx,
		tag.Upsert(metrics.DeadlineIndex, strconv.Itoa(int(di.Index))),
	)
	stats.Record(ctx, metrics.DeadlineCost.M(currentCost))

	return nil
}
