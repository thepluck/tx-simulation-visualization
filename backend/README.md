# Tx Simulation Backend

Local Go server that accepts simulation parameters, maps `chain` to an RPC URL from config, compiles optional state override Solidity, runs the existing Foundry script with CLI arguments, and returns both the raw Forge trace and a structured trace tree.

## Run

```sh
cd backend
go run ./cmd/server
```

The server loads config from `TXSIM_CONFIG` when set. Otherwise it searches for `config.json`, `backend/config.json`, `config.example.json`, then `backend/config.example.json`.

Use `config.example.json` as the starting point. The backend loads `.env` from the repo root and `backend/.env` before expanding RPC values, so requests only need to pass a chain name.

`max_concurrent_runs` is a channel-backed limiter for Forge executions. Keep it at `1` for the safest local behavior, or raise it if your machine/RPC can handle parallel simulations.

## Endpoints

- `GET /docs`
- `GET /openapi.json`
- `GET /health`
- `GET /chains`
- `POST /simulate`

## Simulate Request

```json
{
  "chain": "mainnet",
  "blockNumber": "23000000",
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

`stateOverride` is optional. When provided, the backend writes the source into the per-request work directory, runs `forge inspect <file>:<contractName> bytecode`, passes that creation bytecode as a `run(...)` argument, then executes `contracts/SimulateTx.s.sol:SimulateTxScript` directly with `forge script`.

`chain` becomes `--rpc-url <configured-url>` and `blockNumber` becomes `--fork-block-number <block>`. They are not part of the Solidity script signature.

`compiler` is optional and maps to popular Forge compiler flags for both the state override `forge inspect` compile and the final `forge script` run. Supported fields are `use`, `offline`, `noAutoDetect`, `viaIR`, `useLiteralContent`, `noMetadata`, `evmVersion`, `optimize`, `optimizerRuns`, and `revertStrings`. `viaIR` and `optimize` default to `true`.
