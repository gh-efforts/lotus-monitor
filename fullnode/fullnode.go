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

	API    v0api.FullNode
	closer jsonrpc.ClientCloser

	miners []address.Address

	interval time.Duration
}

func NewFullNode(ctx context.Context, conf *config.Config) (*FullNode, error) {
	interval, err := time.ParseDuration(conf.RecordInterval.Lotus)
	if err != nil {
		return nil, err
	}
	var miners []address.Address
	for m := range conf.Miners {
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

	n := &FullNode{
		ctx:      ctx,
		API:      api,
		closer:   closer,
		miners:   miners,
		interval: interval,
	}
	n.run()
	log.Infow("NewFullNode", "interval", interval.String(), "head", head.Height(), "miners", miners)
	return n, nil
}

func (n *FullNode) run() {
	go func() {
		t := time.NewTicker(n.interval)
		for {
			select {
			case <-t.C:
				n.minerRecords()
				n.deadlineRecords()
			case <-n.ctx.Done():
				n.closer()
				log.Info("closed to fullnode")
				return
			}
		}
	}()
}
