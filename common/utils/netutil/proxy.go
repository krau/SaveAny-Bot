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

func NewProxyHTTPClient(proxyURL string) (*http.Client, error) {
	transport, err := NewProxyTransport(proxyURL)
	if err != nil {
		return nil, err
	}
	return &http.Client{Transport: transport}, nil
}

var defaultParserClient = struct {
	sync.Mutex
	key    string
	client *http.Client
}{}

// DefaultParserHTTPClient returns a parser client for the current parser proxy
// setting. The client is rebuilt when the setting changes so a settings reload
// does not leave requests pinned to the old proxy.
func DefaultParserHTTPClient() *http.Client {
	key := config.C().Parser.Proxy
	defaultParserClient.Lock()
	defer defaultParserClient.Unlock()
	if defaultParserClient.client != nil && defaultParserClient.key == key {
		return defaultParserClient.client
	}
	client, err := NewProxyHTTPClient(key)
	if err != nil {
		log.Warn("Failed to create default proxy HTTP client, using direct client", "error", err)
		fallback, fallbackErr := NewProxyHTTPClient("")
		if fallbackErr != nil {
			log.Errorf("Failed to create direct parser HTTP client: %v", fallbackErr)
			fallback = http.DefaultClient
		}
		client = fallback
	}
	if defaultParserClient.client != nil {
		if transport, ok := defaultParserClient.client.Transport.(*http.Transport); ok {
			transport.CloseIdleConnections()
		}
	}
	defaultParserClient.client = client
	defaultParserClient.key = key
	return client
}

func NewProxyTransport(proxyStr string) (*http.Transport, error) {
	transport := &http.Transport{
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
	}
	if proxyStr == "" {
		return transport, nil
	}
	proxyURL, err := url.Parse(proxyStr)
	if err != nil {
		return nil, err
	}
	switch proxyURL.Scheme {
	case "http", "https":
		transport.Proxy = http.ProxyURL(proxyURL)
	case "socks5", "socks5h":
		dialer, err := proxy.FromURL(proxyURL, proxy.Direct)
		if err != nil {
			return nil, err
		}
		contextDialer, ok := dialer.(proxy.ContextDialer)
		if !ok {
			return nil, fmt.Errorf("proxy %s does not support context cancellation", proxyURL.Scheme)
		}
		transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			return contextDialer.DialContext(ctx, network, addr)
		}
	default:
		return nil, fmt.Errorf("unsupported proxy type: %s", proxyURL.Scheme)
	}
	return transport, nil
}
