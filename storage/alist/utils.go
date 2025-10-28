package alist

import (
	"net/http"
	"time"
)

func getHttpClient() *http.Client {
	return &http.Client{
		Timeout: 12 * time.Hour,
		Transport: &http.Transport{
			TLSHandshakeTimeout: 10 * time.Second,
		},
	}
}
