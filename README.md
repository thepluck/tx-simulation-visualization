# Tx Simulation Visualization

Local transaction simulation and visualization tooling. The backend runs Foundry scripts and returns trace, fund-flow, and balance-analysis data; the frontend provides the local UI.

## Local Run

Run the backend directly:

```sh
cd backend
go run ./cmd/server
```

Run the frontend directly:

```sh
cd frontend
yarn install
yarn dev
```

The local frontend defaults to `http://127.0.0.1:8080` for the backend API.

## Docker Run

Docker is optional and does not replace local deployment.

```sh
docker compose up --build
```

Then open:

- Frontend: `http://127.0.0.1:5173`
- Backend: `http://127.0.0.1:8080`
- Swagger UI: `http://127.0.0.1:8080/docs`

The Compose stack reads `.env` for `MAINNET_RPC_URL`, `BASE_RPC_URL`, `ARBITRUM_RPC_URL`, and optional `COINGECKO_API_KEY`.

Override Docker host ports through `.env` or shell variables:

```sh
TXSIM_BACKEND_PORT=18080 TXSIM_FRONTEND_PORT=15173 docker compose up --build
```

For local deployment without Docker, use `TXSIM_LISTEN_ADDR` for the backend and `TXSIM_FRONTEND_PORT` for the Vite frontend:

```sh
(cd backend && TXSIM_LISTEN_ADDR=127.0.0.1:18080 go run ./cmd/server)
(cd frontend && TXSIM_FRONTEND_PORT=15173 yarn dev)
```

For external Foundry projects, Docker mounts the parent directory of this repo by default:

```text
.. -> /workspace/projects
```

That means a sibling project such as `/Users/lanhfff/Kyber/ks-dex-aggregator-sc` can still be entered in the UI as usual, and the backend will resolve it to `/workspace/projects/ks-dex-aggregator-sc` inside the container. If your projects live somewhere else, set:

```sh
TXSIM_PROJECTS_HOST_PATH=/Users/lanhfff/Kyber
TXSIM_PROJECTS_CONTAINER_PATH=/workspace/projects
```

The native folder picker remains a local macOS backend feature and is not available inside the Linux container, so Docker users should type or paste the project path.

## Foundry

**Foundry is a blazing fast, portable and modular toolkit for Ethereum application development written in Rust.**

Foundry consists of:

- **Forge**: Ethereum testing framework (like Truffle, Hardhat and DappTools).
- **Cast**: Swiss army knife for interacting with EVM smart contracts, sending transactions and getting chain data.
- **Anvil**: Local Ethereum node, akin to Ganache, Hardhat Network.
- **Chisel**: Fast, utilitarian, and verbose solidity REPL.

## Documentation

https://book.getfoundry.sh/

## Usage

### Build

```shell
$ forge build
```

### Test

```shell
$ forge test
```

### Format

```shell
$ forge fmt
```

### Gas Snapshots

```shell
$ forge snapshot
```

### Anvil

```shell
$ anvil
```

### Deploy

```shell
$ forge script script/Counter.s.sol:CounterScript --rpc-url <your_rpc_url> --private-key <your_private_key>
```

### Cast

```shell
$ cast <subcommand>
```

### Help

```shell
$ forge --help
$ anvil --help
$ cast --help
```
