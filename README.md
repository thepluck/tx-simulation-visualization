# Tx Simulation Visualization

Local transaction simulation and visualization tooling. The backend runs Foundry scripts and returns trace, fund-flow, and balance-analysis data; the frontend provides the local UI.

## Local Run

Run the backend and frontend together:

```sh
python3 scripts/dev.py
```

Use custom local ports:

```sh
python3 scripts/dev.py --backend-port 18080 --frontend-port 15173
```

The script also points the frontend's default API URL at the selected backend port.

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

When run directly, the local frontend defaults to `http://127.0.0.1:8080` for the backend API.

## Configuration

Both local and Docker deployments can read `.env` values for `MAINNET_RPC_URL`, `BASE_RPC_URL`, `ARBITRUM_RPC_URL`, optional `ETHERSCAN_API_KEY`, and optional `COINGECKO_API_KEY`. Backend environment variables override YAML config values: use `TXSIM_` names for top-level backend settings and chain-specific names such as `MAINNET_RPC_URL` for RPC endpoints.

Use `TXSIM_LISTEN_ADDR` for the local backend bind address, `TXSIM_FRONTEND_PORT` for the Vite frontend port, and `TXSIM_API_URL` when the browser should call a specific backend URL.

For local deployment without Docker:

```sh
(cd backend && TXSIM_LISTEN_ADDR=127.0.0.1:18080 go run ./cmd/server)
(cd frontend && TXSIM_FRONTEND_PORT=15173 TXSIM_API_URL=http://127.0.0.1:18080 yarn dev)
```

Local deployment stores recently used Foundry project paths in `backend/.runs/projects.json` by default.

## Docker Run

Docker is optional and does not replace local deployment.

```sh
docker compose up --build
```

Then open:

- Frontend: `http://127.0.0.1:5173`
- Backend: `http://127.0.0.1:8080`
- Swagger UI: `http://127.0.0.1:8080/docs`

Docker stores recently used Foundry project paths in the `backend-runs` volume at `/data/runs/projects.json`, so project suggestions survive container rebuilds.

Override Docker host ports through `.env` or shell variables:

```sh
TXSIM_BACKEND_PORT=18080 TXSIM_FRONTEND_PORT=15173 docker compose up --build
```

The frontend container uses `TXSIM_BACKEND_PORT` to generate its browser runtime config, so the default API URL follows the published backend port. Set `TXSIM_API_URL` if the browser should call a different backend URL.

For external Foundry projects, Docker mounts the parent directory of this repo by default:

```text
.. -> /workspace/projects
```

That means a sibling project such as `~/Kyber/ks-dex-aggregator-sc` can still be entered in the UI as usual, and the backend will resolve it to `/workspace/projects/ks-dex-aggregator-sc` inside the container. If your projects live somewhere else, set:

```sh
TXSIM_PROJECTS_HOST_PATH=~/Kyber
TXSIM_PROJECTS_CONTAINER_PATH=/workspace/projects
```

The backend expands `~` in `projectPath` and `project_roots`. Docker Compose also resolves `~` in `TXSIM_PROJECTS_HOST_PATH`. The native folder picker remains a local macOS backend feature and is not available inside the Linux container, so Docker users should type or paste the project path.

## Foundry Contracts

The local simulation script and its Foundry project live in `contracts/`.

### Build

```shell
$ cd contracts
$ forge build
```

### Test

```shell
$ cd contracts
$ forge test
```

### Format

```shell
$ cd contracts
$ forge fmt
```

You can also run from the repo root by passing `--root contracts`:

```shell
$ forge build --root contracts
$ forge test --root contracts
```
