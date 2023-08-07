package filfox

import (
	"context"
	"net/http"
	"time"

	"github.com/gh-efforts/lotus-monitor/config"
	logging "github.com/ipfs/go-log/v2"
)

var log = logging.Logger("filfox")

type FilFox struct {
	ctx      context.Context
	URL      string
	Client   http.Client
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
		Client:   http.Client{},
		miners:   miners,
		interval: interval,
	}

	f.run()
	log.Infow("NewFilFox", "interval", interval.String())
	return f, nil
}

func (f *FilFox) run() {
	go func() {
		t := time.NewTicker(f.interval)
		for {
			select {
			case <-t.C:
				f.luckyValueRecords()
			case <-f.ctx.Done():
				return
			}
		}
	}()
}
