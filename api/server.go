package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/wcgcyx/ethgen/node"
	"github.com/filecoin-project/go-jsonrpc"
)

type Server struct {
	s http.Server
}

func NewServer(node *node.Node, port int) (*Server, error) {
	rpc := jsonrpc.NewServer()
	apiHandler := apiHandler{
		node: node,
	}
	rpc.Register("ethgen", &apiHandler)
	s := http.Server{
		Addr:           fmt.Sprintf("localhost:%v", port),
		Handler:        rpc,
		ReadTimeout:    60 * time.Second,
		WriteTimeout:   60 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	errChan := make(chan error, 1)
	go func() {
		// Start server.
		errChan <- s.ListenAndServe()
	}()
	// Wait for 3 seconds for the server to start
	tc := time.After(3 * time.Second)
	select {
	case <-tc:
		return &Server{s}, nil
	case err := <-errChan:
		return nil, err
	}
}
