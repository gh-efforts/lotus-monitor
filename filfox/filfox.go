package filfox

import (
	"context"
	"time"

	"github.com/gh-efforts/lotus-monitor/config"
	logging "github.com/ipfs/go-log/v2"
)

var log = logging.Logger("filfox")

type FilFox struct {
	ctx      context.Context
	URL      string
	miners   []string
	interval time.Duration
}

func NewFilFox(ctx context.Context, conf *config.Config) (*FilFox, error) {
	interval, err := time.ParseDuration(conf.RecordInterval.FilFox)
	if err != nil {
		return nil, err
	}

	miners := []string{}
	for m := range conf.Miners {
		miners = append(miners, m)
	}
	f := &FilFox{
		ctx:      ctx,
		URL:      conf.FilFoxURL,
		miners:   miners,
		interval: interval,
	}

	log.Infow("NewFilFox", "interval", conf.RecordInterval.FilFox)
	return f, nil
}

func (f *FilFox) Run(ctx context.Context) {
	go func() {
		t := time.NewTicker(f.interval)
		for {
			select {
			case <-t.C:
				f.luckyValueRecords()
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (f *FilFox) luckyValueRecords() error {
	return nil
}
