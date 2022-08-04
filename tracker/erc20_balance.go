package tracker

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"math/rand"
	"strings"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/wcgcyx/ethgen/idgen"
)

type ERC20BalanceTracker struct {
	// Id generator
	idGen idgen.IdGenerator
	// Signer
	signer types.Signer
	// Configuration
	contractAddr string
	maxBlocks    uint
	// State of the method
	weight   uint
	accessed []uint
	// State of the account list
	accountsAccessed []uint
	accountsFlat     []string
	// Current block
	blk uint64
}

func NewERC20BalanceTracker(idGen idgen.IdGenerator, contractAddr string, maxBlocks uint) *ERC20BalanceTracker {
	return &ERC20BalanceTracker{
		idGen:            idGen,
		signer:           types.NewLondonSigner(big.NewInt(1)),
		contractAddr:     strings.ToLower(contractAddr),
		maxBlocks:        maxBlocks,
		weight:           0,
		accessed:         make([]uint, maxBlocks),
		accountsAccessed: make([]uint, maxBlocks),
		accountsFlat:     make([]string, 0),
	}
}

func (t *ERC20BalanceTracker) ApplyBlock(blk *types.Block) error {
	accessed := uint(0)
	accountsToAdd := make([]string, 0)
	// Apply block
	for i := 0; i < blk.Transactions().Len(); i++ {
		tx := blk.Transactions()[i]
		if tx.To() == nil {
			continue
		}
		to := strings.ToLower(tx.To().String()[2:])
		if to == t.contractAddr {
			// Check method
			if len(tx.Data()) < 4 {
				continue
			}
			method := hex.EncodeToString(tx.Data()[:4])
			if method == "a9059cbb" {
				// transfer
				fromAddr, err := t.signer.Sender(tx)
				if err != nil {
					return err
				}
				sender := strings.ToLower(fromAddr.String()[2:])
				recipient := hex.EncodeToString(tx.Data()[16:36])
				accountsToAdd = append(accountsToAdd, []string{recipient, sender}...)
				// Method accessed once
				accessed++
			} else if method == "23b872dd" {
				// transferFrom
				sender := hex.EncodeToString(tx.Data()[16:36])
				recipient := hex.EncodeToString(tx.Data()[48:68])
				accountsToAdd = append(accountsToAdd, []string{recipient, sender}...)
				// Method accessed once
				accessed++
			}
		}
	}
	t.blk = blk.NumberU64()
	// Update method state
	// Pop
	pop1 := t.accessed[t.maxBlocks-1]
	t.weight -= pop1
	t.accessed = t.accessed[:t.maxBlocks-1]
	// Push
	t.weight += accessed
	t.accessed = append([]uint{accessed}, t.accessed...)
	// Update accounts state
	// Pop
	pop2 := t.accountsAccessed[t.maxBlocks-1]
	t.accountsAccessed = t.accountsAccessed[:t.maxBlocks-1]
	t.accountsFlat = t.accountsFlat[:len(t.accountsFlat)-int(pop2)]
	// Push
	t.accountsAccessed = append([]uint{uint(len(accountsToAdd))}, t.accountsAccessed...)
	t.accountsFlat = append(accountsToAdd, t.accountsFlat...)
	return nil
}

func (t *ERC20BalanceTracker) CurrentWeight() uint {
	return t.weight
}

func (t *ERC20BalanceTracker) GenerateQuery(number uint) ([]string, error) {
	if len(t.accountsFlat) == 0 {
		return nil, fmt.Errorf("empty accounts")
	}
	res := make([]string, number)
	for i := uint(0); i < number; i++ {
		index := rand.Intn(len(t.accountsFlat))
		id := t.idGen.Next()
		res[i] = fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"method":"eth_call","params":[{"to":"0x%v","data":"0x%v"}, "0x%x"]}`, id, t.contractAddr, "70a08231000000000000000000000000"+t.accountsFlat[index], t.blk-1)
	}
	return res, nil
}

func (t *ERC20BalanceTracker) Status() string {
	return ""
}
