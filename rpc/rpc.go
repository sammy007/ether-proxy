package rpc

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type RPCClient struct {
	sync.RWMutex
	Url         string
	Name        string
	sick        bool
	sickRate    int
	successRate int
	client      *http.Client
	height      uint64
	diff        *big.Int
}

type GetBlockReply struct {
	Number           string   `json:"number"`
	Hash             string   `json:"hash"`
	ParentHash       string   `json:"parentHash"`
	Nonce            string   `json:"nonce"`
	Sha3Uncles       string   `json:"sha3Uncles"`
	LogsBloom        string   `json:"logsBloom"`
	TransactionsRoot string   `json:"transactionsRoot"`
	StateRoot        string   `json:"stateRoot"`
	Miner            string   `json:"miner"`
	Difficulty       string   `json:"difficulty"`
	TotalDifficulty  string   `json:"totalDifficulty"`
	Size             string   `json:"size"`
	ExtraData        string   `json:"extraData"`
	GasLimit         string   `json:"gasLimit"`
	GasUsed          string   `json:"gasUsed"`
	Timestamp        string   `json:"timestamp"`
	Transactions     []string `json:"transactions"`
	Uncles           []string `json:"uncles"`
}

type JSONRpcResp struct {
	Id     *json.RawMessage       `json:"id"`
	Result *json.RawMessage       `json:"result"`
	Error  map[string]interface{} `json:"error"`
}

func NewRPCClient(name, url, timeout string) *RPCClient {
	rpcClient := &RPCClient{Name: name, Url: url}
	timeoutIntv, _ := time.ParseDuration(timeout)
	rpcClient.client = &http.Client{
		Timeout: timeoutIntv,
	}
	return rpcClient
}

func (r *RPCClient) GetWork() ([]string, error) {
	params := []string{}

	rpcResp, err := r.doPost(r.Url, "eth_getWork", params)
	var reply []string
	if err != nil {
		return reply, err
	}
	if rpcResp.Error != nil {
		return reply, errors.New(string(rpcResp.Error["message"].(string)))
	}

	err = json.Unmarshal(*rpcResp.Result, &reply)
	// Handle empty result, daemon is catching up (geth bug!!!)
	if len(reply) != 3 || len(reply[0]) == 0 {
		return reply, errors.New("Daemon is not ready")
	}
	return reply, err
}

func (r *RPCClient) getPendingBlock() (GetBlockReply, error) {
	params := []interface{}{"pending", false}

	rpcResp, err := r.doPost(r.Url, "eth_getBlockByNumber", params)
	var reply GetBlockReply
	if err != nil {
		return reply, err
	}
	if rpcResp.Error != nil {
		return reply, errors.New(string(rpcResp.Error["message"].(string)))
	}

	err = json.Unmarshal(*rpcResp.Result, &reply)
	return reply, err
}

func (r *RPCClient) FetchPendingBlock() (uint64, *big.Int, error) {
	reply, err := r.getPendingBlock()
	if err != nil {
		return 0, nil, err
	}
	blockNumber, err := strconv.ParseUint(strings.Replace(reply.Number, "0x", "", -1), 16, 64)
	if err != nil {
		return 0, nil, err
	}
	blockDiff, err := strconv.ParseInt(strings.Replace(reply.Difficulty, "0x", "", -1), 16, 64)
	if err != nil {
		return 0, nil, err
	}

	r.Lock()
	defer r.Unlock()
	r.height = blockNumber
	r.diff = big.NewInt(blockDiff)

	return r.height, r.diff, nil
}

func (r *RPCClient) SubmitBlock(params []string) (bool, error) {
	rpcResp, err := r.doPost(r.Url, "eth_submitWork", params)
	var result bool
	if err != nil {
		return false, err
	}
	err = json.Unmarshal(*rpcResp.Result, &result)
	if !result {
		return false, errors.New("Block not accepted, result=false")
	}
	return result, nil
}

func (r *RPCClient) doPost(url string, method string, params interface{}) (JSONRpcResp, error) {
	jsonReq := map[string]interface{}{"jsonrpc": "2.0", "id": "0", "method": method, "params": params}
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

func (r *RPCClient) Check() bool {
	_, err := r.GetWork()
	if err != nil {
		return false
	}
	_, _, err = r.FetchPendingBlock()
	if err != nil {
		return false
	}
	r.markAlive()
	return !r.Sick()
}

func (r *RPCClient) Sick() bool {
	r.RLock()
	defer r.RUnlock()
	return r.sick
}

func (r *RPCClient) markSick() {
	r.Lock()
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
