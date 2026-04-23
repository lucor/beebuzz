package httpfetch

import (
	"net"
	"net/http"
	"testing"
)

func TestRequireHTTPSRedirectAllowsHTTPS(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "https://example.com/next", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	if err := requireHTTPSRedirect(req, nil); err != nil {
		t.Fatalf("requireHTTPSRedirect: got %v, want nil", err)
	}
}

func TestRequireHTTPSRedirectRejectsHTTP(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "http://example.com/next", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	if err := requireHTTPSRedirect(req, nil); err == nil {
		t.Fatal("requireHTTPSRedirect: got nil, want error")
	}
}

func TestIsDisallowedIP(t *testing.T) {
	tests := []struct {
		name string
		ip   string
		want bool
	}{
		{name: "public ipv4", ip: "8.8.8.8", want: false},
		{name: "public ipv6", ip: "2001:4860:4860::8888", want: false},
		{name: "unspecified ipv4", ip: "0.0.0.0", want: true},
		{name: "unspecified ipv6", ip: "::", want: true},
		{name: "loopback ipv4", ip: "127.0.0.1", want: true},
		{name: "loopback ipv6", ip: "::1", want: true},
		{name: "private ipv4", ip: "10.0.0.1", want: true},
		{name: "link local ipv4", ip: "169.254.1.1", want: true},
		{name: "multicast ipv4", ip: "224.0.0.1", want: true},
		{name: "multicast ipv6", ip: "ff02::1", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isDisallowedIP(net.ParseIP(tt.ip)); got != tt.want {
				t.Fatalf("isDisallowedIP(%q) = %v, want %v", tt.ip, got, tt.want)
			}
		})
	}
}
