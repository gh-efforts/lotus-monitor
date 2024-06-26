package main

import (
	"net/http"
	"os"
	"time"

	"contrib.go.opencensus.io/exporter/prometheus"
	"github.com/mitchellh/go-homedir"
	"github.com/urfave/cli/v2"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"

	logging "github.com/ipfs/go-log/v2"

	_ "net/http/pprof"

	cliutil "github.com/filecoin-project/lotus/cli/util"
	"github.com/gh-efforts/lotus-monitor/blocks"
	"github.com/gh-efforts/lotus-monitor/build"
	"github.com/gh-efforts/lotus-monitor/config"
	"github.com/gh-efforts/lotus-monitor/control"
	"github.com/gh-efforts/lotus-monitor/filfox"
	"github.com/gh-efforts/lotus-monitor/fullnode"
	"github.com/gh-efforts/lotus-monitor/metrics"
	"github.com/gh-efforts/lotus-monitor/mpool"
	"github.com/gh-efforts/lotus-monitor/storageminer"
)

var (
	log = logging.Logger("monitor/main")
)

func main() {
	logging.SetLogLevel("*", "INFO")

	local := []*cli.Command{
		runCmd,
		reloadCmd,
		minerCmd,
		pprofCmd,
	}

	app := &cli.App{
		Name:     "lotus-monitor",
		Usage:    "lotus monitor server",
		Version:  build.UserVersion(),
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
		&cli.BoolFlag{
			Name:  "debug",
			Value: false,
		},
	},
	Action: func(cctx *cli.Context) error {
		path, err := homedir.Expand(cctx.String("config"))
		if err != nil {
			return err
		}

		if cctx.Bool("debug") {
			logging.SetLogLevelRegex("monitor/*", "DEBUG")
		}

		log.Info("starting lotus monitor...")

		ctx := cliutil.ReqContext(cctx)
		dc, err := config.NewDynamicConfig(ctx, path)
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
			tag.Insert(metrics.Version, build.BuildVersion),
			tag.Insert(metrics.Commit, build.CurrentCommit),
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
		control.NewControl(ctx, dc).Run()
		mpool.NewMpool(ctx, dc).Run()

		listen := cctx.String("listen")
		log.Infow("monitor server", "listen", listen)

		http.Handle("/metrics", exporter)
		http.Handle("/blocks", blocks.NewBlocks(ctx, dc))
		http.Handle("/reload", http.HandlerFunc(dc.ReloadHandle))
		http.Handle("/miner/add", http.HandlerFunc(dc.AddMinerHandle))
		http.Handle("/miner/remove/", http.HandlerFunc(dc.RemoveMinerHandle))
		http.Handle("/miner/list", http.HandlerFunc(dc.ListMinerHandle))

		server := &http.Server{
			Addr: listen,
		}

		go func() {
			<-ctx.Done()
			time.Sleep(time.Millisecond * 100)
			log.Info("closed monitor server")
			server.Shutdown(ctx)
		}()

		return server.ListenAndServe()
	},
}
