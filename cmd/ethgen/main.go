package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/wcgcyx/ethgen/api"
	"github.com/wcgcyx/ethgen/node"
	"github.com/wcgcyx/ethgen/request"
	"github.com/urfave/cli/v2"
)

type config struct {
	ERC20  []string `json:"erc20"`
	ERC721 []string `json:"erc721"`
}

func main() {
	app := &cli.App{
		Name:  "ethgen",
		Usage: "A adaptive eth_call query generator",
		Commands: []*cli.Command{
			{
				Name: "daemon",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:  "window",
						Value: 2880,
						Usage: "specify window size",
					},
					&cli.IntFlag{
						Name:  "token_weight",
						Value: 85,
						Usage: "specify token weight",
					},
					&cli.IntFlag{
						Name:  "tx_weight",
						Value: 15,
						Usage: "specify tx weight",
					},
					&cli.StringFlag{
						Name:  "config",
						Value: "",
						Usage: "specify contract config file",
					},
					&cli.IntFlag{
						Name:  "port",
						Value: 9999,
						Usage: "specify api port",
					},
					&cli.StringFlag{
						Name:  "chain_ap",
						Value: "http://127.0.0.1:8545",
						Usage: "specify chain access addr",
					},
				},
				Action: func(c *cli.Context) error {
					// First try to read config
					data, err := os.ReadFile(c.String("config"))
					if err != nil {
						return err
					}
					cfg := config{}
					err = json.Unmarshal(data, &cfg)
					if err != nil {
						return err
					}
					n, err := node.NewNode(uint(c.Int("window")), c.String("chain_ap"), cfg.ERC20, cfg.ERC721, uint(c.Int("token_weight")), uint(c.Int("tx_weight")))
					if err != nil {
						return err
					}
					// Start API server
					_, err = api.NewServer(n, c.Int("port"))
					if err != nil {
						return err
					}
					n.Run()
					return nil
				},
			},
			{
				Name: "generate",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:  "port",
						Value: 9999,
						Usage: "specify api port",
					},
					&cli.IntFlag{
						Name:  "number",
						Value: 250,
						Usage: "specify number to generate",
					},
					&cli.DurationFlag{
						Name:  "duration",
						Value: 0,
						Usage: "specify frequency",
					},
				},
				Action: func(c *cli.Context) error {
					// First try to get client
					client, closer, err := api.NewClient(c.Context, c.Int("port"))
					if err != nil {
						return err
					}
					defer closer()
					// First check if client is ready
					ready := client.Upcheck()
					if !ready {
						return fmt.Errorf("daemon not ready to generate queries")
					}
					// Generate queries
					duration := c.Duration("duration")
					for {
						queries, err := client.Generate(uint(c.Int("number")))
						if err != nil {
							return err
						}
						for _, query := range queries {
							fmt.Println(query)
						}
						if duration == 0 {
							break
						}
						time.Sleep(duration)
					}
					return nil
				},
			},
			{
				Name: "request",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:  "port",
						Value: 9999,
						Usage: "specify api port",
					},
					&cli.IntFlag{
						Name:  "number",
						Value: 250,
						Usage: "specify number to generate",
					},
					&cli.DurationFlag{
						Name:  "duration",
						Value: time.Second,
						Usage: "specify frequency",
					},
					&cli.IntFlag{
						Name:  "concurrency",
						Value: 5,
						Usage: "specify concurrency",
					},
					&cli.StringFlag{
						Name:  "chain_ap",
						Value: "http://127.0.0.1:8545",
						Usage: "specify chain access addr",
					},
				},
				Action: func(c *cli.Context) error {
					// First try to get client
					client, closer, err := api.NewClient(c.Context, c.Int("port"))
					if err != nil {
						return err
					}
					defer closer()
					// First check if client is ready
					ready := client.Upcheck()
					if !ready {
						return fmt.Errorf("daemon not ready to generate queries")
					}
					// Generate queries
					duration := c.Duration("duration")
					for {
						queries, err := client.Generate(uint(c.Int("number")))
						if err != nil {
							return err
						}
						err = request.Request(c.String("chain_ap"), queries, duration, c.Int("concurrency"))
						if err != nil {
							return err
						}
						if duration == 0 {
							break
						}
					}
					return nil
				},
			},
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err.Error())
	}
}
