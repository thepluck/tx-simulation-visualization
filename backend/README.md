# Tx Simulation Backend

Local Go server that accepts simulation parameters, maps `chain` to an RPC URL from config, compiles optional state override Solidity, runs the existing Foundry script with CLI arguments, and returns both the raw Forge trace and a structured trace tree.

## Run

```sh
cd backend
go run ./cmd/server
```

To log HTTP request and response bodies while debugging:

```sh
cd backend
TXSIM_DEBUG_HTTP=1 go run ./cmd/server
```

Docker is available as an optional deployment path from the repo root:

```sh
docker compose up --build backend
```

The Docker backend listens on `0.0.0.0:8080` inside the container and is published to `http://127.0.0.1:8080` by `docker-compose.yml`. Local `go run` deployment is unchanged.

Override the Docker host port with `TXSIM_BACKEND_PORT`:

```sh
TXSIM_BACKEND_PORT=18080 docker compose up --build backend
```

Override the local backend listen address with `TXSIM_LISTEN_ADDR`:

```sh
TXSIM_LISTEN_ADDR=127.0.0.1:18080 go run ./cmd/server
```

The server loads config from `TXSIM_CONFIG` when set. Otherwise it searches for `config.yaml`, `backend/config.yaml`, `config.yml`, `backend/config.yml`, then example YAML files.

Use `config.example.yaml` as the starting point. The backend loads `.env` from the repo root and `backend/.env` before expanding RPC values, so requests only need to pass a chain name. `explorer_urls` maps the same chain names to block explorer base URLs for frontend address links. Set `COINGECKO_API_KEY` in `.env` if you want CoinGecko requests to include a demo API key.

`max_concurrent_runs` is a channel-backed limiter for Forge executions. Keep it at `1` for the safest local behavior, or raise it if your machine/RPC can handle parallel simulations.

## Endpoints

- `GET /docs`
- `GET /openapi.json`
- `GET /health`
- `GET /chains`
- `GET /browse/project`
- `POST /simulate`

`GET /browse/project` opens a native local folder picker and returns the selected project path. It is intended for the local frontend's Foundry Project browse button.

Inside Docker, native project browsing is unavailable because the backend runs in a Linux container. Type or paste `projectPath` manually. The Compose stack mounts `TXSIM_PROJECTS_HOST_PATH` to `TXSIM_PROJECTS_CONTAINER_PATH`, and the backend can resolve host-style paths against that mounted project root.

## Simulate Request

```json
{
  "chain": "mainnet",
  "blockNumber": "23000000",
  "projectPath": "/Users/me/foundry-project",
  "labelOverrides": [
    {
      "account": "0x0000000000000000000000000000000000000000",
      "label": "ExampleAccount"
    }
  ],
  "erc20BalanceOverrides": [
    {
      "token": "0x0000000000000000000000000000000000000000",
      "account": "0x0000000000000000000000000000000000000000",
      "balance": "1000000000000000000"
    }
  ],
  "erc20ApprovalOverrides": [
    {
      "token": "0x0000000000000000000000000000000000000000",
      "owner": "0x0000000000000000000000000000000000000000",
      "spender": "0x0000000000000000000000000000000000000000",
      "amount": "1000000000000000000"
    }
  ],
  "erc721ApprovalOverrides": [
    {
      "token": "0x0000000000000000000000000000000000000000",
      "owner": "0x0000000000000000000000000000000000000000",
      "spender": "0x0000000000000000000000000000000000000000",
      "tokenId": "1"
    }
  ],
  "stateOverride": {
    "contractName": "MyStateOverride",
    "source": "// SPDX-License-Identifier: UNLICENSED\npragma solidity ^0.8.30;\ncontract MyStateOverride { fallback() external {} }"
  },
  "compiler": {
    "use": "0.8.30",
    "viaIR": true,
    "optimize": true,
    "optimizerRuns": 200,
    "evmVersion": "cancun",
    "revertStrings": "default"
  },
  "sender": "0x0000000000000000000000000000000000000000",
  "target": "0x0000000000000000000000000000000000000000",
  "data": "0x"
}
```

`blockNumber`, balances, approvals, and token IDs should be strings when they may exceed JavaScript's safe integer range. Hex strings are accepted for uint fields.

`stateOverride` is optional. When provided, the backend writes the source into the per-request work directory for local runs, or into a temporary file under `<projectPath>/script/` for external-project runs. It then runs `forge inspect <file>:<contractName> bytecode`, passes that creation bytecode as a `run(...)` argument, and executes the simulation script with `forge script`.

`chain` becomes `--rpc-url <configured-url>` and `blockNumber` becomes `--fork-block-number <block>`. They are not part of the Solidity script signature.

`projectPath` is optional. When provided, the backend treats it as another Foundry project, runs `forge build src --root <projectPath>`, copies `contracts/SimulateTx.s.sol` into a temporary file under `<projectPath>/script/`, runs `forge script` against that copied script with `--root <projectPath>`, then removes the temporary script file. Relative paths are resolved against the backend repo root.

`compiler` is optional and maps to popular Forge compiler flags. Supported fields are `use`, `offline`, `noAutoDetect`, `viaIR`, `useLiteralContent`, `noMetadata`, `evmVersion`, `optimize`, `optimizerRuns`, and `revertStrings`. The state override `forge inspect` compile and final `forge script` run default `viaIR` and `optimize` to `true`; external-project `forge build src` uses the target project's defaults unless compiler fields are explicitly set.

The response includes `erc20Transfers`, parsed from ERC20-style `Transfer(from, to, value)` trace events for later fund flow graph construction. Each item contains `token`, `from`, `to`, raw `amount`, and, when metadata is available, `normalizedAmount`, `symbol`, and `logoUrl`.

The response also includes `balanceAnalysis`, which aggregates ERC20 transfers into signed per-user token balance changes. It fetches token decimals and symbols from the configured chain RPC, then merges current USD prices and metadata from DefiLlama, CoinGecko, and DexScreener. Trust Wallet token logo URLs are used as a fallback when the token address can be checksummed. USD values are only calculated when both a price and token decimals are available.
