package main

import (
	"net/http"
	"os"

	"contrib.go.opencensus.io/exporter/prometheus"
	"github.com/urfave/cli/v2"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"

	logging "github.com/ipfs/go-log/v2"

	cliutil "github.com/filecoin-project/lotus/cli/util"
	"github.com/gh-efforts/lotus-monitor/fullnode"
	"github.com/gh-efforts/lotus-monitor/metrics"
	"github.com/gh-efforts/lotus-monitor/storageminer"
)

var (
	log = logging.Logger("lotus-monitor")
)

func main() {
	app := &cli.App{
		Name:    "lotus-monitor",
		Usage:   "lotus monitor server",
		Version: UserVersion(),
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "listen",
				Usage: "host address and port the monitor will listen on",
				Value: "0.0.0.0:6789",
				EnvVars: []string{
					"LOTUS_MONITOR_LISTEN",
				},
			},
			&cli.StringFlag{
				Name:    "repo",
				EnvVars: []string{"LOTUS_PATH"},
				Value:   "~/.lotus",
			},
			&cli.StringFlag{
				Name:    "miner-repo",
				Aliases: []string{"storagerepo"},
				EnvVars: []string{"LOTUS_MINER_PATH", "LOTUS_STORAGE_PATH"},
				Value:   "~/.lotusminer",
			},
		},
		Action: func(cctx *cli.Context) error {
			api, closer, err := cliutil.GetFullNodeAPI(cctx)
			if err != nil {
				return err
			}
			defer closer()

			minerApi, mcloser, err := cliutil.GetStorageMinerAPI(cctx)
			if err != nil {
				return err
			}
			defer mcloser()

			pe, err := prometheus.NewExporter(prometheus.Options{
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

			maddr, err := minerApi.ActorAddress(ctx)
			if err != nil {
				return err
			}

			n := fullnode.NewFullNode(api, maddr)
			err = n.Init(ctx)
			if err != nil {
				return err
			}
			n.Run(ctx)

			storageminer.NewStorageMiner(minerApi).Run(ctx)

			log.Infow("start monitor server", "listen", cctx.String("listen"))
			mux := http.NewServeMux()
			mux.Handle("/metrics", pe)
			return http.ListenAndServe(cctx.String("listen"), mux)
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Errorf("%+v", err)
	}
}
