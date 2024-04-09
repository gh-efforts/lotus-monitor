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
	log.Debug(miners)

	wg := sync.WaitGroup{}
	wg.Add(len(miners))

	for _, maddr := range miners {
		go func(maddr address.Address) {
			defer wg.Done()
			if err := n.minerRecord(maddr); err != nil {
				log.Errorw("minerRecord failed", "miner", maddr, "err", err)
				metrics.RecordError(n.ctx, "fullnode/minerRecord")
			} else {
				log.Debugw("minerRecord success", "miner", maddr)
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

	ms, err := api.StateMinerSectorCount(ctx, maddr, types.EmptyTSK)
	if err != nil {
		return err
	}
	stats.Record(ctx, metrics.MinerFaults.M(int64(ms.Faulty)))
	stats.Record(ctx, metrics.MinerActives.M(int64(ms.Active)))
	stats.Record(ctx, metrics.MinerLives.M(int64(ms.Live)))

	mp, err := api.StateMinerPower(ctx, maddr, types.EmptyTSK)
	if err != nil {
		return err
	}
	stats.Record(ctx, metrics.MinerRawBytePower.M(mp.MinerPower.RawBytePower.Int64()))
	stats.Record(ctx, metrics.MinerQualityAdjPower.M(mp.MinerPower.QualityAdjPower.Int64()))

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

	balance, err := api.StateMinerAvailableBalance(ctx, maddr, types.EmptyTSK)
	if err != nil {
		return err
	}
	stats.Record(ctx, metrics.MinerAvailableBalance.M(types.BigDivFloat(balance, types.FromFil(1))))

	return nil
}
