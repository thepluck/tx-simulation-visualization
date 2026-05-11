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

	"github.com/spf13/viper"
)

const (
	defaultHost         = "127.0.0.1"
	defaultBackendPort  = 8080
	defaultFrontendPort = 5173
)

var configCandidates = []string{
	"backend/config.yml",
	"backend/config.yaml",
	"backend/config.example.yaml",
	"backend/config.example.yml",
	"config.yml",
	"config.yaml",
	"config.example.yaml",
	"config.example.yml",
}

var colors = map[string]string{
	"backend":  "\033[36m",
	"frontend": "\033[35m",
	"status":   "\033[2m",
	"reset":    "\033[0m",
}

var printMu sync.Mutex

type optionalPort struct {
	value int
	set   bool
}

func (p *optionalPort) Set(value string) error {
	port, err := parsePort(value)
	if err != nil {
		return err
	}
	p.value = port
	p.set = true
	return nil
}

func (p *optionalPort) String() string {
	if !p.set {
		return ""
	}
	return strconv.Itoa(p.value)
}

type requiredPort int

func (p *requiredPort) Set(value string) error {
	port, err := parsePort(value)
	if err != nil {
		return err
	}
	*p = requiredPort(port)
	return nil
}

func (p requiredPort) String() string {
	return strconv.Itoa(int(p))
}

type devArgs struct {
	backendPort  optionalPort
	frontendPort requiredPort
	host         string
}

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

	args, err := parseArgs(os.Args[1:])
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0, nil
		}
		return 2, err
	}
	if err := requireCommand("go"); err != nil {
		return 1, err
	}

	baseConfigPath, err := resolveBackendConfig(rootDir, os.Getenv("TXSIM_CONFIG"))
	if err != nil {
		return 1, err
	}
	listenAddr, err := readListenAddr(baseConfigPath)
	if err != nil {
		return 1, err
	}
	configHost, configPort, err := parseListenAddr(listenAddr)
	if err != nil {
		return 1, err
	}

	backendHost := configHost
	if strings.TrimSpace(args.host) != "" {
		backendHost = strings.TrimSpace(args.host)
	}
	backendPort := configPort
	if args.backendPort.set {
		backendPort = args.backendPort.value
	}
	frontendHost := defaultHost
	if strings.TrimSpace(args.host) != "" {
		frontendHost = strings.TrimSpace(args.host)
	}

	backendAddr := formatListenAddr(backendHost, backendPort)
	backendURL := fmt.Sprintf("http://%s:%d", browserHost(backendHost), backendPort)
	frontendPort := int(args.frontendPort)

	var devConfigPath string
	backendEnv := os.Environ()
	if args.backendPort.set || strings.TrimSpace(args.host) != "" {
		devConfigPath, err = writeDevBackendConfig(baseConfigPath, backendAddr)
		if err != nil {
			return 1, err
		}
		backendEnv = withEnv(backendEnv, "TXSIM_CONFIG", devConfigPath)
	} else {
		backendEnv = withEnv(backendEnv, "TXSIM_CONFIG", baseConfigPath)
	}
	defer func() {
		if devConfigPath != "" {
			_ = os.Remove(devConfigPath)
		}
	}()

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
	signal.Notify(interruptCh, os.Interrupt)
	defer signal.Stop(interruptCh)

	select {
	case result := <-resultCh:
		printStatus(fmt.Sprintf("%s exited with status %d; stopping both servers.", result.name, result.status))
		stopProcesses(processes, syscall.SIGTERM)
		waitForProcesses(processes, resultCh, map[string]bool{result.name: true})
		return result.status, nil
	case <-interruptCh:
		printStatus("\nStopping both servers.")
		stopProcesses(processes, syscall.SIGTERM)
		waitForProcesses(processes, resultCh, nil)
		return 130, nil
	}
}

func parseArgs(values []string) (devArgs, error) {
	args := devArgs{frontendPort: requiredPort(defaultFrontendPort)}
	flags := flag.NewFlagSet("dev", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	flags.Usage = func() {
		fmt.Fprintln(flags.Output(), "Usage: ./dev.sh [options]")
		fmt.Fprintln(flags.Output(), "")
		fmt.Fprintln(flags.Output(), "Run the local Foundry Tx Simulator backend and frontend together.")
		fmt.Fprintln(flags.Output(), "")
		fmt.Fprintln(flags.Output(), "Options:")
		flags.PrintDefaults()
	}
	flags.Var(&args.backendPort, "backend-port", "override backend port from backend/config.yml")
	flags.Var(&args.frontendPort, "frontend-port", fmt.Sprintf("frontend port, default: %d", defaultFrontendPort))
	flags.StringVar(&args.host, "host", "", fmt.Sprintf("override backend bind host from backend/config.yml, default frontend host: %s", defaultHost))
	if err := flags.Parse(values); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return args, err
		}
		return args, err
	}
	return args, nil
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

func resolveBackendConfig(rootDir string, configured string) (string, error) {
	configured = strings.TrimSpace(configured)
	if configured != "" {
		candidate, err := expandHome(configured)
		if err != nil {
			return "", err
		}
		if !filepath.IsAbs(candidate) {
			candidate = filepath.Join(rootDir, candidate)
		}
		if fileExists(candidate) {
			return candidate, nil
		}
		return "", fmt.Errorf("TXSIM_CONFIG points to missing config: %s", candidate)
	}

	for _, relativePath := range configCandidates {
		candidate := filepath.Join(rootDir, relativePath)
		if fileExists(candidate) {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("backend config is required; create backend/config.yml or set TXSIM_CONFIG")
}

func readListenAddr(configPath string) (string, error) {
	config := newDevConfigViper(configPath)
	if err := config.ReadInConfig(); err != nil {
		return "", err
	}
	return strings.TrimSpace(config.GetString("listen_addr")), nil
}

func parseListenAddr(value string) (string, int, error) {
	value = strings.Trim(strings.TrimSpace(value), "\"'")
	if value == "" {
		return defaultHost, defaultBackendPort, nil
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
	return value, defaultBackendPort, nil
}

func formatListenAddr(host string, port int) string {
	if strings.Contains(host, ":") && !strings.HasPrefix(host, "[") {
		return fmt.Sprintf("[%s]:%d", host, port)
	}
	return fmt.Sprintf("%s:%d", host, port)
}

func browserHost(host string) string {
	switch host {
	case "", "0.0.0.0", "::", "[::]":
		return defaultHost
	default:
		return strings.Trim(host, "[]")
	}
}

func writeDevBackendConfig(configPath string, listenAddr string) (string, error) {
	config := newDevConfigViper(configPath)
	if err := config.ReadInConfig(); err != nil {
		return "", err
	}
	config.Set("listen_addr", listenAddr)

	path := filepath.Join(filepath.Dir(configPath), fmt.Sprintf(".dev-config-%d.yml", os.Getpid()))
	if err := config.WriteConfigAs(path); err != nil {
		return "", err
	}
	return path, nil
}

func newDevConfigViper(configPath string) *viper.Viper {
	config := viper.New()
	config.SetConfigFile(configPath)
	config.SetDefault("listen_addr", fmt.Sprintf("%s:%d", defaultHost, defaultBackendPort))
	return config
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

func expandHome(path string) (string, error) {
	if path != "~" && !strings.HasPrefix(path, "~/") {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if path == "~" {
		return home, nil
	}
	return filepath.Join(home, strings.TrimPrefix(path, "~/")), nil
}
