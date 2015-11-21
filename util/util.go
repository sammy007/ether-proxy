package util

import (
	"math/big"
	"math/rand"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

var pow256 = common.BigPow(2, 256)

func Random() string {
	min := int64(100000000000000)
	max := int64(999999999999999)
	n := rand.Int63n(max-min+1) + min
	return strconv.FormatInt(n, 10)
}

func MakeTimestamp() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

func MakeTargetHex(minerDifficulty float64) string {
	minerAdjustedDifficulty := int64(minerDifficulty * 1000000 * 100)
	difficulty := big.NewInt(minerAdjustedDifficulty)
	diff1 := new(big.Int).Div(pow256, difficulty)
	return string(common.ToHex(diff1.Bytes()))
}

func TargetHexToDiff(targetHex string) *big.Int {
	targetBytes := common.FromHex(targetHex)
	return new(big.Int).Div(pow256, common.BytesToBig(targetBytes))
}
