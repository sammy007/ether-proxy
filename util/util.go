package util

import (
	"math/rand"
	"strconv"
	"time"
)

func Random() string {
	min := int64(100000000000000)
	max := int64(999999999999999)
	n := rand.Int63n(max-min+1) + min
	return strconv.FormatInt(n, 10)
}

func MakeTimestamp() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}
