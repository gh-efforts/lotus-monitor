package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/gh-efforts/lotus-monitor/config"
	"github.com/urfave/cli/v2"
)

var minerCmd = &cli.Command{
	Name:  "miner",
	Usage: "manage miner list",
	Subcommands: []*cli.Command{
		minerAddCmd,
		minerRemoveCmd,
		minerListCmd,
	},
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "connect",
			Value: "127.0.0.1:6789",
		},
	},
}

var minerAddCmd = &cli.Command{
	Name:  "add",
	Usage: "add new miner",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name: "miner-id",
		},
		&cli.StringFlag{
			Name: "addr",
		},
		&cli.StringFlag{
			Name: "token",
		},
	},
	Action: func(cctx *cli.Context) error {
		id := cctx.String("miner-id")
		addr := cctx.String("addr")
		token := cctx.String("token")
		if id == "" {
			return errors.New("miner-id is empty")
		}

		api := config.APIInfo{
			Addr:  addr,
			Token: token,
		}
		miner := config.MinerAPI{
			Miner: id,
			API:   api,
		}
		body, err := json.Marshal(&miner)
		if err != nil {
			return err
		}

		url := fmt.Sprintf("http://%s/miner/add", cctx.String("connect"))
		resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			r, err := io.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			return fmt.Errorf("status: %s msg: %s", resp.Status, string(r))
		}
		return nil
	},
}

var minerRemoveCmd = &cli.Command{
	Name:  "remove",
	Usage: "remove miner",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name: "miner-id",
		},
	},
	Action: func(cctx *cli.Context) error {
		id := cctx.String("miner-id")
		if id == "" {
			return errors.New("miner-id is empty")
		}

		url := fmt.Sprintf("http://%s/miner/remove/%s", cctx.String("connect"), id)
		resp, err := http.Get(url)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			r, err := io.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			return fmt.Errorf("status: %s msg: %s", resp.Status, string(r))
		}
		return nil
	},
}

var minerListCmd = &cli.Command{
	Name:  "list",
	Usage: "list all miner",
	Action: func(cctx *cli.Context) error {
		url := fmt.Sprintf("http://%s/miner/list", cctx.String("connect"))
		resp, err := http.Get(url)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("status: %s", resp.Status)
		}

		var miners []string
		err = json.NewDecoder(resp.Body).Decode(&miners)
		if err != nil {
			return err
		}

		fmt.Println(miners)

		return nil
	},
}
