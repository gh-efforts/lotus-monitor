package config

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/api/client"
	"github.com/filecoin-project/lotus/api/v0api"
	"github.com/filecoin-project/lotus/api/v1api"
	cliutil "github.com/filecoin-project/lotus/cli/util"
	"github.com/filecoin-project/lotus/storage/sealer/sealtasks"
	logging "github.com/ipfs/go-log/v2"
)

var log = logging.Logger("monitor/config")

type Duration time.Duration

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	td, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	*d = Duration(td)

	return nil
}

type APIInfo struct {
	Addr  string `json:"addr"`
	Token string `json:"token"`
}

type RecordInterval struct {
	Lotus  Duration `json:"lotus"`
	Miner  Duration `json:"miner"`
	FilFox Duration `json:"filFox"`
	Blocks Duration `json:"blocks"`
}

type Config struct {
	Lotus             []string                                           `json:"lotus"`
	Miners            map[string]APIInfo                                 `json:"miners"`
	Running           map[abi.SectorSize]map[sealtasks.TaskType]Duration `json:"running"`
	RecordInterval    RecordInterval                                     `json:"recordInterval"`
	FilFoxURL         string                                             `json:"filFoxURL"`
	OrphanCheckHeight int                                                `json:"orphanCheckHeight"`
}

type MinerInfo struct {
	Api     v0api.StorageMiner
	closer  jsonrpc.ClientCloser
	Address address.Address
	Size    abi.SectorSize
}
type DynamicConfig struct {
	ctx context.Context

	cfg           *Config
	path          string
	reloadRequest chan struct{}

	LotusApi api.FullNode
	closer   jsonrpc.ClientCloser

	Running           map[abi.SectorSize]map[sealtasks.TaskType]Duration
	RecordInterval    RecordInterval
	FilFoxURL         string
	OrphanCheckHeight int

	lk     sync.RWMutex
	miners map[address.Address]MinerInfo
}

type httpHead struct {
	addr   string
	header http.Header
}

func getFullNodeAPIV1(ctx context.Context, ainfoCfg []string) (v1api.FullNode, jsonrpc.ClientCloser, error) {
	var httpHeads []httpHead
	version := "v1"
	{
		if len(ainfoCfg) == 0 {
			return nil, nil, fmt.Errorf("could not get API info: ainfoCfg is empty")
		}
		for _, i := range ainfoCfg {
			ainfo := cliutil.ParseApiInfo(i)
			addr, err := ainfo.DialArgs(version)
			if err != nil {
				return nil, nil, fmt.Errorf("could not get DialArgs: %w", err)
			}
			httpHeads = append(httpHeads, httpHead{addr: addr, header: ainfo.AuthHeader()})
		}
	}

	var fullNodes []api.FullNode
	var closers []jsonrpc.ClientCloser

	for _, head := range httpHeads {
		v1api, closer, err := client.NewFullNodeRPCV1(ctx, head.addr, head.header)
		if err != nil {
			log.Warnf("Not able to establish connection to node with addr: %s, Reason: %s", head.addr, err.Error())
			continue
		}
		log.Infow("connected to lotus", "addr", head.addr)
		fullNodes = append(fullNodes, v1api)
		closers = append(closers, closer)
	}

	if len(httpHeads) > 1 && len(fullNodes) < 2 {
		return nil, nil, fmt.Errorf("not able to establish connection to more than a single node")
	}

	finalCloser := func() {
		for _, c := range closers {
			c()
		}
	}

	var v1API api.FullNodeStruct
	cliutil.FullNodeProxy(fullNodes, &v1API)

	v, err := v1API.Version(ctx)
	if err != nil {
		return nil, nil, err
	}
	if !v.APIVersion.EqMajorMinor(api.FullAPIVersion1) {
		return nil, nil, fmt.Errorf("remote API version didn't match (expected %s, remote %s)", api.FullAPIVersion1, v.APIVersion)
	}

	return &v1API, finalCloser, nil
}

func NewDynamicConfig(ctx context.Context, path string) (*DynamicConfig, error) {
	cfg, err := LoadConfig(path)
	if err != nil {
		return nil, err
	}
	a, c, err := getFullNodeAPIV1(ctx, cfg.Lotus)
	if err != nil {
		return nil, err
	}

	head, err := a.ChainHead(ctx)
	if err != nil {
		return nil, err
	}
	log.Infow("connected to lotus proxy", "head", head.Height())

	miners := map[address.Address]MinerInfo{}
	for m, info := range cfg.Miners {
		mi, err := toMinerInfo(ctx, m, info)
		if err != nil {
			return nil, err
		}

		miners[mi.Address] = mi
	}

	dc := &DynamicConfig{
		ctx:               ctx,
		cfg:               cfg,
		path:              path,
		reloadRequest:     make(chan struct{}, 10),
		LotusApi:          a,
		closer:            c,
		Running:           cfg.Running,
		RecordInterval:    cfg.RecordInterval,
		FilFoxURL:         cfg.FilFoxURL,
		OrphanCheckHeight: cfg.OrphanCheckHeight,
		miners:            miners,
	}
	dc.watch()

	return dc, nil
}

func (dc *DynamicConfig) watch() {
	go func() {
		for {
			select {
			case <-dc.reloadRequest:
				log.Info("recieved chan reloading config...")
				if err := dc.reload(); err != nil {
					log.Errorf("reload config: %s", err)
				}
			case <-dc.ctx.Done():
				dc.Close()
				log.Info("closed all api")
				return
			}
		}
	}()
}

func (dc *DynamicConfig) reload() error {
	cfg, err := LoadConfig(dc.path)
	if err != nil {
		return err
	}
	new := cfg.Miners
	old := dc.cfg.Miners

	var update []MinerInfo
	var insert []MinerInfo
	var remove []address.Address

	for nk, nv := range new {
		if v, ok := old[nk]; ok {
			if nv != v {
				mi, err := toMinerInfo(dc.ctx, nk, nv)
				if err != nil {
					log.Error(err)
					continue
				}
				update = append(update, mi)
			}
		} else {
			mi, err := toMinerInfo(dc.ctx, nk, nv)
			if err != nil {
				log.Error(err)
				continue
			}
			insert = append(insert, mi)
		}
	}

	for k := range old {
		if _, ok := new[k]; !ok {
			maddr, err := address.NewFromString(k)
			if err != nil {
				log.Error(err)
				continue
			}
			remove = append(remove, maddr)
		}
	}

	dc.cfg.Miners = new
	dc.updateMiners(update, insert, remove)

	var uu []address.Address
	for _, u := range update {
		uu = append(uu, u.Address)
	}
	var ii []address.Address
	for _, i := range insert {
		ii = append(ii, i.Address)
	}
	log.Infow("reload config success", "update", uu, "insert", ii, "remove", remove)
	return nil
}

func (dc *DynamicConfig) updateMiners(update []MinerInfo, insert []MinerInfo, remove []address.Address) {
	dc.lk.Lock()
	defer dc.lk.Unlock()

	for _, u := range update {
		if c := dc.miners[u.Address].closer; c != nil {
			c()
			log.Infow("closed old miner api", "miner", u.Address)
		}
		dc.miners[u.Address] = u
	}

	for _, i := range insert {
		dc.miners[i.Address] = i
	}

	for _, r := range remove {
		if c := dc.miners[r].closer; c != nil {
			c()
			log.Infow("closed removed miner api", "miner", r)
		}
		delete(dc.miners, r)
	}
}

func (dc *DynamicConfig) addMiner(miner address.Address, api APIInfo) error {
	dc.lk.Lock()
	defer dc.lk.Unlock()

	mi, err := toMinerInfo(dc.ctx, miner.String(), api)
	if err != nil {
		return err
	}

	_, ok := dc.miners[miner]
	if ok {
		if c := dc.miners[miner].closer; c != nil {
			c()
			log.Infow("closed removed miner api", "miner", miner)
		}
	}

	dc.miners[miner] = mi

	return dc.updateConfig(miner.String(), &api)
}

func (dc *DynamicConfig) removeMiner(miner address.Address) error {
	dc.lk.Lock()
	defer dc.lk.Unlock()

	if c := dc.miners[miner].closer; c != nil {
		c()
		log.Infow("closed removed miner api", "miner", miner)
	}

	delete(dc.miners, miner)

	return dc.updateConfig(miner.String(), nil)
}

func (dc *DynamicConfig) updateConfig(miner string, api *APIInfo) error {
	c, err := LoadConfig(dc.path)
	if err != nil {
		return err
	}

	if api == nil {
		delete(c.Miners, miner)
	} else {
		c.Miners[miner] = *api
	}

	data, err := json.MarshalIndent(c, "", "\t")
	if err != nil {
		return err
	}

	err = os.WriteFile(dc.path, data, 0666)
	if err != nil {
		return err
	}

	dc.cfg.Miners = c.Miners

	log.Infof("update config: %s", miner)
	return nil
}

func (dc *DynamicConfig) Close() {
	dc.closer()

	for _, m := range dc.miners {
		if m.closer != nil {
			m.closer()
		}
	}
}

func (dc *DynamicConfig) MinersList() []address.Address {
	dc.lk.RLock()
	defer dc.lk.RUnlock()

	var ret []address.Address
	for m := range dc.miners {
		ret = append(ret, m)
	}

	return ret
}

func (dc *DynamicConfig) MinersInfo() []MinerInfo {
	dc.lk.RLock()
	defer dc.lk.RUnlock()

	var ret []MinerInfo
	for _, mi := range dc.miners {
		if mi.Api != nil {
			ret = append(ret, mi)
		}
	}

	return ret
}

func LoadConfig(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var c Config
	err = json.Unmarshal(raw, &c)
	if err != nil {
		return nil, err
	}

	return &c, nil
}

func DefaultConfig() *Config {
	lotus := []string{"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJBbGxvdyI6WyJyZWFkIiwid3JpdGUiLCJzaWduIiwiYWRtaW4iXX0.rdUrdfAtXRQjqTQSDR_mHTJnU1loMg49bED-78WIrRE:/ip4/127.0.0.1/tcp/1234/http"}
	lotus = append(lotus, "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJBbGxvdyI6WyJyZWFkIiwid3JpdGUiLCJzaWduIiwiYWRtaW4iXX0.Znxv4ZV4djSOqhvPDGGANOzYpfSexMohq4Ba-9dJlaU:/ip4/10.122.4.30/tcp/1234/http")
	miner := APIInfo{
		Addr:  "10.122.1.29:2345",
		Token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJBbGxvdyI6WyJyZWFkIiwid3JpdGUiLCJzaWduIiwiYWRtaW4iXX0.tlJ8d4RIudknLHrKDSjyKzfbh8hGp9Ez1FZszblQLAI",
	}
	miner64 := APIInfo{
		Addr:  "10.122.1.29:2346",
		Token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJBbGxvdyI6WyJyZWFkIiwid3JpdGUiLCJzaWduIiwiYWRtaW4iXX0.7ZoJAcyY9ictWUdWsiV5AwmSTPHCczkT8Y6mTiN3Azw",
	}

	miners := make(map[string]APIInfo)
	miners["t017387"] = miner
	miners["t028064"] = miner64
	miners["t01037"] = APIInfo{}
	miners["t03751"] = APIInfo{}

	running := map[abi.SectorSize]map[sealtasks.TaskType]Duration{}
	entry32 := map[sealtasks.TaskType]Duration{
		sealtasks.TTAddPiece:   Duration(time.Minute * 5),
		sealtasks.TTPreCommit1: Duration(time.Hour * 5),
		sealtasks.TTPreCommit2: Duration(time.Minute * 15),
		sealtasks.TTCommit1:    Duration(time.Minute),
		sealtasks.TTFetch:      Duration(time.Minute * 5),
	}
	entry64 := map[sealtasks.TaskType]Duration{
		sealtasks.TTAddPiece:   Duration(time.Minute * 10),
		sealtasks.TTPreCommit1: Duration(time.Hour * 10),
		sealtasks.TTPreCommit2: Duration(time.Minute * 30),
		sealtasks.TTCommit1:    Duration(time.Minute * 2),
		sealtasks.TTFetch:      Duration(time.Minute * 10),
	}

	running[abi.SectorSize(34359738368)] = entry32
	running[abi.SectorSize(68719476736)] = entry64

	interval := RecordInterval{
		Lotus:  Duration(time.Second * 30),
		Miner:  Duration(time.Minute * 5),
		FilFox: Duration(time.Hour),
		Blocks: Duration(time.Minute),
	}

	return &Config{
		Lotus:             lotus,
		Miners:            miners,
		Running:           running,
		RecordInterval:    interval,
		FilFoxURL:         "https://calibration.filfox.info/api/v1", //mainnet: "https://filfox.info/api/v1"
		OrphanCheckHeight: 3,
	}
}

func toMinerInfo(ctx context.Context, m string, info APIInfo) (MinerInfo, error) {
	maddr, err := address.NewFromString(m)
	if err != nil {
		return MinerInfo{}, err
	}

	if info.Addr == "" || info.Token == "" {
		log.Warnf("miner: %s api info empty", maddr)
		return MinerInfo{Api: nil, Address: maddr}, nil
	}

	addr := "ws://" + info.Addr + "/rpc/v0"
	headers := http.Header{"Authorization": []string{"Bearer " + info.Token}}
	api, closer, err := client.NewStorageMinerRPCV0(ctx, addr, headers)
	if err != nil {
		return MinerInfo{}, err
	}

	apiAddr, err := api.ActorAddress(ctx)
	if err != nil {
		return MinerInfo{}, err
	}
	if apiAddr != maddr {
		closer()
		return MinerInfo{}, fmt.Errorf("maddr not match, config maddr: %s, api maddr: %s", maddr, apiAddr)
	}
	size, err := api.ActorSectorSize(ctx, maddr)
	if err != nil {
		return MinerInfo{}, err
	}
	log.Infow("connected to miner", "miner", maddr, "addr", info.Addr)

	return MinerInfo{Api: api, closer: closer, Address: maddr, Size: size}, nil
}
