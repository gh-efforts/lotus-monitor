package fullnode

import (
	"sync"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/gh-efforts/lotus-monitor/metrics"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
)

func (n *FullNode) minerRecords() error {
	wg := sync.WaitGroup{}
	wg.Add(len(n.miners))

	for _, maddr := range n.miners {
		go func(maddr address.Address) {
			defer wg.Done()
			err := n.minerRecord(maddr)
			if err != nil {
				log.Errorw("miner record", "maddr", maddr, "err", err)
			}
			log.Infof("miner: %s minerRecord success", maddr)
		}(maddr)
	}
	wg.Wait()
	return nil
}

func (n *FullNode) minerRecord(maddr address.Address) error {
	ctx, _ := tag.New(n.ctx,
		tag.Upsert(metrics.MinerID, maddr.String()),
	)

	faults, err := n.api.StateMinerFaults(ctx, maddr, types.EmptyTSK)
	if err != nil {
		return err
	}
	f, err := faults.Count()
	if err != nil {
		return err
	}
	stats.Record(ctx, metrics.MinerFaults.M(int64(f)))

	recoveries, err := n.api.StateMinerRecoveries(ctx, maddr, types.EmptyTSK)
	if err != nil {
		return err
	}
	r, err := recoveries.Count()
	if err != nil {
		return err
	}
	stats.Record(ctx, metrics.MinerRecoveries.M(int64(r)))

	mi, err := n.api.StateMinerInfo(ctx, maddr, types.EmptyTSK)
	if err != nil {
		return err
	}

	actors := map[address.Address]string{}
	actors[mi.Owner] = "owner"
	actors[mi.Worker] = "worker"
	for _, c := range mi.ControlAddresses {
		actors[c] = "control"
	}

	for k, v := range actors {
		ctx, _ = tag.New(ctx,
			tag.Upsert(metrics.ActorAddress, k.String()),
			tag.Upsert(metrics.AddressType, v),
		)
		actor, err := n.api.StateGetActor(ctx, k, types.EmptyTSK)
		if err != nil {
			return err
		}

		stats.Record(ctx, metrics.Balance.M(types.BigDivFloat(actor.Balance, types.FromFil(1))))
	}

	return nil
}
