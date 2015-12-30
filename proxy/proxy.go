package proxy

import (
	"bufio"
	"encoding/json"
	"errors"
	"github.com/gorilla/mux"
	"io"
	"log"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"../rpc"
)

type ProxyServer struct {
	config          *Config
	miners          MinersMap
	blockTemplate   atomic.Value
	upstream        int32
	upstreams       []*rpc.RPCClient
	hashrateWindow  time.Duration
	timeout         time.Duration
	roundShares     int64
	blocksMu        sync.RWMutex
	blockStats      map[int64]float64
	luckWindow      int64
	luckLargeWindow int64
}

type Session struct {
	enc *json.Encoder
	ip  string
}

const (
	MaxReqSize = 1 * 1024
)

func NewEndpoint(cfg *Config) *ProxyServer {
	proxy := &ProxyServer{config: cfg, blockStats: make(map[int64]float64)}

	proxy.upstreams = make([]*rpc.RPCClient, len(cfg.Upstream))
	for i, v := range cfg.Upstream {
		client, err := rpc.NewRPCClient(v.Name, v.Url, v.Timeout, v.Pool)
		if err != nil {
			log.Fatal(err)
		} else {
			proxy.upstreams[i] = client
			log.Printf("Upstream: %s => %s", v.Name, v.Url)
		}
	}
	log.Printf("Default upstream: %s => %s", proxy.rpc().Name, proxy.rpc().Url)

	proxy.miners = NewMinersMap()

	timeout, _ := time.ParseDuration(cfg.Proxy.ClientTimeout)
	proxy.timeout = timeout

	hashrateWindow, _ := time.ParseDuration(cfg.Proxy.HashrateWindow)
	proxy.hashrateWindow = hashrateWindow

	luckWindow, _ := time.ParseDuration(cfg.Proxy.LuckWindow)
	proxy.luckWindow = int64(luckWindow / time.Millisecond)
	luckLargeWindow, _ := time.ParseDuration(cfg.Proxy.LargeLuckWindow)
	proxy.luckLargeWindow = int64(luckLargeWindow / time.Millisecond)

	proxy.blockTemplate.Store(&BlockTemplate{})
	proxy.fetchBlockTemplate()

	refreshIntv, _ := time.ParseDuration(cfg.Proxy.BlockRefreshInterval)
	refreshTimer := time.NewTimer(refreshIntv)
	log.Printf("Set block refresh every %v", refreshIntv)

	checkIntv, _ := time.ParseDuration(cfg.UpstreamCheckInterval)
	checkTimer := time.NewTimer(checkIntv)

	go func() {
		for {
			select {
			case <-refreshTimer.C:
				proxy.fetchBlockTemplate()
				refreshTimer.Reset(refreshIntv)
			}
		}
	}()

	go func() {
		for {
			select {
			case <-checkTimer.C:
				proxy.checkUpstreams()
				checkTimer.Reset(checkIntv)
			}
		}
	}()

	return proxy
}

func (s *ProxyServer) rpc() *rpc.RPCClient {
	i := atomic.LoadInt32(&s.upstream)
	return s.upstreams[i]
}

func (s *ProxyServer) checkUpstreams() {
	candidate := int32(0)
	backup := false

	for i, v := range s.upstreams {
		ok, err := v.Check()
		if err != nil {
			log.Printf("Upstream %v didn't pass check: %v", v.Name, err)
		}
		if ok && !backup {
			candidate = int32(i)
			backup = true
		}
	}

	if s.upstream != candidate {
		log.Printf("Switching to %v upstream", s.upstreams[candidate].Name)
		atomic.StoreInt32(&s.upstream, candidate)
	}
}

func (s *ProxyServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		s.writeError(w, 405, "rpc: POST method required, received "+r.Method)
		return
	}
	s.handleClient(w, r)
}

func (s *ProxyServer) handleClient(w http.ResponseWriter, r *http.Request) error {
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	cs := &Session{ip: ip, enc: json.NewEncoder(w)}
	defer r.Body.Close()
	connbuff := bufio.NewReaderSize(r.Body, MaxReqSize)

	for {
		data, isPrefix, err := connbuff.ReadLine()
		if isPrefix {
			log.Printf("Socket flood detected")
			return errors.New("Socket flood")
		} else if err == io.EOF {
			break
		}

		if len(data) > 1 {
			var req JSONRpcReq
			err = json.Unmarshal(data, &req)
			if err != nil {
				log.Printf("Malformed request: %v", err)
				return err
			}
			cs.handleMessage(s, r, &req)
		}
	}
	return nil
}

func (cs *Session) handleMessage(s *ProxyServer, r *http.Request, req *JSONRpcReq) {
	if req.Id == nil {
		log.Println("Missing RPC id")
		r.Close = true
		return
	}

	vars := mux.Vars(r)

	// Handle RPC methods
	switch req.Method {
	case "eth_getWork":
		reply, errReply := s.handleGetWorkRPC(cs, vars["diff"], vars["id"])
		if errReply != nil {
			cs.sendError(req.Id, errReply)
			break
		}
		cs.sendResult(req.Id, &reply)
	case "eth_submitWork":
		var params []string
		err := json.Unmarshal(*req.Params, &params)
		if err != nil {
			log.Println("Unable to parse params")
			break
		}
		reply, errReply := s.handleSubmitRPC(cs, vars["diff"], vars["id"], params)
		if errReply != nil {
			err = cs.sendError(req.Id, errReply)
			break
		}
		cs.sendResult(req.Id, &reply)
	case "eth_submitHashrate":
		reply := true
		if s.config.Proxy.SubmitHashrate {
			reply = s.handleSubmitHashrate(cs, req)
		}
		cs.sendResult(req.Id, reply)
	default:
		errReply := s.handleUnknownRPC(cs, req)
		cs.sendError(req.Id, errReply)
	}
}

func (cs *Session) sendResult(id *json.RawMessage, result interface{}) error {
	message := JSONRpcResp{Id: id, Version: "2.0", Error: nil, Result: result}
	return cs.enc.Encode(&message)
}

func (cs *Session) sendError(id *json.RawMessage, reply *ErrorReply) error {
	message := JSONRpcResp{Id: id, Version: "2.0", Error: reply}
	return cs.enc.Encode(&message)
}

func (s *ProxyServer) writeError(w http.ResponseWriter, status int, msg string) {
	w.WriteHeader(status)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
}

func (s *ProxyServer) currentBlockTemplate() *BlockTemplate {
	return s.blockTemplate.Load().(*BlockTemplate)
}

func (s *ProxyServer) registerMiner(miner *Miner) {
	s.miners.Set(miner.Id, miner)
}
