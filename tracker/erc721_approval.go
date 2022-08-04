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

type ERC721ApprovalTracker struct {
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
	nftsFlat    [][2]string
	// Current block
	blk uint64
}

func NewERC721ApprovalTracker(idGen idgen.IdGenerator, contractAddr string, maxBlocks uint) *ERC721ApprovalTracker {
	return &ERC721ApprovalTracker{
		idGen:        idGen,
		signer:       types.NewLondonSigner(big.NewInt(1)),
		contractAddr: strings.ToLower(contractAddr),
		maxBlocks:    maxBlocks,
		weight:       0,
		accessed:     make([]uint, maxBlocks),
		nftAccessed:  make([]uint, maxBlocks),
		nftsFlat:     make([][2]string, 0),
	}
}

func (t *ERC721ApprovalTracker) ApplyBlock(blk *types.Block) error {
	accessed := uint(0)
	nftsToAdd := make([][2]string, 0)
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
			if method == "a22cb465" {
				// set approval for all
				fromAddr, err := t.signer.Sender(tx)
				if err != nil {
					return err
				}
				owner := strings.ToLower(fromAddr.String()[2:])
				operator := hex.EncodeToString(tx.Data()[36:68])
				nftsToAdd = append(nftsToAdd, [][2]string{{owner, operator}}...)
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
	pop2 := t.nftAccessed[t.maxBlocks-1]
	t.nftAccessed = t.nftAccessed[:t.maxBlocks-1]
	t.nftsFlat = t.nftsFlat[:len(t.nftsFlat)-int(pop2)]
	// Push
	t.nftAccessed = append([]uint{uint(len(nftsToAdd))}, t.nftAccessed...)
	t.nftsFlat = append(nftsToAdd, t.nftsFlat...)
	return nil
}

func (t *ERC721ApprovalTracker) CurrentWeight() uint {
	return t.weight
}

func (t *ERC721ApprovalTracker) GenerateQuery(number uint) ([]string, error) {
	if len(t.nftsFlat) == 0 {
		return nil, fmt.Errorf("empty nfts")
	}
	res := make([]string, number)
	for i := uint(0); i < number; i++ {
		index := rand.Intn(len(t.nftsFlat))
		id := t.idGen.Next()
		res[i] = fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"method":"eth_call","params":[{"to":"0x%v","data":"0x%v"}, "0x%x"]}`, id, t.contractAddr, "e985e9c5"+"000000000000000000000000"+t.nftsFlat[index][0]+"000000000000000000000000"+t.nftsFlat[index][1], t.blk)
	}
	return res, nil
}

func (t *ERC721ApprovalTracker) Status() string {
	return ""
}
