package filfox

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gh-efforts/lotus-monitor/metrics"

	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
)

var ErrTooManyRequests = errors.New("429 Too Many Requests")

type rateLimit struct {
	limit     int
	remaining int
	reset     int
}
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

func (f *FilFox) luckyValueRecords() {
	stop := metrics.Timer(f.ctx, "filfox/luckyValueRecords")
	defer stop()

	miners := f.dc.MinersList()

	for _, maddr := range miners {
		days := []string{"1d", "7d", "30d"}
		for _, day := range days {
			rl, err := f.luckyValueRecord(maddr.String(), day)
			if err != nil {
				log.Errorw("luckyValueRecord failed", "miner", maddr, "err", err)
				metrics.RecordError(f.ctx, "filfox/luckyValueRecord")
				if err == ErrTooManyRequests {
					log.Warn("429 Too Many Requests, sleep one minute....")
					time.Sleep(time.Minute)
				}
				continue
			}
			if rl.remaining < 3 {
				log.Warnw("rate limit", "limit", rl.limit, "remaining", rl.remaining, "reset", rl.reset)
				time.Sleep(time.Duration(rl.reset+1) * time.Second)
			}
		}
	}
}

func (f *FilFox) luckyValueRecord(maddr, day string) (rateLimit, error) {
	url := fmt.Sprintf("%s/address/%s/mining-stats?duration=%s", f.dc.FilFoxURL, maddr, day)
	log.Debug(url)
	resp, err := f.Client.Get(url)
	if err != nil {
		return rateLimit{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusTooManyRequests {
			return rateLimit{}, ErrTooManyRequests
		}
		return rateLimit{}, errors.New(resp.Status)
	}

	var res MiningStats
	err = json.NewDecoder(resp.Body).Decode(&res)
	if err != nil {
		return rateLimit{}, err
	}

	ctx, _ := tag.New(f.ctx,
		tag.Upsert(metrics.MinerID, maddr),
		tag.Upsert(metrics.LuckyValueDay, day),
	)
	stats.Record(ctx, metrics.LuckyValue.M(res.LuckyValue))

	return parseRateLimit(resp.Header)
}

func parseRateLimit(header http.Header) (rateLimit, error) {
	log.Debug(header)
	limit, err := strconv.Atoi(header.Get("x-ratelimit-limit"))
	if err != nil {
		return rateLimit{}, err
	}
	remaining, err := strconv.Atoi(header.Get("x-ratelimit-remaining"))
	if err != nil {
		return rateLimit{}, err
	}
	reset, err := strconv.Atoi(header.Get("x-ratelimit-reset"))
	if err != nil {
		return rateLimit{}, err
	}

	r := rateLimit{
		limit:     limit,
		remaining: remaining,
		reset:     reset,
	}
	log.Debug(r)
	return r, nil
}
