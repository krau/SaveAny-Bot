package netutil

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/krau/SaveAny-Bot/config"
	"golang.org/x/net/proxy"
)

func NewProxyHTTPClient(proxyUrl string) (*http.Client, error) {
	if proxyUrl == "" {
		return http.DefaultClient, nil
	}
	transport, err := NewProxyTransport(proxyUrl)
	if err != nil {
		return nil, err
	}
	return &http.Client{
		Transport: transport,
	}, nil
}

var (
	defaultProxyHttpClient         *http.Client
	onceLoadDefaultProxyHttpClient sync.Once
)

func DefaultParserHTTPClient() *http.Client {
	onceLoadDefaultProxyHttpClient.Do(func() {
		client, err := NewProxyHTTPClient(config.C().Parser.Proxy)
		if err != nil {
			log.Warn("Failed to create default proxy HTTP client, using http.DefaultClient", "error", err)
			defaultProxyHttpClient = http.DefaultClient
		} else {
			defaultProxyHttpClient = client
		}
	})
	return defaultProxyHttpClient
}

func NewProxyTransport(proxyStr string) (*http.Transport, error) {
	proxyURL, err := url.Parse(proxyStr)
	if err != nil {
		return nil, err
	}
	transport := &http.Transport{
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	switch proxyURL.Scheme {
	case "http", "https":
		transport.Proxy = http.ProxyURL(proxyURL)

	case "socks5", "socks5h":
		dialer, err := proxy.FromURL(proxyURL, proxy.Direct)
		if err != nil {
			return nil, err
		}
		transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.(proxy.ContextDialer).DialContext(ctx, network, addr)
		}

	default:
		return nil, fmt.Errorf("unsupported proxy type: %s", proxyURL.Scheme)
	}

	return transport, nil
}
