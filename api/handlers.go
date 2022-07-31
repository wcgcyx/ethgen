package api

import (
	"github.com/wcgcyx/ethgen/node"
)

type apiHandler struct {
	node *node.Node
}

func (h *apiHandler) Upcheck() bool {
	return h.node.OK()
}

func (h *apiHandler) Generate(number uint) ([]string, error) {
	return h.node.GenerateQuery(number)
}
