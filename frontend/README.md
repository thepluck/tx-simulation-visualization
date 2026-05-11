# Foundry Tx Simulator Frontend

```bash
yarn install
yarn dev
```

The app calls the backend at `http://127.0.0.1:8080` by default. Set `TXSIM_API_URL` when the browser should call a different backend URL:

```bash
TXSIM_API_URL=http://127.0.0.1:18080 yarn dev
```

You can also change the API URL field in the app for one-off local testing.

Docker is available as an optional deployment path from the repo root:

```bash
docker compose up --build frontend
```

The Docker frontend is served at `http://127.0.0.1:5173` and still calls the backend at `http://127.0.0.1:8080` by default. The local `yarn dev` workflow is unchanged.

Override the Docker host port with `TXSIM_FRONTEND_PORT`:

```bash
TXSIM_FRONTEND_PORT=15173 docker compose up --build frontend
```

When using `./dev.sh`, set the local Vite dev server port with `frontend_port` in repo-root `config.yml`. When running `yarn dev` directly from this folder, override the port with:

```bash
TXSIM_FRONTEND_PORT=15173 yarn dev
```
