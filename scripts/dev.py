#!/usr/bin/env python3
from __future__ import annotations

import argparse
import os
import shutil
import signal
import subprocess
import sys
import threading
import time
from pathlib import Path


ROOT_DIR = Path(__file__).resolve().parents[1]
DEFAULT_HOST = "127.0.0.1"
DEFAULT_BACKEND_PORT = 8080
DEFAULT_FRONTEND_PORT = 5173
CONFIG_CANDIDATES = [
    "backend/config.yml",
    "backend/config.yaml",
    "backend/config.example.yaml",
    "backend/config.example.yml",
    "config.yml",
    "config.yaml",
    "config.example.yaml",
    "config.example.yml",
]
COLORS = {
    "backend": "\033[36m",
    "frontend": "\033[35m",
    "status": "\033[2m",
    "reset": "\033[0m",
}
print_lock = threading.Lock()


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
        description="Run the local Foundry Tx Simulator backend and frontend together.",
    )
    parser.add_argument(
        "--backend-port",
        type=parse_port,
        default=None,
        help="override backend port from backend/config.yml",
    )
    parser.add_argument(
        "--frontend-port",
        type=parse_port,
        default=DEFAULT_FRONTEND_PORT,
        help=f"frontend port, default: {DEFAULT_FRONTEND_PORT}",
    )
    parser.add_argument(
        "--host",
        default=None,
        help=f"override backend bind host from backend/config.yml, default frontend host: {DEFAULT_HOST}",
    )
    return parser.parse_args()


def require_command(command: str) -> None:
    if shutil.which(command) is None:
        raise SystemExit(f"{command} is required but was not found on PATH")


def use_color() -> bool:
    return sys.stdout.isatty() and "NO_COLOR" not in os.environ


def colorize(value: str, color: str) -> str:
    if not color or not use_color():
        return value
    return f"{color}{value}{COLORS['reset']}"


def frontend_command(host: str, port: int) -> list[str]:
    vite_bin = ROOT_DIR / "frontend" / "node_modules" / ".bin" / "vite"
    vite_args = ["--host", host, "--port", str(port)]

    if vite_bin.exists():
        return [str(vite_bin), *vite_args]

    require_command("yarn")
    return ["yarn", "dev", *vite_args]


def resolve_backend_config(env: dict[str, str]) -> Path:
    configured = env.get("TXSIM_CONFIG", "").strip()
    if configured:
        candidate = Path(configured).expanduser()
        if not candidate.is_absolute():
            candidate = ROOT_DIR / candidate
        if candidate.exists():
            return candidate
        raise SystemExit(f"TXSIM_CONFIG points to missing config: {candidate}")

    for relative_path in CONFIG_CANDIDATES:
        candidate = ROOT_DIR / relative_path
        if candidate.exists():
            return candidate
    raise SystemExit("backend config is required; create backend/config.yml or set TXSIM_CONFIG")


def read_listen_addr(config_path: Path) -> str:
    for line in config_path.read_text(encoding="utf-8").splitlines():
        stripped = line.strip()
        if not line.startswith((" ", "\t")) and stripped.startswith("listen_addr:"):
            return stripped.split(":", 1)[1].strip().strip("\"'")
    return f"{DEFAULT_HOST}:{DEFAULT_BACKEND_PORT}"


def parse_listen_addr(value: str) -> tuple[str, int]:
    value = value.strip().strip("\"'")
    if not value:
        return DEFAULT_HOST, DEFAULT_BACKEND_PORT
    if value.startswith(":"):
        return DEFAULT_HOST, parse_port(value[1:])
    if value.startswith("[") and "]:" in value:
        host, port = value.rsplit("]:", 1)
        return host.lstrip("["), parse_port(port)
    if ":" in value:
        host, port = value.rsplit(":", 1)
        return host or DEFAULT_HOST, parse_port(port)
    return value, DEFAULT_BACKEND_PORT


def format_listen_addr(host: str, port: int) -> str:
    if ":" in host and not host.startswith("["):
        return f"[{host}]:{port}"
    return f"{host}:{port}"


def browser_host(host: str) -> str:
    if host in {"", "0.0.0.0", "::", "[::]"}:
        return DEFAULT_HOST
    return host.strip("[]")


def write_dev_backend_config(config_path: Path, listen_addr: str) -> Path:
    lines = config_path.read_text(encoding="utf-8").splitlines()
    out: list[str] = []
    replaced = False

    for line in lines:
        stripped = line.lstrip()
        is_top_level = len(stripped) == len(line)
        if is_top_level and stripped.startswith("listen_addr:"):
            out.append(f'listen_addr: "{listen_addr}"')
            replaced = True
        else:
            out.append(line)

    if not replaced:
        out.insert(0, f'listen_addr: "{listen_addr}"')

    dev_config = config_path.parent / f".dev-config-{os.getpid()}.yml"
    dev_config.write_text("\n".join(out) + "\n", encoding="utf-8")
    return dev_config


def start_process(name: str, cwd: Path, command: list[str], env: dict[str, str]) -> subprocess.Popen:
    print_status(f"Starting {name}: {' '.join(command)}")
    return subprocess.Popen(
        command,
        cwd=cwd,
        env=env,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        text=True,
        bufsize=1,
        start_new_session=True,
    )


def print_status(message: str) -> None:
    with print_lock:
        print(colorize(message, COLORS["status"]), flush=True)


def stream_output(name: str, process: subprocess.Popen) -> None:
    prefix = colorize(f"[{name}]", COLORS.get(name, ""))
    if process.stdout is None:
        return

    for line in process.stdout:
        with print_lock:
            print(f"{prefix} {line}", end="", flush=True)


def start_output_thread(name: str, process: subprocess.Popen) -> threading.Thread:
    thread = threading.Thread(target=stream_output, args=(name, process), daemon=True)
    thread.start()
    return thread


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


def normalize_exit_status(status: int) -> int:
    if status < 0:
        return 128 + abs(status)
    return status


def main() -> int:
    args = parse_args()
    require_command("go")

    env = os.environ.copy()
    dev_config_path: Path | None = None
    base_config_path = resolve_backend_config(env)
    config_host, config_port = parse_listen_addr(read_listen_addr(base_config_path))
    backend_host = args.host or config_host
    backend_port = args.backend_port or config_port
    frontend_host = args.host or DEFAULT_HOST
    backend_addr = format_listen_addr(backend_host, backend_port)
    backend_url = f"http://{browser_host(backend_host)}:{backend_port}"
    should_override_config = args.host is not None or args.backend_port is not None

    backend_env = env.copy()
    if should_override_config:
        dev_config_path = write_dev_backend_config(base_config_path, backend_addr)
        backend_env["TXSIM_CONFIG"] = str(dev_config_path)
    else:
        backend_env["TXSIM_CONFIG"] = str(base_config_path)

    frontend_env = env.copy()
    frontend_env["TXSIM_API_URL"] = backend_url

    processes: dict[str, subprocess.Popen] = {}
    output_threads: list[threading.Thread] = []

    try:
        processes["backend"] = start_process(
            "backend",
            ROOT_DIR / "backend",
            ["go", "run", "./cmd/server"],
            backend_env,
        )
        output_threads.append(start_output_thread("backend", processes["backend"]))

        processes["frontend"] = start_process(
            "frontend",
            ROOT_DIR / "frontend",
            frontend_command(frontend_host, args.frontend_port),
            frontend_env,
        )
        output_threads.append(start_output_thread("frontend", processes["frontend"]))

        print_status("")
        print_status(f"Frontend: http://{frontend_host}:{args.frontend_port}")
        print_status(f"Backend:  {backend_url}")
        print_status(f"Swagger:  {backend_url}/docs")
        print_status("")
        print_status("Press Ctrl-C to stop both servers.")

        exited_name, status = wait_for_exit(processes)
        exit_status = normalize_exit_status(status)
        print_status(f"{exited_name} exited with status {exit_status}; stopping both servers.")
        return exit_status
    except KeyboardInterrupt:
        print_status("\nStopping both servers.")
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

        for thread in output_threads:
            thread.join(timeout=1)

        if dev_config_path is not None:
            try:
                dev_config_path.unlink()
            except FileNotFoundError:
                pass


if __name__ == "__main__":
    raise SystemExit(main())
