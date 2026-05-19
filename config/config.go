package config

import (
	"flag"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

type Config struct {
	WebAddr string
	MCPAddr string
	DataDir string
	Dev     bool
	NoMCP   bool
	Open    bool
}

func Load() *Config {
	cfg := &Config{}

	flag.StringVar(&cfg.WebAddr, "webAddr", ":3666", "Web UI + REST API address")
	flag.StringVar(&cfg.MCPAddr, "mcpAddr", ":3667", "MCP SSE server address")
	flag.BoolVar(&cfg.Dev, "dev", false, "Enable dev mode")
	flag.BoolVar(&cfg.NoMCP, "noMcp", false, "Disable MCP server")

	defaultOpen := runtime.GOOS == "windows"
	flag.BoolVar(&cfg.Open, "open", defaultOpen, "Open browser on start")

	defaultDataDir := defaultDataDir()
	flag.StringVar(&cfg.DataDir, "dataDir", defaultDataDir, "Central database directory")

	flag.Parse()

	return cfg
}

func HandleVersion() bool {
	for _, arg := range os.Args[1:] {
		if arg == "-version" || arg == "--version" {
			return true
		}
	}
	return false
}

func defaultDataDir() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = "."
	}
	return filepath.Join(configDir, "Waypoint")
}

func (c *Config) CentralDBPath() string {
	return filepath.Join(c.DataDir, "waypoint.db")
}

func (c *Config) ProjectsDir() string {
	return filepath.Join(c.DataDir, "projects")
}

func (c *Config) EnsureDataDir() error {
	return os.MkdirAll(c.DataDir, 0755)
}

func (c *Config) EnsureProjectsDir() error {
	return os.MkdirAll(c.ProjectsDir(), 0755)
}

func ResolveAddr(addr string) string {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return addr
	}
	for i := port; i < port+100; i++ {
		ln, err := net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(i)))
		if err == nil {
			ln.Close()
			return net.JoinHostPort(host, strconv.Itoa(i))
		}
	}
	return addr
}

func AddrPort(addr string) string {
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return ""
	}
	if strings.HasPrefix(port, "0") {
		return ""
	}
	return port
}
