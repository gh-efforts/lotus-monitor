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
	//n.minerRecords()
	//n.deadlineRecords()
	go func() {
		t := time.NewTicker(time.Duration(n.dc.RecordInterval.Lotus))
		for {
			select {
			case <-t.C:
				n.minerRecords()
				n.deadlineRecords()
			case <-n.ctx.Done():
				return
			}
		}
	}()
}
