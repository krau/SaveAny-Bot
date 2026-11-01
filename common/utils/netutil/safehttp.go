package netutil

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

// PublicOnlyHTTPClient returns a client that refuses to connect to loopback,
// private, link-local, or unspecified addresses. Use it for URLs taken from
// untrusted content.
func PublicOnlyHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			DialContext:       publicDialContext,
			ForceAttemptHTTP2: true,
			MaxIdleConns:      10,
			IdleConnTimeout:   30 * time.Second,
		},
	}
}

func publicDialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	addresses, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("netutil: resolve %s: %w", host, err)
	}
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	var lastErr error
	for _, address := range addresses {
		if !isPublicIP(address.IP) {
			continue
		}
		conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(address.IP.String(), port))
		if err == nil {
			return conn, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return nil, fmt.Errorf("netutil: no reachable public address for %s: %w", host, lastErr)
	}
	return nil, fmt.Errorf("netutil: %s resolves only to non-public addresses", host)
}

func isPublicIP(ip net.IP) bool {
	return ip.IsGlobalUnicast() &&
		!ip.IsPrivate() &&
		!ip.IsLoopback() &&
		!ip.IsLinkLocalUnicast() &&
		!ip.IsLinkLocalMulticast() &&
		!ip.IsUnspecified()
}
