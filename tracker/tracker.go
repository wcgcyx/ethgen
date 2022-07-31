package tracker

import (
	"github.com/ethereum/go-ethereum/core/types"
)

type Tracker interface {
	ApplyBlock(blk *types.Block) error

	CurrentWeight() uint

	GenerateQuery(number uint) ([]string, error)

	Status() string
}
