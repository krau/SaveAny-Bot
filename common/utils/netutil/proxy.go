package netutil

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sync"

	"github.com/charmbracelet/log"
	"github.com/krau/SaveAny-Bot/config"
	"golang.org/x/net/proxy"
)

func NewProxyDialer(proxyUrl string) (proxy.Dialer, error) {
	url, err := url.Parse(proxyUrl)
	if err != nil {
		return nil, err
	}
	return proxy.FromURL(url, proxy.Direct)
}

func NewProxyHTTPClient(proxyUrl string) (*http.Client, error) {
	if proxyUrl == "" {
		return &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
			},
		}, nil
	}

	u, err := url.Parse(proxyUrl)
	if err != nil {
		return nil, err
	}

	switch u.Scheme {
	case "http", "https":
		return &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyURL(u),
			},
		}, nil
	case "socks5":
		dialer, err := proxy.SOCKS5("tcp", u.Host, nil, proxy.Direct)
		if err != nil {
			return nil, err
		}

		return &http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					return dialer.Dial(network, addr)
				},
			},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported proxy scheme: %s", u.Scheme)
	}
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
