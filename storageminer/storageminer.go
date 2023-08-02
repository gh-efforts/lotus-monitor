package storageminer

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/lotus/api/client"
	"github.com/filecoin-project/lotus/api/v0api"
	"github.com/gh-efforts/lotus-monitor/config"
	logging "github.com/ipfs/go-log/v2"
)

var log = logging.Logger("storageminer")

type minerApi struct {
	api    v0api.StorageMiner
	closer jsonrpc.ClientCloser
}
type StorageMiner struct {
	ctx    context.Context
	miners map[address.Address]minerApi
}

func NewStorageMiner(ctx context.Context, conf *config.Config) (*StorageMiner, error) {
	miners := map[address.Address]minerApi{}
	for m, info := range conf.Miners {
		if info.Addr == "" || info.Token == "" {
			log.Warnf("miner: %s api info empty", m)
			continue
		}
		maddr, err := address.NewFromString(m)
		if err != nil {
			return nil, err
		}

		addr := "ws://" + info.Addr + "/rpc/v0"
		headers := http.Header{"Authorization": []string{"Bearer " + info.Token}}
		api, closer, err := client.NewStorageMinerRPCV0(ctx, addr, headers)
		if err != nil {
			return nil, err
		}

		apiAddr, err := api.ActorAddress(ctx)
		if err != nil {
			return nil, err
		}
		if apiAddr != maddr {
			return nil, fmt.Errorf("maddr not match, config maddr: %s, api maddr: %s", maddr, apiAddr)
		}

		miners[maddr] = minerApi{api: api, closer: closer}
	}

	sm := &StorageMiner{
		ctx:    ctx,
		miners: miners,
	}

	log.Info("Init storage miner success")
	return sm, nil

}

func (m *StorageMiner) Run(ctx context.Context) {
	go func() {
		t := time.NewTicker(time.Minute * 5)
		for {
			select {
			case <-t.C:
				m.jobsRecord(ctx)
			case <-ctx.Done():
				m.close()
				log.Info("closed to storage miner")
				return
			}
		}
	}()
}

func (m *StorageMiner) close() {
	for _, api := range m.miners {
		api.closer()
	}
}

func (m *StorageMiner) jobsRecord(ctx context.Context) {

}
