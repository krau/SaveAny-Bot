package tgutil

import (
	"net/url"

	"github.com/gotd/td/telegram/dcs"
	"github.com/krau/SaveAny-Bot/config"
	"golang.org/x/net/proxy"
)

func newProxyDialer(proxyUrl string) (proxy.Dialer, error) {
	url, err := url.Parse(proxyUrl)
	if err != nil {
		return nil, err
	}
	return proxy.FromURL(url, proxy.Direct)
}

func NewConfigProxyResolver() (dcs.Resolver, error) {
	resolver := dcs.DefaultResolver()
	if config.C().Proxy != "" {
		// gloabl proxy, which has lower priority
		dialer, err := newProxyDialer(config.C().Proxy)
		if err != nil {
			return nil, err
		}
		resolver = dcs.Plain(dcs.PlainOptions{
			Dial: dialer.(proxy.ContextDialer).DialContext,
		})
	}
	if config.C().Telegram.Proxy.Enable && config.C().Telegram.Proxy.URL != "" {
		dialer, err := newProxyDialer(config.C().Telegram.Proxy.URL)
		if err != nil {
			return nil, err
		}
		resolver = dcs.Plain(dcs.PlainOptions{
			Dial: dialer.(proxy.ContextDialer).DialContext,
		})
	}
	return resolver, nil
}
