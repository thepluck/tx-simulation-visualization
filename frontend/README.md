# Foundry Tx Simulator Frontend

```bash
yarn install
yarn dev
```

The app calls the backend at `http://127.0.0.1:8080` by default. Change the API URL field if the local server is running elsewhere.

Docker is available as an optional deployment path from the repo root:

```bash
docker compose up --build frontend
```

The Docker frontend is served at `http://127.0.0.1:5173` and still calls the backend at `http://127.0.0.1:8080` by default. The local `yarn dev` workflow is unchanged.

Override the Docker host port with `TXSIM_FRONTEND_PORT`:

```bash
TXSIM_FRONTEND_PORT=15173 docker compose up --build frontend
```

Override the local Vite dev server port with the same variable:

```bash
TXSIM_FRONTEND_PORT=15173 yarn dev
```
