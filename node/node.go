package node

import (
	"context"
	"fmt"
	"math/big"
	"math/rand"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/sheerun/queue"
	"github.com/wcgcyx/ethgen/idgen"
	tk "github.com/wcgcyx/ethgen/tracker"
)

type Node struct {
	ok bool

	window uint

	client *ethclient.Client

	tokenTracker tk.Tracker
	txTracker    tk.Tracker
	lock         sync.RWMutex
}

func NewNode(window uint, ap string, erc20 []string, erc721 []string) (*Node, error) {
	client, err := ethclient.Dial(ap)
	if err != nil {
		return nil, err
	}

	idGen := idgen.NewIdGenerator()
	tokenTracker := tk.NewBatchTracker([]tk.Tracker{
		tk.NewERC20BatchTracker(idGen, erc20, window),
		tk.NewERC721BatchTracker(idGen, erc721, window),
	})
	txTracker := tk.NewTransactionTracker(idGen, 3) // Near-head transaction, 3 blocks

	return &Node{
		ok:           false,
		window:       window,
		client:       client,
		tokenTracker: tokenTracker,
		txTracker:    txTracker,
		lock:         sync.RWMutex{},
	}, nil
}

func (n *Node) OK() bool {
	return n.ok
}

func (n *Node) Run() {
	ctx := context.Background()

	blkQueue := queue.New()
	go func() {
		go func() {
			last, err := n.client.BlockNumber(ctx)
			if err != nil {
				panic(err)
			}
			blk, err := n.client.BlockByNumber(ctx, big.NewInt(int64(last)))
			if err != nil {
				panic(err)
			}
			blkQueue.Append(blk)
			for {
				time.Sleep(time.Second * 15)
				current, err := n.client.BlockNumber(ctx)
				if err != nil {
					fmt.Printf("Fail to get block head: %v\n", err.Error())
				} else {
					for i := last + 1; i <= current; i++ {
						blk, err := n.client.BlockByNumber(ctx, big.NewInt(int64(i)))
						if err != nil {
							fmt.Printf("Fail to get block for %v: %v\n", i, err.Error())
						} else {
							blkQueue.Append(blk)
						}
					}
					last = current
				}
			}
		}()
	}()

	var headHeight uint64
	// Wait new head arrive
	fmt.Println("Wait new head to arrive...")
	for {
		time.Sleep(100 * time.Millisecond)
		item := blkQueue.Front()
		if item == nil {
			// No block  yet
			continue
		}
		blk := item.(*types.Block)
		headHeight = blk.NumberU64()
		fmt.Printf("New head arrived: %v, start syncing...\n", blk.NumberU64())
		break
	}

	currentHeight := uint64(0)
	for currentHeight = headHeight - uint64(n.window); currentHeight < headHeight; currentHeight++ {
		blk, err := n.client.BlockByNumber(ctx, big.NewInt(int64(currentHeight)))
		if err != nil {
			fmt.Printf("Warn: fail to get block %v: %v\n", currentHeight, err.Error())
			continue
		}
		err = n.tokenTracker.ApplyBlock(blk)
		if err != nil {
			fmt.Printf("Warn: fail to apply block at token tracker: %v\n", err.Error())
		}
		err = n.txTracker.ApplyBlock(blk)
		if err != nil {
			fmt.Printf("Warn: fail to apply block at tx tracker: %v\n", err.Error())
		}
		fmt.Printf("Imported blk: %v, target %v, diff %v\n", currentHeight, headHeight, headHeight-currentHeight)
		fmt.Printf("\tERC20: %v\n", n.tokenTracker.(*tk.BatchTracker).Trackers()[0].Status())
		fmt.Printf("\tERC721: %v\n", n.tokenTracker.(*tk.BatchTracker).Trackers()[1].Status())
		fmt.Printf("\tTxn: %v\n", n.txTracker.Status())
	}
	fmt.Println("Ready to generate queries...")
	n.ok = true
	for {
		item := blkQueue.Pop()
		blk := item.(*types.Block)
		n.lock.Lock()
		// Configure random source
		rand.Seed(int64(blk.NumberU64()))
		err := n.tokenTracker.ApplyBlock(blk)
		if err != nil {
			fmt.Printf("Warn: fail to apply block at token tracker: %v\n", err.Error())
		}
		err = n.txTracker.ApplyBlock(blk)
		if err != nil {
			fmt.Printf("Warn: fail to apply block at tx tracker: %v\n", err.Error())
		}
		fmt.Printf("Imported blk: %v\n", blk.NumberU64())
		fmt.Printf("\tERC20: %v\n", n.tokenTracker.(*tk.BatchTracker).Trackers()[0].Status())
		fmt.Printf("\tERC721: %v\n", n.tokenTracker.(*tk.BatchTracker).Trackers()[1].Status())
		fmt.Printf("\tTxn: %v\n", n.txTracker.Status())
		n.lock.Unlock()
	}
}

func (n *Node) GenerateQuery(number uint, tokenWeight uint, txWeight uint) ([]string, error) {
	n.lock.RLock()
	defer n.lock.RUnlock()
	number1 := number * tokenWeight / (tokenWeight + txWeight)
	number2 := number - number1
	var res1 []string
	var res2 []string
	var err1 error
	var err2 error
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		res1, err1 = n.tokenTracker.GenerateQuery(number1)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		res2, err2 = n.txTracker.GenerateQuery(number2)
	}()
	wg.Wait()
	if err1 != nil || err2 != nil {
		return nil, fmt.Errorf("%v-%v", err1, err2)
	}
	return append(res1, res2...), nil
}
