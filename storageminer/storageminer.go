package storageminer

import (
	"context"
	"time"

	"github.com/gh-efforts/lotus-monitor/config"
	logging "github.com/ipfs/go-log/v2"
)

var log = logging.Logger("monitor/storageminer")

type StorageMiner struct {
	ctx context.Context
	dc  *config.DynamicConfig
}

func NewStorageMiner(ctx context.Context, dc *config.DynamicConfig) *StorageMiner {
	sm := &StorageMiner{
		ctx: ctx,
		dc:  dc,
	}

	return sm
}

func (m *StorageMiner) Run() {
	//m.jobsRecords()
	go func() {
		t := time.NewTicker(time.Duration(m.dc.RecordInterval.Miner))
		for {
			select {
			case <-t.C:
				m.jobsRecords()
			case <-m.ctx.Done():
				return
			}
		}
	}()
}
