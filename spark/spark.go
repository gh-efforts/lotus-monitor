package spark

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/robfig/cron/v3"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"

	"github.com/gh-efforts/lotus-monitor/config"
	"github.com/gh-efforts/lotus-monitor/metrics"
	logging "github.com/ipfs/go-log/v2"
)

var log = logging.Logger("monitor/spark")

const BaseURL = "https://stats.filspark.com"

type Spark struct {
	ctx context.Context
	dc  *config.DynamicConfig
}

type RetrievalRate struct {
	MinerID     string  `json:"miner_id"`
	Total       string  `json:"total"`
	Successful  string  `json:"successful"`
	SuccessRate float64 `json:"success_rate"`
}

func New(ctx context.Context, dc *config.DynamicConfig) *Spark {
	return &Spark{
		ctx: ctx,
		dc:  dc,
	}
}

func (s *Spark) Run() {
	//record once immediately
	if err := s.sparkRecords(); err != nil {
		log.Errorw("sparkRecords", "error", err)
	}

	//record every day at 9am
	r := cron.New()
	_, err := r.AddFunc("CRON_TZ=Asia/Shanghai 00 09 * * *", func() {
		if err := s.sparkRecords(); err != nil {
			log.Errorw("sparkRecords", "error", err)
		}
	})
	if err != nil {
		panic(err)
	}
	r.Start()
}

func (s *Spark) sparkRecords() error {
	stop := metrics.Timer(s.ctx, "spark/sparkRecords")
	defer stop()

	date := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	url := fmt.Sprintf("%s/miners/retrieval-success-rate/summary?from=%s&to=%s", BaseURL, date, date)
	log.Debugw("spark", "url", url)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.New(resp.Status)
	}

	var rates []RetrievalRate
	if err := json.NewDecoder(resp.Body).Decode(&rates); err != nil {
		return err
	}

	miners := s.dc.MinersList()
	minerSet := make(map[string]struct{}, len(miners))
	for _, miner := range miners {
		minerSet[miner.String()] = struct{}{}
	}

	var filteredRates []RetrievalRate
	for _, rate := range rates {
		if _, ok := minerSet[rate.MinerID]; ok {
			filteredRates = append(filteredRates, rate)
		}
	}
	log.Debugw("spark", "rates", filteredRates)

	for _, rate := range filteredRates {
		ctx, _ := tag.New(s.ctx,
			tag.Upsert(metrics.MinerID, rate.MinerID),
		)
		stats.Record(ctx, metrics.SparkRetrievalRate.M(rate.SuccessRate))
	}

	return nil
}
