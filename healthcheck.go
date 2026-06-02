package main

import (
	"fmt"
	"net/http"
	"os"
	"time"
)

const (
	healthcheckTimeout     = 5 * time.Second
	healthcheckDefaultPort = "8899"
)

func runHealthcheck() error {
	port := os.Getenv("BEEBUZZ_PORT")
	if port == "" {
		port = healthcheckDefaultPort
	}

	return runHealthcheckRequest("http://localhost:"+port+"/health", &http.Client{Timeout: healthcheckTimeout})
}

// runHealthcheckRequest keeps the liveness probe side-effect free so container
// health checks do not depend on full config validation or directory creation.
func runHealthcheckRequest(healthURL string, client *http.Client) error {
	resp, err := client.Get(healthURL)
	if err != nil {
		return fmt.Errorf("health check request failed: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed: status %d", resp.StatusCode)
	}

	return nil
}
