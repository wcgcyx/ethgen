# ethgen

A tool to generate realistic near-head read only eth_call queries.

# Build
make build

# Run
First run a daemon to follow the chain and build a adaptive dataset.
```
./build/ethgen daemon --config=./contracts.json --chain_ap=http://127.0.0.1:8545
```
To generate 250 queries every 1 second:
```
./build/ethgen generate --number=250 --duration=1s
```