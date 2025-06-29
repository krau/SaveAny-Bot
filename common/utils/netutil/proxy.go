package netutil

import (
	"net/url"

	"golang.org/x/net/proxy"
)

func NewProxyDialer(proxyUrl string) (proxy.Dialer, error) {
	url, err := url.Parse(proxyUrl)
	if err != nil {
		return nil, err
	}
	return proxy.FromURL(url, proxy.Direct)
}
