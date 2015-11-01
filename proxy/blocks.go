package proxy

import (
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type BlockTemplate struct {
	Header     string
	Seed       string
	Target     string
	Difficulty *big.Int
	Height     uint64
}

type Block struct {
	difficulty  *big.Int
	hashNoNonce common.Hash
	nonce       uint64
	mixDigest   common.Hash
	number      uint64
}

func (b Block) Difficulty() *big.Int     { return b.difficulty }
func (b Block) HashNoNonce() common.Hash { return b.hashNoNonce }
func (b Block) Nonce() uint64            { return b.nonce }
func (b Block) MixDigest() common.Hash   { return b.mixDigest }
func (b Block) NumberU64() uint64        { return b.number }

func (s *ProxyServer) fetchBlockTemplate() {
	rpc := s.rpc()
	reply, err := rpc.GetWork()
	if err != nil {
		log.Printf("Error while refreshing block template on %s: %s", rpc.Name, err)
		return
	}

	t := s.currentBlockTemplate()

	height, diff, err := rpc.FetchPendingBlock()
	if err != nil {
		log.Printf("Error while refreshing pending block on %s: %s", rpc.Name, err)
		return
	}
	newTemplate := BlockTemplate{
		Header:     reply[0],
		Seed:       reply[1],
		Target:     reply[2],
		Height:     height,
		Difficulty: diff,
	}
	s.blockTemplate.Store(&newTemplate)

	if height != t.Height {
		log.Printf("New block to mine on %s at height: %d", rpc.Name, height)
	}
	return
}
