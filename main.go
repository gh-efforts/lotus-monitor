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
	log.Info("starting lotus monitor...")

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
			Name:  "config",
			Value: "./config.json",
			Usage: "specify config file path",
		},
	},
	Action: func(cctx *cli.Context) error {
		conf, err := config.LoadConfig(cctx.String("config"))
		if err != nil {
			return err
		}

		exporter, err := prometheus.NewExporter(prometheus.Options{
			Namespace: "lotusmonitor",
		})
		if err != nil {
			return err
		}

		ctx, _ := tag.New(cliutil.ReqContext(cctx),
			tag.Insert(metrics.Version, BuildVersion),
			tag.Insert(metrics.Commit, CurrentCommit),
		)
		if err := view.Register(
			metrics.Views...,
		); err != nil {
			return err
		}
		stats.Record(ctx, metrics.Info.M(1))

		n, err := fullnode.NewFullNode(ctx, conf)
		if err != nil {
			return err
		}

		if _, err = storageminer.NewStorageMiner(ctx, conf); err != nil {
			return err
		}

		if _, err = filfox.NewFilFox(ctx, conf); err != nil {
			return err
		}

		b, err := blocks.NewBlocks(ctx, n.API, conf)
		if err != nil {
			return err
		}

		log.Infow("monitor server", "listen", conf.Listen)

		go func() {
			<-ctx.Done()
			time.Sleep(time.Millisecond * 200)
			log.Info("closed monitor server")
			os.Exit(0)
		}()

		http.Handle("/metrics", exporter)
		http.Handle("/blocks", b)
		server := &http.Server{
			Addr: conf.Listen,
		}
		return server.ListenAndServe()
	},
}
