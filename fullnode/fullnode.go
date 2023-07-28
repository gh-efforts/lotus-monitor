package fullnode

import (
	"context"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/api/v0api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/gh-efforts/lotus-monitor/metrics"
	logging "github.com/ipfs/go-log/v2"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
)

var log = logging.Logger("fullnode")

type FullNode struct {
	api    v0api.FullNode
	maddr  address.Address
	actors map[address.Address]string
}

func NewFullNode(api v0api.FullNode, maddr address.Address) *FullNode {
	return &FullNode{
		api:    api,
		maddr:  maddr,
		actors: make(map[address.Address]string),
	}
}

func (n *FullNode) Run(ctx context.Context) {
	go func() {
		t := time.NewTicker(time.Second * 30)
		for {
			select {
			case <-t.C:
				n.balanceRecord(ctx)
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (n *FullNode) Init(ctx context.Context) error {
	mi, err := n.api.StateMinerInfo(ctx, n.maddr, types.EmptyTSK)
	if err != nil {
		return err
	}

	n.actors[mi.Owner] = "owner"
	n.actors[mi.Worker] = "worker"
	for _, c := range mi.ControlAddresses {
		n.actors[c] = "control"
	}

	return nil
}

func (n *FullNode) balanceRecord(ctx context.Context) {
	for k, v := range n.actors {
		ctx, _ = tag.New(ctx,
			tag.Upsert(metrics.ActorAddress, k.String()),
			tag.Upsert(metrics.AddressType, v),
		)

		actor, err := n.api.StateGetActor(ctx, k, types.EmptyTSK)
		if err != nil {
			log.Error(err)
			continue
		}

		stats.Record(ctx, metrics.Balance.M(types.BigDivFloat(actor.Balance, types.FromFil(1))))
	}
}
