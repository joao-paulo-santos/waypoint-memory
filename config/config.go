package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	WebAddr     string
	MCPAddr     string
	DatabaseURL string
	NoMCP       bool
	Secret      string
	DataDir     string
}

func Load() *Config {
	cfg := &Config{}

	cfg.WebAddr = envOr("WAYPOINT_WEB_ADDR", ":7666")
	cfg.MCPAddr = envOr("WAYPOINT_MCP_ADDR", ":7667")
	cfg.DatabaseURL = envOr("WAYPOINT_DATABASE_URL", "")
	cfg.NoMCP = envBool("WAYPOINT_NO_MCP")
	cfg.Secret = envOr("WAYPOINT_SECRET", "")
	cfg.DataDir = envOr("WAYPOINT_DATA_DIR", "/data")

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

func (c *Config) Validate() error {
	if c.Secret == "" {
		return fmt.Errorf("WAYPOINT_SECRET is required")
	}
	if c.DatabaseURL == "" {
		return fmt.Errorf("WAYPOINT_DATABASE_URL is required")
	}
	return nil
}

func (c *Config) EnsureDataDir() error {
	if c.DataDir == "" {
		return nil
	}
	return os.MkdirAll(c.DataDir, 0755)
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

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envBool(key string) bool {
	v := os.Getenv(key)
	return strings.EqualFold(v, "true") || strings.EqualFold(v, "1") || strings.EqualFold(v, "yes")
}
