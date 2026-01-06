package tgutil

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/url"

	"github.com/gotd/td/telegram/dcs"
	"github.com/krau/SaveAny-Bot/config"
	"golang.org/x/net/proxy"
)

// httpProxyDialer implements proxy.ContextDialer for HTTP CONNECT proxies
type httpProxyDialer struct {
	proxyURL *url.URL
	forward  proxy.Dialer
}

func (d *httpProxyDialer) Dial(network, addr string) (net.Conn, error) {
	return d.DialContext(context.Background(), network, addr)
}

func (d *httpProxyDialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	proxyAddr := d.proxyURL.Host
	if d.proxyURL.Port() == "" {
		if d.proxyURL.Scheme == "https" {
			proxyAddr = net.JoinHostPort(d.proxyURL.Hostname(), "443")
		} else {
			proxyAddr = net.JoinHostPort(d.proxyURL.Hostname(), "80")
		}
	}

	var conn net.Conn
	var err error
	if ctxDialer, ok := d.forward.(proxy.ContextDialer); ok {
		conn, err = ctxDialer.DialContext(ctx, "tcp", proxyAddr)
	} else {
		conn, err = d.forward.Dial("tcp", proxyAddr)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to connect to proxy: %w", err)
	}

	// Send CONNECT request
	connectReq := &http.Request{
		Method: "CONNECT",
		URL:    &url.URL{Opaque: addr},
		Host:   addr,
		Header: make(http.Header),
	}

	// Add proxy authentication if provided
	if d.proxyURL.User != nil {
		username := d.proxyURL.User.Username()
		password, _ := d.proxyURL.User.Password()
		auth := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
		connectReq.Header.Set("Proxy-Authorization", "Basic "+auth)
	}

	if err := connectReq.Write(conn); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to write CONNECT request: %w", err)
	}

	// Read response
	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, connectReq)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to read CONNECT response: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		conn.Close()
		return nil, fmt.Errorf("proxy CONNECT failed with status: %s", resp.Status)
	}

	return conn, nil
}

func newProxyDialer(proxyUrl string) (proxy.ContextDialer, error) {
	parsedURL, err := url.Parse(proxyUrl)
	if err != nil {
		return nil, err
	}

	switch parsedURL.Scheme {
	case "http", "https":
		return &httpProxyDialer{
			proxyURL: parsedURL,
			forward:  proxy.Direct,
		}, nil
	case "socks5", "socks5h":
		dialer, err := proxy.FromURL(parsedURL, proxy.Direct)
		if err != nil {
			return nil, err
		}
		return dialer.(proxy.ContextDialer), nil
	default:
		return nil, fmt.Errorf("unsupported proxy scheme: %s", parsedURL.Scheme)
	}
}

func NewConfigProxyResolver() (dcs.Resolver, error) {
	resolver := dcs.DefaultResolver()
	if config.C().Proxy != "" {
		// global proxy, which has lower priority
		dialer, err := newProxyDialer(config.C().Proxy)
		if err != nil {
			return nil, err
		}
		resolver = dcs.Plain(dcs.PlainOptions{
			Dial: dialer.DialContext,
		})
	}
	if config.C().Telegram.Proxy.Enable && config.C().Telegram.Proxy.URL != "" {
		dialer, err := newProxyDialer(config.C().Telegram.Proxy.URL)
		if err != nil {
			return nil, err
		}
		resolver = dcs.Plain(dcs.PlainOptions{
			Dial: dialer.DialContext,
		})
	}
	return resolver, nil
}
