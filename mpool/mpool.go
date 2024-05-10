package mpool

import (
	"context"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/gh-efforts/lotus-monitor/config"
	"github.com/gh-efforts/lotus-monitor/metrics"
	logging "github.com/ipfs/go-log/v2"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
)

var log = logging.Logger("monitor/mpool")

type Mpool struct {
	ctx context.Context
	dc  *config.DynamicConfig
}

func NewMpool(ctx context.Context, dc *config.DynamicConfig) *Mpool {
	return &Mpool{
		ctx: ctx,
		dc:  dc,
	}
}

func (m *Mpool) Run() {
	go func() {
		m.record()
		t := time.NewTicker(time.Duration(m.dc.RecordInterval.Mpool))
		for {
			select {
			case <-t.C:
				m.record()
			case <-m.ctx.Done():
				return
			}
		}
	}()
}

func (m *Mpool) record() {
	err := m._record()
	if err != nil {
		log.Errorw("mpool record failed", "err", err)
		metrics.RecordError(m.ctx, "mpool/record")
	}
}

func (m *Mpool) _record() error {
	actors := map[address.Address]int64{}

	miners := m.dc.MinersList()
	for _, maddr := range miners {
		mi, err := m.dc.LotusApi.StateMinerInfo(m.ctx, maddr, types.EmptyTSK)
		if err != nil {
			return err
		}

		actors[mi.Owner] = 0
		actors[mi.Worker] = 0
		for _, c := range mi.ControlAddresses {
			actors[c] = 0
		}
	}

	msgs, err := m.dc.LotusApi.MpoolPending(m.ctx, types.EmptyTSK)
	if err != nil {
		return err
	}

	for _, v := range msgs {
		actors[v.Message.From] += 1
	}

	for k, v := range actors {
		ctx, _ := tag.New(m.ctx,
			tag.Upsert(metrics.ActorAddress, k.String()),
		)
		stats.Record(ctx, metrics.MpoolMsgNumber.M(v))
	}

	return nil
}
