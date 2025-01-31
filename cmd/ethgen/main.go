package main

import (
	"fmt"
	"os"
	"time"

	"github.com/urfave/cli/v2"
	"github.com/wcgcyx/ethgen/node2"
)

func main() {
	app := &cli.App{
		Name:  "ethgen",
		Usage: "A adaptive eth_call query generator",
		Commands: []*cli.Command{
			{
				Name: "start",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:  "window",
						Value: 64,
						Usage: "specify window size",
					},
					&cli.StringFlag{
						Name:  "chain_ap",
						Value: "http://localhost:8545",
						Usage: "specify chain access addr",
					},
					&cli.IntFlag{
						Name:  "concurrency",
						Value: 5,
						Usage: "specify concurrency",
					},
					&cli.DurationFlag{
						Name:  "frequency",
						Value: 5 * time.Millisecond,
						Usage: "specify frequency",
					},
				},
				Action: func(c *cli.Context) error {
					return node2.StartNode(c.String("chain_ap"), uint(c.Int("window")), uint(c.Int("concurrency")), c.Duration("frequency"))
				},
			},
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err.Error())
	}
}
