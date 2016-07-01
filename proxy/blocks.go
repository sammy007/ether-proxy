package proxy

import (
	"../util"
	"log"
	"math/big"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

const maxBacklog = 8

type heightDiffPair struct {
	diff   *big.Int
	height uint64
}

type BlockTemplate struct {
	Header     string
	Seed       string
	Target     string
	Difficulty *big.Int
	Height     uint64
	headers    map[string]heightDiffPair
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
	// No need to update, we have fresh job
	if t != nil && t.Header == reply[0] {
		return
	}
	height, diff, err := s.fetchPendingBlock()
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
		headers:    make(map[string]heightDiffPair),
	}
	// Copy headers backlog and add current one
	newTemplate.headers[reply[0]] = heightDiffPair{diff: util.TargetHexToDiff(reply[2]), height: height}
	if t != nil {
		for k, v := range t.headers {
			if v.height > height-maxBacklog {
				newTemplate.headers[k] = v
			}
		}
	}
	s.blockTemplate.Store(&newTemplate)
	log.Printf("New block to mine on %s at height %d / %s", rpc.Name, height, reply[0][0:10])
}

func (s *ProxyServer) fetchPendingBlock() (uint64, *big.Int, error) {
	rpc := s.rpc()
	reply, err := rpc.GetPendingBlock()
	if err != nil {
		return 0, nil, err
	}
	blockNumber, err := strconv.ParseUint(strings.Replace(reply.Number, "0x", "", -1), 16, 64)
	if err != nil {
		log.Println("Can't parse pending block number")
		return 0, nil, err
	}
	blockDiff, err := strconv.ParseInt(strings.Replace(reply.Difficulty, "0x", "", -1), 16, 64)
	if err != nil {
		log.Println("Can't parse pending block difficulty")
		return 0, nil, err
	}

	return blockNumber, big.NewInt(blockDiff), nil
}
