package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const healthTimeout = 3 * time.Second

func runHealthCheck(addr string) error {
	client := &http.Client{Timeout: healthTimeout}

	resp, err := client.Get("http://" + healthTarget(addr) + "/healthz")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("healthz returned %d", resp.StatusCode)
	}
	return nil
}

func healthTarget(addr string) string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "127.0.0.1" + addr
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	return net.JoinHostPort(host, port)
}

func envAddr() string {
	if addr := os.Getenv("DW_ADDR"); addr != "" {
		return addr
	}
	return ":8111"
}

func useHostFilesystem(hostRoot string) error {
	if hostRoot == "" {
		return nil
	}
	for key, dir := range map[string]string{
		"HOST_PROC": "proc",
		"HOST_SYS":  "sys",
		"HOST_ETC":  "etc",
	} {
		if err := os.Setenv(key, filepath.Join(hostRoot, dir)); err != nil {
			return fmt.Errorf("set %s: %w", key, err)
		}
	}
	return nil
}
