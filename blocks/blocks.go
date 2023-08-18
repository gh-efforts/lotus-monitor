package blocks

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/gh-efforts/lotus-monitor/config"
	"github.com/gh-efforts/lotus-monitor/metrics"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
)

var log = logging.Logger("monitor/blocks")

type Block struct {
	Cid       cid.Cid         `json:"cid"`
	Miner     address.Address `json:"miner"`
	Height    abi.ChainEpoch  `json:"height"`
	Timestamp uint64          `json:"timestamp"`
	Took      time.Duration   `json:"took"`
}

type Blocks struct {
	ctx context.Context
	dc  *config.DynamicConfig

	lk     sync.Mutex
	blocks map[cid.Cid]Block
}

func NewBlocks(ctx context.Context, dc *config.DynamicConfig) *Blocks {
	b := &Blocks{
		ctx:    ctx,
		dc:     dc,
		blocks: make(map[cid.Cid]Block),
	}
	b.run()
	return b
}

func (b *Blocks) run() {
	go func() {
		t := time.NewTicker(time.Duration(b.dc.RecordInterval.Blocks))
		for {
			select {
			case <-t.C:
				if err := b.orphanCheck(); err != nil {
					log.Errorw("orphanCheck failed", "err", err)
					metrics.RecordError(b.ctx, "blocks/orphanCheck")
				} else {
					log.Debug("orphanCheck success")
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
		metrics.RecordError(r.Context(), "blocks/StatusInternalServerError")
		return
	}

	b.add(block)
	b.recordBlockTook(block)
	log.Infow("received block", "cid", block.Cid, "miner", block.Miner)
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

func (b *Blocks) recordBlockOnchain(block Block) {
	ctx, _ := tag.New(b.ctx,
		tag.Upsert(metrics.MinerID, block.Miner.String()),
	)

	stats.Record(ctx, metrics.BlockOnchain.M(1))
}

func (b *Blocks) recordOrphan(block Block) {
	ctx, _ := tag.New(b.ctx,
		tag.Upsert(metrics.MinerID, block.Miner.String()),
	)
	stats.Record(ctx, metrics.BlockOrphanCount.M(1))

	ctx, _ = tag.New(ctx,
		tag.Upsert(metrics.BlockCID, block.Cid.String()),
		tag.Upsert(metrics.BlockHeight, block.Height.String()),
	)
	stats.Record(ctx, metrics.BlockOrphan.M(1))

	time.AfterFunc(time.Minute, func() {
		stats.Record(ctx, metrics.BlockOrphan.M(0))
	})
}

func (b *Blocks) recordBlockTook(block Block) {
	ctx, _ := tag.New(b.ctx,
		tag.Upsert(metrics.MinerID, block.Miner.String()),
	)

	stats.Record(ctx, metrics.BlockTookDuration.M(block.Took.Seconds()))
}

func (b *Blocks) orphanCheck() error {
	stop := metrics.Timer(b.ctx, "blocks/orphanCheck")
	defer stop()

	api := b.dc.LotusApi

	head, err := api.ChainHead(b.ctx)
	if err != nil {
		return err
	}

	for _, block := range b.filter(head.Height() - abi.ChainEpoch(b.dc.OrphanCheckHeight)) {
		ts, err := api.ChainGetTipSetByHeight(b.ctx, block.Height, types.EmptyTSK)
		if err != nil {
			return err
		}

		if ts.Contains(block.Cid) {
			b.recordBlockOnchain(block)
		} else {
			b.recordOrphan(block)
			log.Infow("orphan block", "cid", block.Cid, "miner", block.Miner)
		}
		b.delete(block.Cid)
	}

	return nil
}
