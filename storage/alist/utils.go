package alist

import (
	"net/http"
	"time"
)

var (
	httpClient *http.Client
)

func getHttpClient() *http.Client {
	if httpClient != nil {
		return httpClient
	}
	httpClient = &http.Client{
		Timeout: 12 * time.Hour,
		Transport: &http.Transport{
			TLSHandshakeTimeout: 10 * time.Second,
		},
	}
	return httpClient
}
