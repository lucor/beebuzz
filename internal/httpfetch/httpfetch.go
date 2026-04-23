// Package httpfetch provides an SSRF-safe HTTP client for fetching external URLs.
package httpfetch

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"lucor.dev/beebuzz/internal/core"
)

const (
	fetchTimeout = 10 * time.Second
	schemeHTTPS  = "https"
)

// Fetch downloads the resource at rawURL and returns its body and content-type.
// It enforces HTTPS-only, rejects private/loopback/link-local IPs to prevent SSRF,
// and limits the response body to maxBytes.
func Fetch(ctx context.Context, rawURL string, maxBytes int64) (body []byte, contentType string, err error) {
	transport := &http.Transport{
		DialContext: ssrfSafeDialContext,
	}

	client := &http.Client{
		Transport:     transport,
		Timeout:       fetchTimeout,
		CheckRedirect: requireHTTPSRedirect,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("httpfetch: invalid request: %w", err)
	}

	if req.URL.Scheme != schemeHTTPS {
		return nil, "", fmt.Errorf("httpfetch: only HTTPS is allowed")
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("httpfetch: request failed: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode >= 400 {
		return nil, "", fmt.Errorf("httpfetch: server returned %d", resp.StatusCode)
	}

	limited := io.LimitReader(resp.Body, maxBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, "", fmt.Errorf("httpfetch: failed to read body: %w", err)
	}

	if int64(len(data)) > maxBytes {
		return nil, "", core.ErrPayloadTooLarge
	}

	return data, resp.Header.Get("Content-Type"), nil
}

// requireHTTPSRedirect rejects redirects to non-HTTPS URLs.
func requireHTTPSRedirect(req *http.Request, _ []*http.Request) error {
	if req.URL.Scheme == schemeHTTPS {
		return nil
	}

	return fmt.Errorf("httpfetch: redirect to non-HTTPS URL is not allowed")
}

// isDisallowedIP reports whether the resolved address should be rejected for outbound fetches.
func isDisallowedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}

	if ip.IsUnspecified() || ip.IsMulticast() {
		return true
	}

	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast()
}

// ssrfSafeDialContext resolves DNS and rejects private/loopback/link-local IPs.
func ssrfSafeDialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("httpfetch: invalid address %q: %w", addr, err)
	}

	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("httpfetch: DNS lookup failed for %q: %w", host, err)
	}

	for _, a := range addrs {
		ip := a.IP
		if isDisallowedIP(ip) {
			return nil, fmt.Errorf("httpfetch: IP %s is not allowed (non-public)", ip)
		}
	}

	if len(addrs) == 0 {
		return nil, fmt.Errorf("httpfetch: no addresses resolved for %q", host)
	}

	dialer := &net.Dialer{}
	return dialer.DialContext(ctx, network, net.JoinHostPort(addrs[0].IP.String(), port))
}
