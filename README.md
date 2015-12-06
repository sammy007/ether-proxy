# ether-proxy

Ethereum mining proxy with web-interface.

**Proxy feature list:**

* Rigs availability monitoring
* Keep track of accepts, rejects, blocks stats
* Easy detection of sick rigs
* Daemon failover list

![Demo](https://raw.githubusercontent.com/sammy007/ether-proxy/master/proxy.png)

### Building on Linux

Dependencies:

  * go >= 1.4
  * geth

Export GOPATH:

    export GOPATH=$HOME/go

Install required packages:

    go get github.com/ethereum/ethash
    go get github.com/ethereum/go-ethereum/common
    go get github.com/goji/httpauth
    go get github.com/gorilla/mux
    go get github.com/yvasiyarov/gorelic

Compile:

    go build -o ether-proxy main.go

### Building on Windows

Follow [this wiki paragraph](https://github.com/ethereum/go-ethereum/wiki/Installation-instructions-for-Windows#building-from-source) in order to prepare your environment.
Install required packages (look at Linux install guide above). Then compile:

    go build -o ether-proxy.exe main.go

### Building on Mac OS X

If you didn't install [Brew](http://brew.sh/), do it. Then install Golang:

    brew install go

And follow Linux installation instructions because they are the same for OS X.

### Configuration

Configuration is self-describing, just copy *config.example.json* to *config.json* and specify endpoint URL and upstream URLs.

#### Example upstream section

```javascript
"upstream": [
  {
    "pool": true,
    "name": "EuroHash.net",
    "url": "http://eth-eu.eurohash.net:8888/miner/0xb85150eb365e7df0941f0cf08235f987ba91506a/proxy",
    "timeout": "10s"
  },
  {
    "name": "backup-geth",
    "url": "http://127.0.0.1:8545",
    "timeout": "10s"
  }
],
```

In this example we specified [EuroHash.net](https://eurohash.net) mining pool as main mining target and a local geth node as backup for solo.

With <code>"submitHashrate": true|false</code> proxy will forward <code>eth_submitHashrate</code> requests to upstream.

#### Running

    ./ether-proxy config.json

#### Mining

    ethminer -F http://x.x.x.x:8546/miner/5/gpu-rig -G
    ethminer -F http://x.x.x.x:8546/miner/0.1/cpu-rig -C

### Pools that work with this proxy

* [EuroHash.net](https://eurohash.net) EU Ethereum mining pool

Pool owners, apply for listing here. PM me for implementation details.

### TODO

**Currently it's solo-only solution.**

* Report block numbers
* Report average luck
* Report luck per rig
* Maybe add more stats
* Maybe add charts

### Donations

* **ETH**: [0xb85150eb365e7df0941f0cf08235f987ba91506a](https://etherchain.org/account/0xb85150eb365e7df0941f0cf08235f987ba91506a)

* **BTC**: [1PYqZATFuYAKS65dbzrGhkrvoN9au7WBj8](https://blockchain.info/address/1PYqZATFuYAKS65dbzrGhkrvoN9au7WBj8)

Thanks to a couple of dudes who donated some Ether to me, I believe, you can do the same.

### License

The MIT License (MIT).
