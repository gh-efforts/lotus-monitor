package blocks

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api/v0api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/gh-efforts/lotus-monitor/config"
	"github.com/gh-efforts/lotus-monitor/metrics"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
)

var log = logging.Logger("blocks")

type Block struct {
	Cid              cid.Cid         `json:"cid"`
	Miner            address.Address `json:"miner"`
	Height           abi.ChainEpoch  `json:"height"`
	Timestamp        uint64          `json:"timestamp"`
	BaseDeltaSeconds uint64          `json:"baseDeltaSeconds"`
	Took             time.Duration   `json:"took"`
	Now              time.Time       `json:"now"`
}

type Blocks struct {
	ctx      context.Context
	api      v0api.FullNode
	interval time.Duration

	lk     sync.Mutex
	blocks map[cid.Cid]Block
}

func NewBlocks(ctx context.Context, api v0api.FullNode, conf *config.Config) (*Blocks, error) {
	interval, err := time.ParseDuration(conf.RecordInterval.Blocks)
	if err != nil {
		return nil, err
	}
	b := &Blocks{
		ctx:      ctx,
		api:      api,
		interval: interval,
		blocks:   make(map[cid.Cid]Block),
	}
	b.run()
	log.Infow("NewBlocks")
	return b, nil
}

func (b *Blocks) run() {
	go func() {
		t := time.NewTicker(time.Minute * 3)
		for {
			select {
			case <-t.C:
				if err := b.orphanCheck(); err != nil {
					log.Error(err)
				}
			case <-b.ctx.Done():
				return
			}
		}
	}()
}

func (b *Blocks) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var block Block
	err := json.NewDecoder(r.Body).Decode(&block)
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	b.add(block)
	log.Infow("reveived block", "cid", block.Cid, "miner", block.Miner)
}

func (b *Blocks) add(block Block) {
	b.lk.Lock()
	defer b.lk.Unlock()

	b.blocks[block.Cid] = block
}

func (b *Blocks) delete(blockCid cid.Cid) {
	b.lk.Lock()
	defer b.lk.Unlock()

	delete(b.blocks, blockCid)
}

func (b *Blocks) filter(head abi.ChainEpoch) []Block {
	b.lk.Lock()
	defer b.lk.Unlock()

	var bb []Block
	for _, block := range b.blocks {
		if block.Height < head {
			bb = append(bb, block)
		}
	}

	return bb
}

func (b *Blocks) recordBlock(block Block) {
	ctx, _ := tag.New(b.ctx,
		tag.Upsert(metrics.MinerID, block.Miner.String()),
		tag.Upsert(metrics.BlockCID, block.Cid.String()),
		tag.Upsert(metrics.BlockHeight, block.Height.String()),
	)
	stats.Record(ctx, metrics.MiningBlockDuration.M(float64(block.Took)))
}

func (b *Blocks) recordOrphan(block Block) {
	ctx, _ := tag.New(b.ctx,
		tag.Upsert(metrics.MinerID, block.Miner.String()),
		tag.Upsert(metrics.BlockCID, block.Cid.String()),
		tag.Upsert(metrics.BlockHeight, block.Height.String()),
	)
	stats.Record(ctx, metrics.MiningOrphanBlock.M(1))
}

func (b *Blocks) orphanCheck() error {
	head, err := b.api.ChainHead(b.ctx)
	if err != nil {
		return err
	}

	for _, block := range b.filter(head.Height()) {
		ts, err := b.api.ChainGetTipSetByHeight(b.ctx, block.Height, types.EmptyTSK)
		if err != nil {
			return err
		}

		if ts.Contains(block.Cid) {
			b.recordBlock(block)
		} else {
			b.recordOrphan(block)
		}

		b.delete(block.Cid)
	}

	return nil
}
