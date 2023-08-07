package storageminer

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api/client"
	"github.com/filecoin-project/lotus/api/v0api"
	"github.com/gh-efforts/lotus-monitor/config"
	logging "github.com/ipfs/go-log/v2"
)

var log = logging.Logger("storageminer")

type minerInfo struct {
	api    v0api.StorageMiner
	closer jsonrpc.ClientCloser
	size   abi.SectorSize
}
type StorageMiner struct {
	ctx      context.Context
	miners   map[address.Address]minerInfo
	running  map[abi.SectorSize]map[string]time.Duration
	tasks    []string
	interval time.Duration
}

func NewStorageMiner(ctx context.Context, conf *config.Config) (*StorageMiner, error) {
	miners := map[address.Address]minerInfo{}
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
		size, err := api.ActorSectorSize(ctx, maddr)
		if err != nil {
			return nil, err
		}

		miners[maddr] = minerInfo{api: api, closer: closer, size: size}
	}

	running := map[abi.SectorSize]map[string]time.Duration{}
	entry32 := map[string]time.Duration{}
	entry64 := map[string]time.Duration{}
	tasks := []string{}
	for task, td := range conf.Running {
		tasks = append(tasks, task)
		td32, err := time.ParseDuration(td[0])
		if err != nil {
			return nil, err
		}
		td64, err := time.ParseDuration(td[1])
		if err != nil {
			return nil, err
		}
		entry32[task] = td32
		entry64[task] = td64
	}
	running[abi.SectorSize(34359738368)] = entry32
	running[abi.SectorSize(68719476736)] = entry64

	interval, err := time.ParseDuration(conf.RecordInterval.Miner)
	if err != nil {
		return nil, err
	}

	sm := &StorageMiner{
		ctx:      ctx,
		miners:   miners,
		running:  running,
		tasks:    tasks,
		interval: interval,
	}

	sm.run()
	log.Infow("NewStorageMiner", "interval", conf.RecordInterval.Miner, "running", fmt.Sprintf("%s", running))
	return sm, nil

}

func (m *StorageMiner) run() {
	go func() {
		t := time.NewTicker(m.interval)
		for {
			select {
			case <-t.C:
				m.jobsRecords()
			case <-m.ctx.Done():
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
