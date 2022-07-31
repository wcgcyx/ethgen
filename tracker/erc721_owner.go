package tracker

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"math/rand"
	"strings"

	"github.com/wcgcyx/ethgen/idgen"
	"github.com/ethereum/go-ethereum/core/types"
)

type ERC721OwnerTracker struct {
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
	// State of the nft list
	nftAccessed []uint
	nftsFlat    []string
}

func NewERC721OwnerTracker(idGen idgen.IdGenerator, contractAddr string, maxBlocks uint) *ERC721OwnerTracker {
	return &ERC721OwnerTracker{
		idGen:        idGen,
		signer:       types.NewLondonSigner(big.NewInt(1)),
		contractAddr: strings.ToLower(contractAddr),
		maxBlocks:    maxBlocks,
		weight:       0,
		accessed:     make([]uint, maxBlocks),
		nftAccessed:  make([]uint, maxBlocks),
		nftsFlat:     make([]string, 0),
	}
}

func (t *ERC721OwnerTracker) ApplyBlock(blk *types.Block) error {
	accessed := uint(0)
	nftsToAdd := make([]string, 0)
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
			if method == "b88d4fde" || method == "42842e0e" || method == "23b872dd" {
				// transfer from
				nftId := hex.EncodeToString(tx.Data()[68:100])
				nftsToAdd = append(nftsToAdd, nftId)
				// Method accessed once
				accessed++
			}
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
	// Update accounts state
	// Pop
	pop2 := t.nftAccessed[t.maxBlocks-1]
	t.nftAccessed = t.nftAccessed[:t.maxBlocks-1]
	t.nftsFlat = t.nftsFlat[:len(t.nftsFlat)-int(pop2)]
	// Push
	t.nftAccessed = append([]uint{uint(len(nftsToAdd))}, t.nftAccessed...)
	t.nftsFlat = append(nftsToAdd, t.nftsFlat...)
	return nil
}

func (t *ERC721OwnerTracker) CurrentWeight() uint {
	return t.weight
}

func (t *ERC721OwnerTracker) GenerateQuery(number uint) ([]string, error) {
	if len(t.nftsFlat) == 0 {
		return nil, fmt.Errorf("empty nfts")
	}
	res := make([]string, number)
	for i := uint(0); i < number; i++ {
		index := rand.Intn(len(t.nftsFlat))
		id := t.idGen.Next()
		res[i] = fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"method":"eth_call","params":[{"to":"0x%v","data":"0x%v"}, "latest"]}`, id, t.contractAddr, "6352211e"+t.nftsFlat[index])
	}
	return res, nil
}

func (t *ERC721OwnerTracker) Status() string {
	return ""
}
