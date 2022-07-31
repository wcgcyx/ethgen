package tracker

import (
	"fmt"
	"strings"
	"sync"

	"github.com/wcgcyx/ethgen/idgen"
	"github.com/ethereum/go-ethereum/core/types"

	wr "github.com/mroth/weightedrand"
)

type ERC20ContractTracker struct {
	// Configuration
	contractAddr string
	maxBlocks    uint
	// Contract state
	weight   uint
	accessed []uint
	// Sub-trackers
	balTracker Tracker
	apvTracker Tracker
}

func NewERC20ContractTracker(idGen idgen.IdGenerator, contractAddr string, maxBlocks uint) *ERC20ContractTracker {
	return &ERC20ContractTracker{
		contractAddr: strings.ToLower(contractAddr),
		maxBlocks:    maxBlocks,
		weight:       0,
		accessed:     make([]uint, maxBlocks),
		balTracker:   NewERC20BalanceTracker(idGen, contractAddr, maxBlocks),
		apvTracker:   NewERC20ApprovalTracker(idGen, contractAddr, maxBlocks),
	}
}

func (t *ERC20ContractTracker) Bal() Tracker {
	return t.balTracker
}

func (t *ERC20ContractTracker) Apv() Tracker {
	return t.apvTracker
}

func (t *ERC20ContractTracker) ApplyBlock(blk *types.Block) error {
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := t.balTracker.ApplyBlock(blk)
		if err != nil {
			fmt.Println(err.Error())
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := t.apvTracker.ApplyBlock(blk)
		if err != nil {
			fmt.Println(err.Error())
		}
	}()
	accessed := uint(0)
	// Apply block
	for i := 0; i < blk.Transactions().Len(); i++ {
		tx := blk.Transactions()[i]
		if tx.To() == nil {
			continue
		}
		to := strings.ToLower(tx.To().String()[2:])
		if to == t.contractAddr {
			// Contract accessed once
			accessed++
		}
	}
	// Update method state
	// Pop
	pop1 := t.accessed[t.maxBlocks-1]
	t.weight -= pop1
	t.accessed = t.accessed[:t.maxBlocks-1]
	// Push
	t.weight += accessed
	t.accessed = append([]uint{accessed}, t.accessed...)
	wg.Wait()
	return nil
}

func (t *ERC20ContractTracker) CurrentWeight() uint {
	return t.weight
}

func (t *ERC20ContractTracker) GenerateQuery(number uint) ([]string, error) {
	contractChooser, err := wr.NewChooser(
		wr.NewChoice(1, t.balTracker.CurrentWeight()),
		wr.NewChoice(2, t.apvTracker.CurrentWeight()),
	)
	if err != nil {
		return nil, err
	}
	bal := uint(0)
	apv := uint(0)
	for i := uint(0); i < number; i++ {
		method := contractChooser.Pick().(int)
		if method == 1 {
			bal++
		} else {
			apv++
		}
	}
	var res1 []string
	var res2 []string
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		var err error
		res1, err = t.balTracker.GenerateQuery(bal)
		if err != nil {
			fmt.Println(err.Error())
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		var err error
		res2, err = t.apvTracker.GenerateQuery(apv)
		if err != nil {
			fmt.Println(err.Error())
		}
	}()
	wg.Wait()
	return append(res1, res2...), nil
}

func (t *ERC20ContractTracker) Status() string {
	return fmt.Sprintf("(%v-%v)", t.balTracker.CurrentWeight(), t.apvTracker.CurrentWeight())
}
