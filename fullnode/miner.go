package fullnode

import (
	"sync"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/gh-efforts/lotus-monitor/metrics"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
)

func (n *FullNode) minerRecords() {
	stop := metrics.Timer(n.ctx, "fullnode/minerRecords")
	defer stop()

	miners := n.dc.MinersList()

	wg := sync.WaitGroup{}
	wg.Add(len(miners))

	for _, maddr := range miners {
		go func(maddr address.Address) {
			defer wg.Done()
			if err := n.minerRecord(maddr); err != nil {
				log.Errorw("minerRecord failed", "miner", maddr, "err", err)
				metrics.RecordError(n.ctx, "fullnode/minerRecord")
			}
		}(maddr)
	}
	wg.Wait()
}

func (n *FullNode) minerRecord(maddr address.Address) error {
	ctx, _ := tag.New(n.ctx,
		tag.Upsert(metrics.MinerID, maddr.String()),
	)
	api := n.dc.LotusApi

	faults, err := api.StateMinerFaults(ctx, maddr, types.EmptyTSK)
	if err != nil {
		return err
	}
	f, err := faults.Count()
	if err != nil {
		return err
	}
	stats.Record(ctx, metrics.MinerFaults.M(int64(f)))

	recoveries, err := api.StateMinerRecoveries(ctx, maddr, types.EmptyTSK)
	if err != nil {
		return err
	}
	r, err := recoveries.Count()
	if err != nil {
		return err
	}
	stats.Record(ctx, metrics.MinerRecoveries.M(int64(r)))

	mi, err := api.StateMinerInfo(ctx, maddr, types.EmptyTSK)
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
		actor, err := api.StateGetActor(ctx, k, types.EmptyTSK)
		if err != nil {
			return err
		}

		stats.Record(ctx, metrics.Balance.M(types.BigDivFloat(actor.Balance, types.FromFil(1))))
	}

	return nil
}
