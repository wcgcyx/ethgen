package api

import (
	"context"
	"fmt"

	"github.com/filecoin-project/go-jsonrpc"
)

func NewClient(ctx context.Context, port int) (API, jsonrpc.ClientCloser, error) {
	var client API
	closer, err := jsonrpc.NewClient(ctx, fmt.Sprintf("http://localhost:%v", port), "ethgen", &client, nil)
	return client, closer, err
}
