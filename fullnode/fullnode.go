package fullnode

import (
	"context"
	"net/http"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/lotus/api/client"
	"github.com/filecoin-project/lotus/api/v0api"
	"github.com/gh-efforts/lotus-monitor/config"
	logging "github.com/ipfs/go-log/v2"
)

var log = logging.Logger("fullnode")

type FullNode struct {
	ctx context.Context

	api    v0api.FullNode
	closer jsonrpc.ClientCloser

	miners []address.Address
}

func NewFullNode(ctx context.Context, conf *config.Config) (*FullNode, error) {
	var miners []address.Address
	for m, _ := range conf.Miners {
		a, err := address.NewFromString(m)
		if err != nil {
			return nil, err
		}
		miners = append(miners, a)
	}

	addr := "ws://" + conf.Lotus.Addr + "/rpc/v0"
	headers := http.Header{"Authorization": []string{"Bearer " + conf.Lotus.Token}}
	api, closer, err := client.NewFullNodeRPCV0(ctx, addr, headers)
	if err != nil {
		return nil, err
	}

	head, err := api.ChainHead(ctx)
	if err != nil {
		return nil, err
	}
	log.Infof("fullnode chain height: %d", head.Height())
	log.Infof("monitor miner list: %s", miners)

	n := &FullNode{
		ctx:    ctx,
		api:    api,
		closer: closer,
		miners: miners,
	}

	return n, nil
}

func (n *FullNode) Run(ctx context.Context) {
	go func() {
		t := time.NewTicker(time.Second * 30)
		for {
			select {
			case <-t.C:
				n.minerRecords()
				n.deadlineRecords()
			case <-ctx.Done():
				n.closer()
				log.Info("closed to fullnode")
				return
			}
		}
	}()
}
