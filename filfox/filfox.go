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
	ctx    context.Context
	dc     *config.DynamicConfig
	Client http.Client
}

func NewFilFox(ctx context.Context, dc *config.DynamicConfig) *FilFox {
	f := &FilFox{
		ctx:    ctx,
		dc:     dc,
		Client: http.Client{},
	}
	return f
}

func (f *FilFox) Run() {
	go func() {
		t := time.NewTicker(time.Duration(f.dc.RecordInterval.FilFox))
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
