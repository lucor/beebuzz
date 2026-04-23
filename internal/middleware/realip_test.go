package middleware

import (
	"context"
	"net/http"
	"net/netip"
	"testing"
)

func TestRealIP_Resolve(t *testing.T) {
	proxySubnet := netip.MustParsePrefix("172.20.0.0/16")

	tests := []struct {
		name       string
		proxy      netip.Prefix
		remoteAddr string
		xff        string
		want       string
	}{
		{
			name:       "direct connection, no proxy configured",
			proxy:      netip.Prefix{},
			remoteAddr: "95.12.34.56:54321",
			want:       "95.12.34.56",
		},
		{
			name:       "direct connection, spoofed XFF ignored",
			proxy:      netip.Prefix{},
			remoteAddr: "95.12.34.56:54321",
			xff:        "1.2.3.4",
			want:       "95.12.34.56",
		},
		{
			name:       "proxy configured, peer in subnet, real client in XFF",
			proxy:      proxySubnet,
			remoteAddr: "172.20.0.2:12345",
			xff:        "95.12.34.56",
			want:       "95.12.34.56",
		},
		{
			name:       "proxy configured, peer in subnet, spoofed leftmost ignored",
			proxy:      proxySubnet,
			remoteAddr: "172.20.0.2:12345",
			xff:        "1.1.1.1, 95.12.34.56",
			want:       "95.12.34.56",
		},
		{
			name:       "proxy configured, peer outside subnet, XFF ignored",
			proxy:      proxySubnet,
			remoteAddr: "203.0.113.5:9999",
			xff:        "1.2.3.4",
			want:       "203.0.113.5",
		},
		{
			name:       "proxy configured, peer in subnet, no XFF, fallback to peer",
			proxy:      proxySubnet,
			remoteAddr: "172.20.0.2:12345",
			want:       "172.20.0.2",
		},
		{
			name:       "proxy configured, IPv6-mapped IPv4 in XFF",
			proxy:      proxySubnet,
			remoteAddr: "172.20.0.2:12345",
			xff:        "::ffff:95.12.34.56",
			want:       "95.12.34.56",
		},
		{
			name:       "proxy configured, invalid XFF, fallback to peer",
			proxy:      proxySubnet,
			remoteAddr: "172.20.0.2:12345",
			xff:        "garbage",
			want:       "172.20.0.2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rip := NewRealIP(tt.proxy)
			req := &http.Request{
				RemoteAddr: tt.remoteAddr,
				Header:     http.Header{},
			}
			if tt.xff != "" {
				req.Header.Set("X-Forwarded-For", tt.xff)
			}

			got := rip.Resolve(req)
			if got != tt.want {
				t.Errorf("Resolve() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRealIP_Resolve_MultipleXFFHeaders(t *testing.T) {
	rip := NewRealIP(netip.MustParsePrefix("172.20.0.0/16"))
	req := &http.Request{
		RemoteAddr: "172.20.0.2:12345",
		Header:     http.Header{},
	}
	req.Header.Add("X-Forwarded-For", "1.1.1.1")     // spoofed by attacker
	req.Header.Add("X-Forwarded-For", "95.12.34.56") // appended by proxy

	got := rip.Resolve(req)
	if got != "95.12.34.56" {
		t.Errorf("Resolve() = %q, want %q", got, "95.12.34.56")
	}
}

func TestClientIPFromContext(t *testing.T) {
	ctx := withClientIP(context.Background(), "95.12.34.56")
	ip, ok := ClientIPFromContext(ctx)
	if !ok {
		t.Error("expected ok to be true")
	}
	if ip != "95.12.34.56" {
		t.Errorf("ip = %q, want %q", ip, "95.12.34.56")
	}

	_, ok = ClientIPFromContext(context.Background())
	if ok {
		t.Error("expected ok to be false")
	}
}
