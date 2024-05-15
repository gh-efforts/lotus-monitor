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
	Cid       string         `json:"cid"`
	Miner     string         `json:"miner"`
	Height    abi.ChainEpoch `json:"height"`
	Timestamp uint64         `json:"timestamp"`
	Took      float64        `json:"took"`
}

type Blocks struct {
	ctx context.Context
	dc  *config.DynamicConfig

	lk     sync.Mutex
	blocks map[string]Block
}

func NewBlocks(ctx context.Context, dc *config.DynamicConfig) *Blocks {
	b := &Blocks{
		ctx:    ctx,
		dc:     dc,
		blocks: make(map[string]Block),
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

	if _, err := cid.Decode(block.Cid); err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		metrics.RecordError(r.Context(), "blocks/StatusBadRequest")
		return
	}
	if _, err := address.NewFromString(block.Miner); err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		metrics.RecordError(r.Context(), "blocks/StatusBadRequest")
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

func (b *Blocks) delete(blockCid string) {
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
		tag.Upsert(metrics.MinerID, block.Miner),
	)

	stats.Record(ctx, metrics.BlockOnchain.M(1))
}

func (b *Blocks) recordOrphan(block Block) {
	ctx, _ := tag.New(b.ctx,
		tag.Upsert(metrics.MinerID, block.Miner),
	)
	stats.Record(ctx, metrics.BlockOrphanCount.M(1))

	ctx, _ = tag.New(ctx,
		tag.Upsert(metrics.BlockCID, block.Cid),
		tag.Upsert(metrics.BlockHeight, block.Height.String()),
	)
	stats.Record(ctx, metrics.BlockOrphan.M(1))

	time.AfterFunc(time.Duration(b.dc.OrphanReset), func() {
		stats.Record(ctx, metrics.BlockOrphan.M(0))
	})
}

func (b *Blocks) recordBlockTook(block Block) {
	ctx, _ := tag.New(b.ctx,
		tag.Upsert(metrics.MinerID, block.Miner),
	)

	stats.Record(ctx, metrics.BlockTookDuration.M(block.Took))
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
		c, err := cid.Decode(block.Cid)
		if err != nil {
			return err
		}
		if ts.Contains(c) {
			b.recordBlockOnchain(block)
		} else {
			b.recordOrphan(block)
			log.Infow("orphan block", "cid", block.Cid, "miner", block.Miner)
		}
		b.delete(block.Cid)
	}

	return nil
}
