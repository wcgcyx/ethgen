# ethgen

A tool to generate realistic near-head read only eth_call queries.

# Build
make build

# Run
```
 ./build/ethgen start --window=64 --concurrency=5 --frequency=5ms --chain_ap=http://localhost:8545
```
You will also potentially need to do the following to release port faster:
```
sudo sysctl -w net.inet.ip.portrange.first=20000
sudo sysctl -w net.inet.ip.portrange.last=65535

sudo sysctl -w kern.maxfiles=1048576
sudo sysctl -w kern.maxfilesperproc=1048576
sudo sysctl -w kern.ipc.somaxconn=2048

sudo sysctl -w net.inet.tcp.msl=100
```