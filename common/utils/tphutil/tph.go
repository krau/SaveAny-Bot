package tphutil

import (
	"encoding/json"
	"strings"
	"sync"

	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/pkg/telegraph"
)

var (
	tphClient *telegraph.Client
	once      sync.Once
)

func DefaultClient() *telegraph.Client {
	once.Do(func() {
		tphClient = initDefault()
	})
	return tphClient
}

func initDefault() *telegraph.Client {
	var client *telegraph.Client
	if config.C().Telegram.Proxy.Enable && config.C().Telegram.Proxy.URL != "" {
		proxyUrl := config.C().Telegram.Proxy.URL
		var err error
		client, err = telegraph.NewClientWithProxy(proxyUrl)
		if err != nil {
			client = telegraph.NewClient()
		}
	} else {
		client = telegraph.NewClient()
	}
	return client
}

func GetNodeImages(node telegraph.Node) []string {
	var srcs []string

	var nodeElement telegraph.NodeElement
	data, err := json.Marshal(node)
	if err != nil {
		return srcs
	}
	err = json.Unmarshal(data, &nodeElement)
	if err != nil {
		return srcs
	}

	if nodeElement.Tag == "img" {
		if src, exists := nodeElement.Attrs["src"]; exists {
			if strings.HasPrefix(src, "/file/") {
				// handle images on telegra.ph server
				src = "https://telegra.ph" + src
			}
			srcs = append(srcs, src)
		}
	}
	for _, child := range nodeElement.Children {
		srcs = append(srcs, GetNodeImages(child)...)
	}
	return srcs
}
