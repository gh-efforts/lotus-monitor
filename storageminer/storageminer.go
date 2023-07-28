package storageminer

import (
	"context"
	"time"

	"github.com/filecoin-project/lotus/api"
	logging "github.com/ipfs/go-log/v2"
)

var log = logging.Logger("storageminer")

type StorageMiner struct {
	api api.StorageMiner
}

func NewStorageMiner(api api.StorageMiner) *StorageMiner {
	return &StorageMiner{
		api: api,
	}
}

func (m *StorageMiner) Run(ctx context.Context) {
	go func() {
		t := time.NewTicker(time.Minute * 5)
		for {
			select {
			case <-t.C:
				m.jobsRecord(ctx)
			case <-ctx.Done():
				return
			}
		}
	}()

	log.Info("lotus monitor storage miner running...")
}

func (m *StorageMiner) jobsRecord(ctx context.Context) {

}
