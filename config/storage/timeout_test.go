package storage

import "testing"

func TestUploadResponseHeaderTimeoutJSON(t *testing.T) {
	cfg, err := NewStorageConfigFromJSON(1, "webdav", "webdav", true, []byte(`{
		"url": "http://example.com",
		"username": "user",
		"password": "password",
		"base_path": "/",
		"upload_response_header_timeout": 42
	}`))
	if err != nil {
		t.Fatal(err)
	}
	webdavConfig, ok := cfg.(*WebdavStorageConfig)
	if !ok {
		t.Fatalf("config type = %T, want *WebdavStorageConfig", cfg)
	}
	if webdavConfig.UploadResponseHeaderTimeout != 42 {
		t.Fatalf("UploadResponseHeaderTimeout = %d, want 42", webdavConfig.UploadResponseHeaderTimeout)
	}
}

func TestUploadResponseHeaderTimeoutValidation(t *testing.T) {
	tests := []struct {
		name   string
		config StorageConfig
	}{
		{
			name: "webdav",
			config: &WebdavStorageConfig{
				URL:                         "http://example.com",
				Username:                    "user",
				Password:                    "password",
				BasePath:                    "/",
				UploadResponseHeaderTimeout: -1,
			},
		},
		{
			name: "alist",
			config: &AlistStorageConfig{
				URL:                         "http://example.com",
				Username:                    "user",
				Password:                    "password",
				BasePath:                    "/",
				UploadResponseHeaderTimeout: -1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.config.Validate(); err == nil {
				t.Fatal("Validate() succeeded for a negative upload response header timeout")
			}
		})
	}
}
