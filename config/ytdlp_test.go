package config

import "testing"

func TestYtdlpConfigValidateProxies(t *testing.T) {
	cases := []struct {
		name    string
		proxies []string
		wantErr bool
	}{
		{name: "empty", proxies: nil},
		{name: "supported schemes", proxies: []string{"http://a:1", "https://a:1", "socks4://a:1", "socks4a://a:1", "socks5://u:p@a:1", "socks5h://a:1"}},
		{name: "blank entries ignored", proxies: []string{" ", ""}},
		{name: "missing scheme", proxies: []string{"127.0.0.1:1080"}, wantErr: true},
		{name: "unsupported scheme", proxies: []string{"ftp://a:1"}, wantErr: true},
		{name: "missing host", proxies: []string{"socks5://"}, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := YtdlpConfig{Proxies: tc.proxies}.ValidateProxies()
			if (err != nil) != tc.wantErr {
				t.Fatalf("ValidateProxies() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}
