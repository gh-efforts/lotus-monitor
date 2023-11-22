package main

import (
	"io"
	"net/http"
	"os"

	"github.com/urfave/cli/v2"
)

var reloadCmd = &cli.Command{
	Name:  "reload",
	Usage: "reload config",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "connect",
			Value: "127.0.0.1:6789",
		},
	},
	Action: func(cctx *cli.Context) error {
		connect := cctx.String("connect")
		url := "http://" + connect + "/reload"

		r, err := http.Get(url)
		if err != nil {
			return err
		}

		if _, err := io.Copy(os.Stdout, r.Body); err != nil {
			return err
		}

		return r.Body.Close()
	},
}
