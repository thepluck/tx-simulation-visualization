#!/usr/bin/env python3
from __future__ import annotations

import argparse
import os
import shutil
import signal
import subprocess
import time
from pathlib import Path


ROOT_DIR = Path(__file__).resolve().parents[1]
DEFAULT_HOST = "127.0.0.1"
DEFAULT_BACKEND_PORT = 8080
DEFAULT_FRONTEND_PORT = 5173


def parse_port(value: str) -> int:
    try:
        port = int(value, 10)
    except ValueError as exc:
        raise argparse.ArgumentTypeError(f"{value!r} is not a valid port") from exc

    if port < 1 or port > 65535:
        raise argparse.ArgumentTypeError("port must be between 1 and 65535")

    return port


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Run the local Tx Simulation backend and frontend together.",
    )
    parser.add_argument(
        "--backend-port",
        type=parse_port,
        default=DEFAULT_BACKEND_PORT,
        help=f"backend port, default: {DEFAULT_BACKEND_PORT}",
    )
    parser.add_argument(
        "--frontend-port",
        type=parse_port,
        default=DEFAULT_FRONTEND_PORT,
        help=f"frontend port, default: {DEFAULT_FRONTEND_PORT}",
    )
    parser.add_argument(
        "--host",
        default=DEFAULT_HOST,
        help=f"local host for printed URLs and backend bind address, default: {DEFAULT_HOST}",
    )
    return parser.parse_args()


def require_command(command: str) -> None:
    if shutil.which(command) is None:
        raise SystemExit(f"{command} is required but was not found on PATH")


def frontend_command(host: str, port: int) -> list[str]:
    vite_bin = ROOT_DIR / "frontend" / "node_modules" / ".bin" / "vite"
    vite_args = ["--host", host, "--port", str(port)]

    if vite_bin.exists():
        return [str(vite_bin), *vite_args]

    require_command("yarn")
    return ["yarn", "dev", *vite_args]


def start_process(name: str, cwd: Path, command: list[str], env: dict[str, str]) -> subprocess.Popen:
    print(f"Starting {name}: {' '.join(command)}", flush=True)
    return subprocess.Popen(
        command,
        cwd=cwd,
        env=env,
        start_new_session=True,
    )


def stop_process(process: subprocess.Popen) -> None:
    if process.poll() is not None:
        return

    try:
        os.killpg(process.pid, signal.SIGTERM)
    except ProcessLookupError:
        return


def wait_for_exit(processes: dict[str, subprocess.Popen]) -> tuple[str, int]:
    while True:
        for name, process in processes.items():
            status = process.poll()
            if status is not None:
                return name, status

        time.sleep(0.25)


def main() -> int:
    args = parse_args()
    require_command("go")

    env = os.environ.copy()
    backend_addr = f"{args.host}:{args.backend_port}"
    backend_url = f"http://{backend_addr}"

    backend_env = env.copy()
    backend_env["TXSIM_LISTEN_ADDR"] = backend_addr

    frontend_env = env.copy()
    frontend_env["TXSIM_API_URL"] = backend_url

    processes: dict[str, subprocess.Popen] = {}

    try:
        processes["backend"] = start_process(
            "backend",
            ROOT_DIR / "backend",
            ["go", "run", "./cmd/server"],
            backend_env,
        )
        processes["frontend"] = start_process(
            "frontend",
            ROOT_DIR / "frontend",
            frontend_command(args.host, args.frontend_port),
            frontend_env,
        )

        print()
        print(f"Frontend: http://{args.host}:{args.frontend_port}")
        print(f"Backend:  {backend_url}")
        print(f"Swagger:  {backend_url}/docs")
        print()
        print("Press Ctrl-C to stop both servers.", flush=True)

        exited_name, status = wait_for_exit(processes)
        print(f"{exited_name} exited with status {status}; stopping both servers.", flush=True)
        return status
    except KeyboardInterrupt:
        print("\nStopping both servers.", flush=True)
        return 130
    finally:
        for process in processes.values():
            stop_process(process)

        for process in processes.values():
            try:
                process.wait(timeout=5)
            except subprocess.TimeoutExpired:
                try:
                    os.killpg(process.pid, signal.SIGKILL)
                except ProcessLookupError:
                    pass


if __name__ == "__main__":
    raise SystemExit(main())
