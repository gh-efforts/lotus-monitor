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

	resolves map[address.Address]address.Address
}

func NewMpool(ctx context.Context, dc *config.DynamicConfig) *Mpool {
	return &Mpool{
		ctx: ctx,
		dc:  dc,

		resolves: make(map[address.Address]address.Address),
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
	stop := metrics.Timer(m.ctx, "mpool/record")
	defer stop()

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

		owner, err := m.resolve(mi.Owner)
		if err != nil {
			return err
		}
		worker, err := m.resolve(mi.Worker)
		if err != nil {
			return err
		}

		actors[owner] = 0
		actors[worker] = 0

		for _, c := range mi.ControlAddresses {
			d, err := m.resolve(c)
			if err != nil {
				return err
			}

			actors[d] = 0
		}
	}

	msgs, err := m.dc.LotusApi.MpoolPending(m.ctx, types.EmptyTSK)
	if err != nil {
		return err
	}

	for _, v := range msgs {
		if _, has := actors[v.Message.From]; has {
			actors[v.Message.From] += 1
		}
	}

	for k, v := range actors {
		ctx, _ := tag.New(m.ctx,
			tag.Upsert(metrics.ActorAddress, k.String()),
		)
		stats.Record(ctx, metrics.MpoolMsgNumber.M(v))
	}

	return nil
}

func (m *Mpool) resolve(id address.Address) (address.Address, error) {
	var addr address.Address
	addr, has := m.resolves[id]
	if has {
		return addr, nil
	}

	addr, err := m.dc.LotusApi.StateAccountKey(m.ctx, id, types.EmptyTSK)
	if err != nil {
		return address.Address{}, err
	}

	m.resolves[id] = addr

	return addr, nil
}
