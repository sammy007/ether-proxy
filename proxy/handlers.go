package proxy

import (
	"log"
	"math/big"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
)

var pow256 = common.BigPow(2, 256)

func (s *ProxyServer) handleGetWorkRPC(cs *Session, diff, id string) (reply []string, errorReply *ErrorReply) {
	t := s.currentBlockTemplate()
	minerDifficulty, err := strconv.ParseFloat(diff, 64)
	if err != nil {
		log.Printf("Invalid difficulty %v from %v@%v ", diff, id, cs.ip)
		minerDifficulty = 5
	}
	if len(t.Header) == 0 {
		return nil, &ErrorReply{Code: -1, Message: "Work not ready"}
	}
	minerAdjustedDifficulty := int64(minerDifficulty * 1000000 * 100)
	difficulty := big.NewInt(minerAdjustedDifficulty)
	diff1 := new(big.Int).Div(pow256, difficulty)
	diffBytes := string(common.ToHex(diff1.Bytes()))

	reply = []string{t.Header, t.Seed, diffBytes}
	return
}

func (s *ProxyServer) handleSubmitRPC(cs *Session, diff string, id string, params []string) (reply bool, errorReply *ErrorReply) {
	miner, ok := s.miners.Get(id)
	if !ok {
		miner = NewMiner(id, cs.ip)
		s.registerMiner(miner)
	}

	t := s.currentBlockTemplate()
	reply = miner.processShare(s, t, diff, params)
	return
}

func (s *ProxyServer) handleUnknownRPC(cs *Session, req *JSONRpcReq) *ErrorReply {
	log.Printf("Unknown RPC method: %v", req)
	return &ErrorReply{Code: -1, Message: "Invalid method"}
}
