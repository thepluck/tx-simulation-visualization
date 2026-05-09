# Backend Agent Notes

## Scope

This folder contains the local Go HTTP server for running Foundry simulations. Keep backend behavior local-first: config maps chain names to RPC URLs, requests pass chain names, worker-owned Anvil forks are reset to the requested chain/block, and Forge runs against the worker Anvil RPC.

## Commands

- `go test ./...`
- `make lint`
- `gofmt -w <changed-go-files>`
- `docker compose config`

`make lint` runs `golangci-lint` through `go run`, so it does not require a globally installed binary.

## Conventions

- Keep packages split by responsibility under `internal/`.
- Do not generate Solidity scripts per request; pass arguments into the existing script contract.
- Do not fail the HTTP request just because `forge script` exits non-zero if there is a trace to return.
- Do not silently reuse an existing Anvil process on the configured port; occupied worker ports should fail clearly so users can adjust `anvil_port_start`.
- Preserve the compact response shape. Add fields only when the frontend or API contract needs them.
- Price-derived USD values must account for token decimals. If a source gives a price without decimals, merge it with RPC or another metadata source before calculating USD.
- Keep Etherscan API key handling backend-side and chain-independent; Etherscan API v2 uses one key across chains.
- If the simulation response schema changes, update the frontend Zod schema and types in `frontend/src/api/`.
- `docker compose config` should pass even when a local `.env` file is absent.
