package fullnode

import (
	"context"
	"time"

	"github.com/gh-efforts/lotus-monitor/config"
	logging "github.com/ipfs/go-log/v2"
)

var log = logging.Logger("monitor/fullnode")

type FullNode struct {
	ctx context.Context
	dc  *config.DynamicConfig
}

func NewFullNode(ctx context.Context, dc *config.DynamicConfig) *FullNode {
	return &FullNode{
		ctx: ctx,
		dc:  dc,
	}
}

func (n *FullNode) Run() {
	go func() {
		//n.deadlineRecords()
		//n.minerRecords()
		t := time.NewTicker(time.Duration(n.dc.RecordInterval.Lotus))
		for {
			select {
			case <-t.C:
				n.deadlineRecords()
				n.minerRecords()
			case <-n.ctx.Done():
				return
			}
		}
	}()
}
