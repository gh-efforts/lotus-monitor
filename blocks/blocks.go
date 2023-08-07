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
	ctx context.Context
	api v0api.FullNode

	lk     sync.Mutex
	blocks map[cid.Cid]Block
}

func NewBlocks(ctx context.Context, api v0api.FullNode) *Blocks {
	b := &Blocks{
		ctx:    ctx,
		api:    api,
		blocks: make(map[cid.Cid]Block),
	}
	b.run()
	log.Infow("NewBlocks")
	return b
}

func (b *Blocks) run() {
	go func() {
		t := time.NewTicker(time.Minute * 3)
		for {
			select {
			case <-t.C:
				b.orphanCheck()
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
	b.recordBlock(block)
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

func (b *Blocks) filter() []Block {
	b.lk.Lock()
	defer b.lk.Unlock()

	var bb []Block
	for _, block := range b.blocks {
		if time.Since(block.Now) > 3*time.Minute {
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

func (b *Blocks) orphanCheck() {
	for _, block := range b.filter() {
		bh, err := b.api.ChainGetBlock(b.ctx, block.Cid)
		if err != nil {
			log.Error(err)
			continue
		}
		if bh == nil {
			b.recordOrphan(block)
		}
		b.delete(block.Cid)
	}
}
