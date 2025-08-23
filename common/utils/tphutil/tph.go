package tphutil

import (
	"encoding/json"

	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/pkg/telegraph"
)

var tphClient *telegraph.Client

func DefaultClient() *telegraph.Client {
	if tphClient != nil {
		return tphClient
	}
	if config.C().Telegram.Proxy.Enable && config.C().Telegram.Proxy.URL != "" {
		proxyUrl := config.C().Telegram.Proxy.URL
		var err error
		tphClient, err = telegraph.NewClientWithProxy(proxyUrl)
		if err != nil {
			tphClient = telegraph.NewClient()
		}
	} else {
		tphClient = telegraph.NewClient()
	}
	return tphClient
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
			srcs = append(srcs, src)
		}
	}
	for _, child := range nodeElement.Children {
		srcs = append(srcs, GetNodeImages(child)...)
	}
	return srcs
}
