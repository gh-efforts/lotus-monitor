package main

import (
	"net/http"
	"os"
	"time"

	"contrib.go.opencensus.io/exporter/prometheus"
	"github.com/urfave/cli/v2"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"

	logging "github.com/ipfs/go-log/v2"

	cliutil "github.com/filecoin-project/lotus/cli/util"
	"github.com/gh-efforts/lotus-monitor/blocks"
	"github.com/gh-efforts/lotus-monitor/config"
	"github.com/gh-efforts/lotus-monitor/filfox"
	"github.com/gh-efforts/lotus-monitor/fullnode"
	"github.com/gh-efforts/lotus-monitor/metrics"
	"github.com/gh-efforts/lotus-monitor/storageminer"
)

var (
	log = logging.Logger("main")
)

func main() {
	logging.SetLogLevel("*", "INFO")

	local := []*cli.Command{
		runCmd,
	}

	app := &cli.App{
		Name:     "lotus-monitor",
		Usage:    "lotus monitor server",
		Version:  UserVersion(),
		Commands: local,
	}

	if err := app.Run(os.Args); err != nil {
		log.Errorf("%+v", err)
	}
}

var runCmd = &cli.Command{
	Name: "run",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "listen",
			Value: "0.0.0.0:6789",
		},
		&cli.StringFlag{
			Name:  "config",
			Value: "./config.json",
			Usage: "specify config file path",
		},
	},
	Action: func(cctx *cli.Context) error {
		log.Info("starting lotus monitor...")

		ctx := cliutil.ReqContext(cctx)
		dc, err := config.NewDynamicConfig(ctx, cctx.String("config"))
		if err != nil {
			return err
		}

		exporter, err := prometheus.NewExporter(prometheus.Options{
			Namespace: "lotusmonitor",
		})
		if err != nil {
			return err
		}

		ctx, _ = tag.New(ctx,
			tag.Insert(metrics.Version, BuildVersion),
			tag.Insert(metrics.Commit, CurrentCommit),
		)
		if err := view.Register(
			metrics.Views...,
		); err != nil {
			return err
		}
		stats.Record(ctx, metrics.Info.M(1))

		fullnode.NewFullNode(ctx, dc).Run()
		storageminer.NewStorageMiner(ctx, dc).Run()
		filfox.NewFilFox(ctx, dc).Run()

		listen := cctx.String("listen")
		log.Infow("monitor server", "listen", listen)

		go func() {
			<-ctx.Done()
			time.Sleep(time.Millisecond * 200)
			log.Info("closed monitor server")
			os.Exit(0)
		}()

		http.Handle("/metrics", exporter)
		http.Handle("/blocks", blocks.NewBlocks(ctx, dc))
		server := &http.Server{
			Addr: listen,
		}
		return server.ListenAndServe()
	},
}
