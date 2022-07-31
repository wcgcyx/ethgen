package tracker

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"math/rand"

	"github.com/wcgcyx/ethgen/idgen"
	"github.com/ethereum/go-ethereum/core/types"
)

type TransactionTracker struct {
	// Id generator
	idGen idgen.IdGenerator
	// Signer
	signer types.Signer
	// Configuration
	maxBlocks uint
	// State of the tracker
	weight   uint
	accessed []uint
	// State of transactions
	transactionsCount []uint
	transactionsFlat  []wrappedTransaction
}

type wrappedTransaction struct {
	transaction *types.Transaction
	number      uint64
}

func NewTransactionTracker(idGen idgen.IdGenerator, maxBlocks uint) *TransactionTracker {
	return &TransactionTracker{
		idGen:             idGen,
		signer:            types.NewLondonSigner(big.NewInt(1)),
		maxBlocks:         maxBlocks,
		weight:            0,
		accessed:          make([]uint, maxBlocks),
		transactionsCount: make([]uint, maxBlocks),
		transactionsFlat:  make([]wrappedTransaction, 0),
	}
}

func (t *TransactionTracker) ApplyBlock(blk *types.Block) error {
	accessed := uint(0)
	transactionsToAdd := make([]wrappedTransaction, 0)
	// Apply block
	for i := 0; i < blk.Transactions().Len(); i++ {
		tx := blk.Transactions()[i]
		accessed += 1
		transactionsToAdd = append(transactionsToAdd, wrappedTransaction{
			transaction: tx,
			number:      blk.NumberU64(),
		})
	}
	// Pop
	pop1 := t.accessed[t.maxBlocks-1]
	t.weight -= pop1
	t.accessed = t.accessed[:t.maxBlocks-1]
	// Push
	t.weight += accessed
	t.accessed = append([]uint{accessed}, t.accessed...)
	// Pop
	pop2 := t.transactionsCount[t.maxBlocks-1]
	t.transactionsCount = t.transactionsCount[:t.maxBlocks-1]
	t.transactionsFlat = t.transactionsFlat[:len(t.transactionsFlat)-int(pop2)]
	// Push
	t.transactionsCount = append([]uint{uint(len(transactionsToAdd))}, t.transactionsCount...)
	t.transactionsFlat = append(transactionsToAdd, t.transactionsFlat...)
	return nil
}

func (t *TransactionTracker) CurrentWeight() uint {
	return t.weight
}

func (t *TransactionTracker) GenerateQuery(number uint) ([]string, error) {
	if len(t.transactionsFlat) == 0 {
		return nil, fmt.Errorf("empty transactions")
	}
	res := make([]string, number)
	for i := uint(0); i < number; i++ {
		index := rand.Intn(len(t.transactionsFlat))
		tx := t.transactionsFlat[index]
		id := t.idGen.Next()
		fromAddr, err := t.signer.Sender(tx.transaction)
		if err != nil {
			return nil, err
		}
		if tx.transaction.To() == nil {
			res[i] = fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"method":"eth_call","params":[{"from":"%v","data":"0x%v"},"0x%x"]}`, id, fromAddr.String(), hex.EncodeToString(tx.transaction.Data()), tx.number-1)
		} else {
			res[i] = fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"method":"eth_call","params":[{"to":"%v","from":"%v","data":"0x%v"},"0x%x"]}`, id, tx.transaction.To().String(), fromAddr.String(), hex.EncodeToString(tx.transaction.Data()), tx.number-1)
		}
	}
	return res, nil
}

func (t *TransactionTracker) Status() string {
	return fmt.Sprintf("%v", t.weight)
}
