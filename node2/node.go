package node2

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/wcgcyx/ethgen/idgen"
	"golang.org/x/exp/rand"
)

var client = &http.Client{
	Transport: &http.Transport{
		MaxIdleConns:        1000,
		MaxIdleConnsPerHost: 1000,
		DisableKeepAlives:   false,
	},
}

type Node struct {
	// id generator
	idGen idgen.IdGenerator
	// eth client to query for new block
	client *ethclient.Client
	// target access url
	targetAP string
	// signer used to do sender recovery
	signer types.Signer
	// past transaction data used to generate eth_call queries
	last           uint
	window         uint
	txnsLock       sync.RWMutex
	txnsCount      []uint
	txnsParamsFlat []string
	txnsGas        []uint64
}

func StartNode(targetAP string, window uint, concurrency uint, delay time.Duration) error {
	client, err := ethclient.Dial(targetAP)
	if err != nil {
		return err
	}
	chainId, err := client.ChainID(context.Background())
	if err != nil {
		return err
	}
	node := &Node{
		idGen:          idgen.NewIdGenerator(),
		client:         client,
		targetAP:       targetAP,
		signer:         types.NewCancunSigner(chainId),
		last:           0,
		window:         window,
		txnsLock:       sync.RWMutex{},
		txnsCount:      make([]uint, 0),
		txnsParamsFlat: make([]string, 0),
	}
	node.Run(concurrency, delay)
	return nil
}

func (n *Node) Run(concurrency uint, delay time.Duration) {
	ctx := context.Background()
	// First query last N blocks
	current, err := n.client.BlockNumber(ctx)
	if err != nil {
		panic(err)
	}
	for i := current - uint64(n.window); i <= current; i++ {
		blk, err := n.client.BlockByNumber(ctx, big.NewInt(int64(i)))
		if err != nil {
			panic(err)
		}
		n.ApplyBlock(blk)
	}
	go n.StartBenchmark(1000, concurrency, delay)
	// Keep query new block every 5 seconds
	for {
		time.Sleep(5 * time.Second)
		new, err := n.client.BlockNumber(ctx)
		if err != nil {
			panic(err)
		}
		if new > current {
			for i := current + 1; i <= new; i++ {
				blk, err := n.client.BlockByNumber(ctx, big.NewInt(int64(i)))
				if err != nil {
					panic(err)
				}
				n.ApplyBlock(blk)
			}
			current = new
		}
	}
}

func (n *Node) ApplyBlock(blk *types.Block) {
	n.txnsLock.Lock()
	defer n.txnsLock.Unlock()

	// fmt.Printf("Process block %v-%v: %v txns\n", blk.NumberU64(), blk.Hash(), blk.Transactions().Len())
	transactionsToAdd := make([]string, 0)
	gasUsed := make([]uint64, 0)
	receipts, err := n.client.BlockReceipts(context.Background(), rpc.BlockNumberOrHashWithHash(blk.Hash(), true))
	if err != nil {
		panic(err)
	}
	// Apply block
	for i := 0; i < blk.Transactions().Len(); i++ {
		txn := blk.Transactions()[i]
		fromAddr, err := n.signer.Sender(txn)
		if err != nil {
			panic(err)
		}
		if txn.To() == nil {
			transactionsToAdd = append(transactionsToAdd,
				fmt.Sprintf(`[{"from":"%v","data":"0x%v"},"0x%x"]`, fromAddr.String(), hex.EncodeToString(txn.Data()), blk.NumberU64()-1))
		} else {
			transactionsToAdd = append(transactionsToAdd,
				fmt.Sprintf(`[{"to":"%v","from":"%v","data":"0x%v"},"0x%x"]`, txn.To().String(), fromAddr.String(), hex.EncodeToString(txn.Data()), blk.NumberU64()-1))
		}
		gasUsed = append(gasUsed, receipts[i].GasUsed)
	}
	// Push
	n.txnsCount = append(n.txnsCount, uint(len(transactionsToAdd)))
	n.txnsParamsFlat = append(n.txnsParamsFlat, transactionsToAdd...)
	n.txnsGas = append(n.txnsGas, gasUsed...)
	// Keep pop if new block - last > window
	if n.last == 0 {
		n.last = uint(blk.NumberU64())
		return
	}
	for blk.NumberU64()-uint64(n.last) > uint64(n.window) {
		// Pop one
		removed := n.txnsCount[0]
		n.txnsCount = n.txnsCount[1:]
		n.txnsParamsFlat = n.txnsParamsFlat[removed:]
		n.txnsGas = n.txnsGas[removed:]
		// fmt.Printf("Popped block %v, %v removed, total %v\n", n.last, removed, len(n.txnsParamsFlat))
		n.last++
	}
}

func (n *Node) GenerateQuery(number uint) ([]string, []uint64, error) {
	n.txnsLock.RLock()
	defer n.txnsLock.RUnlock()
	if len(n.txnsParamsFlat) == 0 {
		return nil, nil, fmt.Errorf("empty transactions")
	}
	res1 := make([]string, number)
	res2 := make([]uint64, number)
	for i := uint(0); i < number; i++ {
		index := rand.Intn(len(n.txnsParamsFlat))
		tx := n.txnsParamsFlat[index]
		id := n.idGen.Next()
		res1[i] = fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"method":"eth_call","params":%v}`, id, tx)
		res2[i] = n.txnsGas[index]
	}
	return res1, res2, nil
}

func (n *Node) StartBenchmark(number uint, concurrency uint, delay time.Duration) error {
	fmt.Println("Start benchmark...")
	actorLocks := make([]sync.RWMutex, concurrency)
	actorReqs := make([]uint64, concurrency)
	actorGases := make([]uint64, concurrency)
	actorTime := make([]time.Duration, concurrency)

	for i := 0; i < int(concurrency); i++ {
		actorLocks[i] = sync.RWMutex{}
		actorTime[i] = time.Duration(0)
		actorGases[i] = 0
		fmt.Printf("Start actor %v...\n", i)
		go func(index int) {
			for {
				// Generate 1k queries
				queries, gases, err := n.GenerateQuery(number)
				if err != nil {
					panic(err)
				}
				for i := 0; i < len(queries); i++ {
					start := time.Now()
					resp, err := client.Post(n.targetAP, "application/json", bytes.NewReader([]byte(queries[i])))
					taken := time.Since(start)
					if err != nil {
						fmt.Printf("Fail to request: %v\n", err.Error())
					} else {
						// Add result
						// TODO: Failed request?
						resp.Body.Close()
						actorLocks[index].Lock()
						actorGases[index] += gases[i]
						actorReqs[index]++
						actorTime[index] += taken
						actorLocks[index].Unlock()
						if resp.StatusCode != 200 {
							fmt.Println("Failed request", resp.StatusCode)
						}
					}
					time.Sleep(delay)
				}
			}
		}(i)
	}

	sample := 0
	for {
		// Report gas rate per second
		time.Sleep(5 * time.Second)
		totalTime := time.Duration(0)
		totalGas := uint64(0)
		totalReq := uint64(0)
		for i := 0; i < int(concurrency); i++ {
			actorLocks[i].Lock()
			totalTime += actorTime[i]
			totalGas += actorGases[i]
			totalReq += actorReqs[i]
			actorTime[i] = time.Duration(0)
			actorGases[i] = 0
			actorReqs[i] = 0
			actorLocks[i].Unlock()
		}
		gasRate := float64(totalGas) / 1e6 / totalTime.Seconds()
		reqRate := float64(totalReq) / totalTime.Seconds()
		fmt.Printf("Sample - %v - Gas speed: %.2f M/s, %.2f Req/s\n", sample, gasRate, reqRate)
		sample++
	}
}
