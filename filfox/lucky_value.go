package filfox

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/gh-efforts/lotus-monitor/metrics"

	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
)

type MiningStats struct {
	RawBytePowerGrowth    string  `json:"rawBytePowerGrowth"`
	QualityAdjPowerGrowth string  `json:"qualityAdjPowerGrowth"`
	RawBytePowerDelta     string  `json:"rawBytePowerDelta"`
	QualityAdjPowerDelta  string  `json:"qualityAdjPowerDelta"`
	BlocksMined           int     `json:"blocksMined"`
	WeightedBlocksMined   int     `json:"weightedBlocksMined"`
	TotalRewards          string  `json:"totalRewards"`
	NetworkTotalRewards   string  `json:"networkTotalRewards"`
	EquivalentMiners      float64 `json:"equivalentMiners"`
	RewardPerByte         float64 `json:"rewardPerByte"`
	LuckyValue            float64 `json:"luckyValue"`
	DurationPercentage    int     `json:"durationPercentage"`
}

func (f *FilFox) luckyValueRecords() error {
	wg := sync.WaitGroup{}
	wg.Add(len(f.miners))

	for _, maddr := range f.miners {
		go func(maddr string) {
			defer wg.Done()
			err := f.luckyValueRecord(maddr)
			if err != nil {
				log.Errorw("luckyValueRecord failed", "miner", maddr, "err", err)
			} else {
				log.Infow("luckyValueRecord success", "miner", maddr)
			}
		}(maddr)
	}
	wg.Wait()
	return nil
}

func (f *FilFox) luckyValueRecord(maddr string) (err error) {
	days := []string{"1d", "7d", "30d"}
	for _, day := range days {
		err = f._luckyValueRecord(maddr, day)
		if err != nil {
			return err
		}
	}

	return nil
}

func (f *FilFox) _luckyValueRecord(maddr, day string) error {
	url := fmt.Sprintf("%s/address/%s/mining-stats?duration=%s", f.URL, maddr, day)
	log.Debug(url)
	resp, err := f.Client.Get(url)
	if err != nil {
		return err
	}

	var res MiningStats
	err = json.NewDecoder(resp.Body).Decode(&res)
	if err != nil {
		return err
	}

	ctx, _ := tag.New(f.ctx,
		tag.Upsert(metrics.MinerID, maddr),
		tag.Upsert(metrics.LuckyValueDay, day),
	)
	stats.Record(ctx, metrics.LuckyValue.M(res.LuckyValue))

	return nil
}
