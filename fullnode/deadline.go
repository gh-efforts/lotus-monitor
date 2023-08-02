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

func (n *FullNode) deadlineRecords() error {
	wg := sync.WaitGroup{}
	wg.Add(len(n.miners))

	for _, maddr := range n.miners {
		go func(maddr address.Address) {
			defer wg.Done()
			err := n.deadlineRecord(maddr)
			if err != nil {
				log.Errorw("deadline record", "maddr", maddr, "err", err)
			}
			log.Infof("miner: %s deadlineRecord success", maddr)
		}(maddr)
	}
	wg.Wait()
	return nil
}

func (n *FullNode) deadlineRecord(maddr address.Address) error {
	ctx, _ := tag.New(n.ctx,
		tag.Upsert(metrics.MinerID, maddr.String()),
	)

	di, err := n.api.StateMinerProvingDeadline(ctx, maddr, types.EmptyTSK)
	if err != nil {
		return err
	}

	deadlines, err := n.api.StateMinerDeadlines(ctx, maddr, types.EmptyTSK)
	if err != nil {
		return err
	}

	dlIdx := di.Index
	provenPartitions, err := deadlines[dlIdx].PostSubmissions.Count()
	if err != nil {
		return err
	}

	partitions, err := n.api.StateMinerPartitions(ctx, maddr, uint64(dlIdx), types.EmptyTSK)
	if err != nil {
		return err
	}

	haveActiveSectorPartitions := uint64(0)
	for _, partition := range partitions {
		active, err := partition.ActiveSectors.Count()
		if err != nil {
			//TODO: return or continue ?
			return err
		}
		if active > 0 {
			haveActiveSectorPartitions += 1
		}
	}

	currentCost := int64(0)
	openEpoch := int64(di.PeriodStart) + (int64(dlIdx) * 60)
	if haveActiveSectorPartitions > provenPartitions {
		currentCost = int64(di.CurrentEpoch) - openEpoch
	} else {
		if uint64(len(partitions)) == provenPartitions {
			currentCost = -1
		} else if uint64(len(partitions)) > provenPartitions {
			currentCost = -2
		} else {
			currentCost = -3
		}
	}

	ctx, _ = tag.New(ctx,
		tag.Upsert(metrics.DeadlineIndex, strconv.Itoa(int(di.Index))),
	)
	stats.Record(ctx, metrics.DeadlineCost.M(currentCost))

	return nil
}
