// 预估用来做windPost的control地址余额能用多少天。
// 当前高度余额 / (昨天高度余额-当前高度余额）= 未来可用天数

package control

import (
	"context"
	"math"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/gh-efforts/lotus-monitor/config"
	"github.com/gh-efforts/lotus-monitor/metrics"
	logging "github.com/ipfs/go-log/v2"
	"github.com/robfig/cron/v3"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
)

var log = logging.Logger("monitor/control")

type Control struct {
	ctx context.Context
	dc  *config.DynamicConfig
}

func NewControl(ctx context.Context, dc *config.DynamicConfig) *Control {
	return &Control{
		ctx: ctx,
		dc:  dc,
	}
}

func (c *Control) Run() {
	//上海时间每天9:30执行一次
	_, err := cron.New().AddFunc("CRON_TZ=Asia/Shanghai 30 09 * * *", func() {
		c.controlRecords()
	})

	if err != nil {
		panic(err)
	}
}

func (c *Control) controlRecords() {
	stop := metrics.Timer(c.ctx, "control/controlRecords")
	defer stop()

	log.Debugw("cron controlRecords", "time", time.Now())

	tody, err := c.dc.LotusApi.ChainHead(c.ctx)
	if err != nil {
		log.Error(err)
		return
	}
	yesterday, err := c.dc.LotusApi.ChainGetTipSetByHeight(c.ctx, tody.Height()-abi.ChainEpoch(2880), types.EmptyTSK)
	if err != nil {
		log.Error(err)
		return
	}

	miners := c.dc.MinersList()
	for _, maddr := range miners {
		if err := c.controlRecord(maddr, tody.Key(), yesterday.Key()); err != nil {
			log.Errorw("controlRecord failed", "miner", maddr, "err", err)
			metrics.RecordError(c.ctx, "control/controlRecord")
		}
	}
}

func (c *Control) controlRecord(maddr address.Address, tody, yesterday types.TipSetKey) error {
	ctx, _ := tag.New(c.ctx,
		tag.Upsert(metrics.MinerID, maddr.String()),
	)

	mi, err := c.dc.LotusApi.StateMinerInfo(ctx, maddr, tody)
	if err != nil {
		return err
	}

	for _, ca := range mi.ControlAddresses {
		actor, err := c.dc.LotusApi.StateGetActor(ctx, ca, tody)
		if err != nil {
			return err
		}
		actor2, err := c.dc.LotusApi.StateGetActor(ctx, ca, yesterday)
		if err != nil {
			return err
		}

		var days float64
		delta := types.BigSub(actor2.Balance, actor.Balance).Abs()
		if delta.IsZero() {
			days = math.MaxFloat64
		} else {
			days = types.BigDivFloat(actor.Balance, delta)
		}

		ctx, _ = tag.New(ctx,
			tag.Upsert(metrics.ActorAddress, ca.String()),
		)
		stats.Record(ctx, metrics.ControlDays.M(days))
	}

	return nil
}
