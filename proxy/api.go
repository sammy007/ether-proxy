package proxy

import (
	"encoding/json"
	"net/http"
	"sync/atomic"

	"../util"
)

func (s *ProxyServer) StatsIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	hashrate, totalOnline, miners := s.collectMinersStats()
	stats := map[string]interface{}{
		"miners":      miners,
		"hashrate":    hashrate,
		"totalMiners": len(miners),
		"totalOnline": totalOnline,
		"timedOut":    len(miners) - totalOnline,
	}

	var upstreams []interface{}
	current := atomic.LoadInt32(&s.upstream)

	for i, u := range s.upstreams {
		upstream := map[string]interface{}{
			"name":    u.Name,
			"url":     u.Url,
			"sick":    u.Sick(),
			"current": current == int32(i),
		}
		upstreams = append(upstreams, upstream)
	}
	stats["upstreams"] = upstreams
	stats["validBlocks"] = atomic.LoadUint64(&s.validBlocks)
	stats["invalidBlocks"] = atomic.LoadUint64(&s.invalidBlocks)
	stats["url"] = "http://" + s.config.Proxy.Listen + "/miner/<diff>/<id>"

	t := s.currentBlockTemplate()
	stats["height"] = t.Height
	stats["diff"] = t.Difficulty

	json.NewEncoder(w).Encode(stats)
}

func (s *ProxyServer) collectMinersStats() (int64, int, []interface{}) {
	now := util.MakeTimestamp()
	var result []interface{}
	totalHashrate := int64(0)
	totalOnline := 0

	for m := range s.miners.Iter() {
		stats := make(map[string]interface{})
		lastBeat := m.Val.getLastBeat()
		hashrate := m.Val.hashrate()
		totalHashrate += hashrate
		stats["name"] = m.Key
		stats["hashrate"] = hashrate
		stats["lastBeat"] = lastBeat
		stats["validShares"] = atomic.LoadUint64(&m.Val.validShares)
		stats["invalidShares"] = atomic.LoadUint64(&m.Val.invalidShares)
		stats["validBlocks"] = atomic.LoadUint64(&m.Val.validBlocks)
		stats["invalidBlocks"] = atomic.LoadUint64(&m.Val.invalidBlocks)
		stats["ip"] = m.Val.IP

		if now-lastBeat > (int64(s.timeout/2) / 1000000) {
			stats["warning"] = true
		}
		if now-lastBeat > (int64(s.timeout) / 1000000) {
			stats["timeout"] = true
		} else {
			totalOnline++
		}
		result = append(result, stats)
	}
	return totalHashrate, totalOnline, result
}
