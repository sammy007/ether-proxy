package proxy

import (
	"../util"
	"log"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/ethash"
	"github.com/ethereum/go-ethereum/common"
)

var hasher = ethash.New()

type Miner struct {
	sync.RWMutex
	Id            string
	IP            string
	startedAt     int64
	lastBeat      int64
	validShares   uint64
	invalidShares uint64
	accepts       uint64
	rejects       uint64
	shares        map[int64]int64
}

func NewMiner(id, ip string) *Miner {
	miner := &Miner{Id: id, IP: ip, shares: make(map[int64]int64), startedAt: util.MakeTimestamp()}
	return miner
}

func (m *Miner) heartbeat() {
	now := util.MakeTimestamp()
	atomic.StoreInt64(&m.lastBeat, now)
}

func (m *Miner) getLastBeat() int64 {
	return atomic.LoadInt64(&m.lastBeat)
}

func (m *Miner) storeShare(diff int64) {
	now := util.MakeTimestamp()
	m.Lock()
	m.shares[now] += diff
	m.Unlock()
}

func (m *Miner) hashrate(hashrateWindow time.Duration) int64 {
	now := util.MakeTimestamp()
	totalShares := int64(0)
	window := int64(hashrateWindow / time.Millisecond)
	boundary := now - m.startedAt

	if boundary > window {
		boundary = window
	}

	m.Lock()
	for k, v := range m.shares {
		if k < now-86400000 {
			delete(m.shares, k)
		} else if k >= now-window {
			totalShares += v
		}
	}
	m.Unlock()
	return totalShares / boundary
}

func (m *Miner) processShare(s *ProxyServer, t *BlockTemplate, diff string, params []string) bool {
	paramsOrig := params[:]

	hashNoNonce := params[1]
	nonce, err := strconv.ParseUint(strings.Replace(params[0], "0x", "", -1), 16, 64)
	if err != nil {
		log.Printf("Malformed nonce: %v", err)
		return false
	}
	mixDigest := params[2]

	rpc := s.rpc()
	var shareDiff *big.Int

	if !rpc.Pool {
		minerDifficulty, err := strconv.ParseFloat(diff, 64)
		if err != nil {
			log.Println("Malformed difficulty: " + diff)
			minerDifficulty = 5
		}
		diff1 := int64(minerDifficulty * 1000000 * 100)
		shareDiff = big.NewInt(diff1)
	} else {
		shareDiff = util.TargetHexToDiff(t.Target)
	}

	share := Block{
		number:      t.Height,
		hashNoNonce: common.HexToHash(hashNoNonce),
		difficulty:  shareDiff,
		nonce:       nonce,
		mixDigest:   common.HexToHash(mixDigest),
	}

	block := Block{
		number:      t.Height,
		hashNoNonce: common.HexToHash(hashNoNonce),
		difficulty:  t.Difficulty,
		nonce:       nonce,
		mixDigest:   common.HexToHash(mixDigest),
	}

	if hasher.Verify(share) {
		m.heartbeat()
		m.storeShare(shareDiff.Int64())
		atomic.AddUint64(&m.validShares, 1)
		// Log round share for solo mode only
		if !rpc.Pool {
			atomic.AddInt64(&s.roundShares, shareDiff.Int64())
		}
		log.Printf("Valid share from %s@%s at difficulty %v", m.Id, m.IP, shareDiff)
	} else {
		atomic.AddUint64(&m.invalidShares, 1)
		log.Printf("Invalid share from %s@%s", m.Id, m.IP)
		return false
	}

	if rpc.Pool || hasher.Verify(block) {
		_, err = rpc.SubmitBlock(paramsOrig)
		now := util.MakeTimestamp()
		if err != nil {
			atomic.AddUint64(&m.rejects, 1)
			atomic.AddUint64(&rpc.Rejects, 1)
			log.Printf("Upstream submission failure on height %v: %v", t.Height, err)
		} else {
			if !rpc.Pool {
				// Solo block found, must refresh job
				s.fetchBlockTemplate()

				// Log this round variance
				roundShares := atomic.SwapInt64(&s.roundShares, 0)
				variance := float64(roundShares) / float64(t.Difficulty.Int64())
				s.blocksMu.Lock()
				s.blockStats[now] = variance
				s.blocksMu.Unlock()
			}
			atomic.AddUint64(&m.accepts, 1)
			atomic.AddUint64(&rpc.Accepts, 1)
			atomic.StoreInt64(&rpc.LastSubmissionAt, now)
			log.Printf("Upstream share found by miner %v@%v at height %d", m.Id, m.IP, t.Height)
		}
	}
	return true
}
