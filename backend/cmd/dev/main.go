package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	backendconfig "foundry-tx-simulator/backend/internal/config"
)

const (
	defaultHost = backendconfig.DefaultListenHost
)

var colors = map[string]string{
	"backend":  "\033[36m",
	"frontend": "\033[35m",
	"status":   "\033[2m",
	"reset":    "\033[0m",
}

var printMu sync.Mutex

type process struct {
	name string
	cmd  *exec.Cmd
}

type processResult struct {
	name   string
	status int
}

func main() {
	status, err := run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(status)
	}
	os.Exit(status)
}

func run() (int, error) {
	rootDir, err := repoRoot()
	if err != nil {
		return 1, err
	}

	if err := parseArgs(os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0, nil
		}
		return 2, err
	}
	if err := requireCommand("go"); err != nil {
		return 1, err
	}

	configPath, err := backendconfig.ResolveConfigPath(rootDir, os.Getenv("TXSIM_CONFIG"))
	if err != nil {
		return 1, err
	}
	cfg, configPath, err := backendconfig.LoadFile(configPath)
	if err != nil {
		return 1, err
	}
	backendHost, backendPort, err := parseListenAddr(cfg.ListenAddr)
	if err != nil {
		return 1, err
	}

	frontendHost := browserHost(backendHost)
	backendURL := fmt.Sprintf("http://%s:%d", browserHost(backendHost), backendPort)
	frontendPort := cfg.FrontendPort

	backendEnv := withEnv(os.Environ(), "TXSIM_CONFIG", configPath)
	frontendEnv := withEnv(os.Environ(), "TXSIM_API_URL", backendURL)
	frontendCmd, err := frontendCommand(rootDir, frontendHost, frontendPort)
	if err != nil {
		return 1, err
	}

	processes := []*process{}
	backend, err := startProcess("backend", filepath.Join(rootDir, "backend"), []string{"go", "run", "./cmd/server"}, backendEnv)
	if err != nil {
		return 1, err
	}
	processes = append(processes, backend)

	frontend, err := startProcess("frontend", filepath.Join(rootDir, "frontend"), frontendCmd, frontendEnv)
	if err != nil {
		stopProcesses(processes, syscall.SIGTERM)
		return 1, err
	}
	processes = append(processes, frontend)

	printStatus("")
	printStatus(fmt.Sprintf("Frontend: http://%s:%d", frontendHost, frontendPort))
	printStatus(fmt.Sprintf("Backend:  %s", backendURL))
	printStatus(fmt.Sprintf("Swagger:  %s/docs", backendURL))
	printStatus("")
	printStatus("Press Ctrl-C to stop both servers.")

	resultCh := make(chan processResult, len(processes))
	for _, proc := range processes {
		go waitProcess(proc, resultCh)
	}

	interruptCh := make(chan os.Signal, 1)
	signal.Notify(interruptCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(interruptCh)

	select {
	case result := <-resultCh:
		printStatus(fmt.Sprintf("%s exited with status %d; stopping both servers.", result.name, result.status))
		stopProcesses(processes, syscall.SIGTERM)
		waitForProcesses(processes, resultCh, map[string]bool{result.name: true})
		return result.status, nil
	case sig := <-interruptCh:
		printStatus(fmt.Sprintf("\nStopping both servers after %s.", sig))
		stopProcesses(processes, syscall.SIGTERM)
		waitForProcesses(processes, resultCh, nil)
		return signalExitStatus(sig), nil
	}
}

func parseArgs(values []string) error {
	flags := flag.NewFlagSet("dev", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	flags.Usage = func() {
		fmt.Fprintln(flags.Output(), "Usage: ./dev.sh")
		fmt.Fprintln(flags.Output(), "")
		fmt.Fprintln(flags.Output(), "Run the local Foundry Tx Simulator backend and frontend together.")
		fmt.Fprintln(flags.Output(), "")
		fmt.Fprintln(flags.Output(), "Configure backend and frontend ports in config.yml.")
	}
	if err := flags.Parse(values); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return err
		}
		return err
	}
	if flags.NArg() > 0 {
		return fmt.Errorf("unexpected arguments: %s", strings.Join(flags.Args(), " "))
	}
	return nil
}

func parsePort(value string) (int, error) {
	port, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%q is not a valid port", value)
	}
	if port < 1 || port > 65535 {
		return 0, fmt.Errorf("port must be between 1 and 65535")
	}
	return port, nil
}

func repoRoot() (string, error) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("resolve script path")
	}
	return filepath.Abs(filepath.Join(filepath.Dir(filename), "..", "..", ".."))
}

func requireCommand(command string) error {
	if _, err := exec.LookPath(command); err != nil {
		return fmt.Errorf("%s is required but was not found on PATH", command)
	}
	return nil
}

func useColor() bool {
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		return false
	}
	stat, err := os.Stdout.Stat()
	return err == nil && stat.Mode()&os.ModeCharDevice != 0
}

func colorize(value string, color string) string {
	if color == "" || !useColor() {
		return value
	}
	return color + value + colors["reset"]
}

func frontendCommand(rootDir string, host string, port int) ([]string, error) {
	viteBin := filepath.Join(rootDir, "frontend", "node_modules", ".bin", "vite")
	viteArgs := []string{"--host", host, "--port", strconv.Itoa(port)}
	if fileExists(viteBin) {
		return append([]string{viteBin}, viteArgs...), nil
	}
	if err := requireCommand("yarn"); err != nil {
		return nil, err
	}
	return append([]string{"yarn", "dev"}, viteArgs...), nil
}

func parseListenAddr(value string) (string, int, error) {
	value = strings.Trim(strings.TrimSpace(value), "\"'")
	if value == "" {
		return parseListenAddr(backendconfig.DefaultListenAddr)
	}
	if strings.HasPrefix(value, ":") {
		port, err := parsePort(value[1:])
		return defaultHost, port, err
	}
	if strings.HasPrefix(value, "[") && strings.Contains(value, "]:") {
		parts := strings.SplitN(value[1:], "]:", 2)
		port, err := parsePort(parts[1])
		return parts[0], port, err
	}
	if index := strings.LastIndex(value, ":"); index >= 0 {
		host := value[:index]
		portValue := value[index+1:]
		port, err := parsePort(portValue)
		if host == "" {
			host = defaultHost
		}
		return host, port, err
	}
	return value, backendconfig.DefaultBackendPort, nil
}

func browserHost(host string) string {
	switch host {
	case "", "0.0.0.0", "::", "[::]":
		return defaultHost
	default:
		return strings.Trim(host, "[]")
	}
}

func startProcess(name string, cwd string, command []string, env []string) (*process, error) {
	printStatus(fmt.Sprintf("Starting %s: %s", name, strings.Join(command, " ")))
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Dir = cwd
	cmd.Env = env
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	go streamOutput(name, stdout)
	go streamOutput(name, stderr)
	return &process{name: name, cmd: cmd}, nil
}

func printStatus(message string) {
	printMu.Lock()
	defer printMu.Unlock()
	fmt.Println(colorize(message, colors["status"]))
}

func streamOutput(name string, reader io.Reader) {
	prefix := colorize("["+name+"]", colors[name])
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		printMu.Lock()
		fmt.Printf("%s %s\n", prefix, scanner.Text())
		printMu.Unlock()
	}
}

func waitProcess(proc *process, resultCh chan<- processResult) {
	err := proc.cmd.Wait()
	resultCh <- processResult{name: proc.name, status: normalizeExitStatus(err)}
}

func stopProcesses(processes []*process, signal syscall.Signal) {
	for _, proc := range processes {
		if proc.cmd.Process == nil {
			continue
		}
		_ = syscall.Kill(-proc.cmd.Process.Pid, signal)
	}
}

func waitForProcesses(processes []*process, resultCh <-chan processResult, received map[string]bool) {
	if received == nil {
		received = map[string]bool{}
	}
	deadline := time.After(5 * time.Second)
	for len(received) < len(processes) {
		select {
		case result := <-resultCh:
			received[result.name] = true
		case <-deadline:
			stopProcesses(processes, syscall.SIGKILL)
			for len(received) < len(processes) {
				result := <-resultCh
				received[result.name] = true
			}
			return
		}
	}
}

func normalizeExitStatus(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return 1
	}
	if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
		if status.Signaled() {
			return 128 + int(status.Signal())
		}
		return status.ExitStatus()
	}
	return exitErr.ExitCode()
}

func signalExitStatus(sig os.Signal) int {
	if systemSignal, ok := sig.(syscall.Signal); ok {
		return 128 + int(systemSignal)
	}
	return 130
}

func withEnv(env []string, key string, value string) []string {
	prefix := key + "="
	out := make([]string, 0, len(env)+1)
	for _, item := range env {
		if strings.HasPrefix(item, prefix) {
			continue
		}
		out = append(out, item)
	}
	return append(out, prefix+value)
}

func fileExists(path string) bool {
	stat, err := os.Stat(path)
	return err == nil && !stat.IsDir()
}
