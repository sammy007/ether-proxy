package rpc

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

type RPCClient struct {
	sync.RWMutex
	Url              *url.URL
	Name             string
	Pool             bool
	sick             bool
	sickRate         int
	successRate      int
	Accepts          uint64
	Rejects          uint64
	LastSubmissionAt int64
	client           *http.Client
	FailsCount       uint64
}

type GetBlockReply struct {
	Number     string `json:"number"`
	Difficulty string `json:"difficulty"`
}

type JSONRpcResp struct {
	Id     *json.RawMessage       `json:"id"`
	Result *json.RawMessage       `json:"result"`
	Error  map[string]interface{} `json:"error"`
}

func NewRPCClient(name, rawUrl, timeout string, pool bool) (*RPCClient, error) {
	url, err := url.Parse(rawUrl)
	if err != nil {
		return nil, err
	}
	rpcClient := &RPCClient{Name: name, Url: url, Pool: pool}
	timeoutIntv, _ := time.ParseDuration(timeout)
	rpcClient.client = &http.Client{
		Timeout: timeoutIntv,
	}
	return rpcClient, nil
}

func (r *RPCClient) GetWork() ([]string, error) {
	params := []string{}

	rpcResp, err := r.doPost(r.Url.String(), "eth_getWork", params)
	var reply []string
	if err != nil {
		return reply, err
	}
	if rpcResp.Error != nil {
		return reply, errors.New(rpcResp.Error["message"].(string))
	}
	err = json.Unmarshal(*rpcResp.Result, &reply)
	return reply, err
}

func (r *RPCClient) GetPendingBlock() (GetBlockReply, error) {
	params := []interface{}{"pending", false}

	rpcResp, err := r.doPost(r.Url.String(), "eth_getBlockByNumber", params)
	var reply GetBlockReply
	if err != nil {
		return reply, err
	}
	if rpcResp.Error != nil {
		return reply, errors.New(rpcResp.Error["message"].(string))
	}
	err = json.Unmarshal(*rpcResp.Result, &reply)
	return reply, err
}

func (r *RPCClient) SubmitBlock(params []string) (bool, error) {
	rpcResp, err := r.doPost(r.Url.String(), "eth_submitWork", params)
	var result bool
	if err != nil {
		return false, err
	}
	if rpcResp.Error != nil {
		return false, errors.New(rpcResp.Error["message"].(string))
	}
	err = json.Unmarshal(*rpcResp.Result, &result)
	if !result {
		return false, errors.New("Block not accepted, result=false")
	}
	return result, nil
}

func (r *RPCClient) SubmitHashrate(params interface{}) (bool, error) {
	rpcResp, err := r.doPost(r.Url.String(), "eth_submitHashrate", params)
	var result bool
	if err != nil {
		return false, err
	}
	if rpcResp.Error != nil {
		return false, errors.New(rpcResp.Error["message"].(string))
	}
	err = json.Unmarshal(*rpcResp.Result, &result)
	if !result {
		return false, errors.New("Request failure")
	}
	return result, nil
}

func (r *RPCClient) doPost(url, method string, params interface{}) (JSONRpcResp, error) {
	jsonReq := map[string]interface{}{"jsonrpc": "2.0", "id": 0, "method": method, "params": params}
	data, _ := json.Marshal(jsonReq)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	req.Header.Set("Content-Length", (string)(len(data)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := r.client.Do(req)
	var rpcResp JSONRpcResp

	if err != nil {
		r.markSick()
		return rpcResp, err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	err = json.Unmarshal(body, &rpcResp)

	if rpcResp.Error != nil {
		r.markSick()
	}
	return rpcResp, err
}

func (r *RPCClient) Check() (bool, error) {
	_, err := r.GetWork()
	if err != nil {
		return false, err
	}
	r.markAlive()
	return !r.Sick(), nil
}

func (r *RPCClient) Sick() bool {
	r.RLock()
	defer r.RUnlock()
	return r.sick
}

func (r *RPCClient) markSick() {
	r.Lock()
	if !r.sick {
		atomic.AddUint64(&r.FailsCount, 1)
	}
	r.sickRate++
	r.successRate = 0
	if r.sickRate >= 5 {
		r.sick = true
	}
	r.Unlock()
}

func (r *RPCClient) markAlive() {
	r.Lock()
	r.successRate++
	if r.successRate >= 5 {
		r.sick = false
		r.sickRate = 0
		r.successRate = 0
	}
	r.Unlock()
}
