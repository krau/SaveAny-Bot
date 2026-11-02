package storagetypes

import (
	"errors"
	"testing"
)

func TestSecureJoin(t *testing.T) {
	tests := []struct {
		name    string
		base    string
		path    string
		want    string
		wantErr bool
	}{
		{name: "relative", base: "base", path: "dir/file.txt", want: "base/dir/file.txt"},
		{name: "empty base", base: "", path: "dir/file.txt", want: "dir/file.txt"},
		{name: "leading slash", base: "base", path: "/dir/file.txt", want: "base/dir/file.txt"},
		{name: "inner cleanup", base: "base", path: "dir/../file.txt", want: "base/file.txt"},
		{name: "backslashes", base: "base", path: `dir\file.txt`, want: "base/dir/file.txt"},
		{name: "traversal", base: "base", path: "../file.txt", wantErr: true},
		{name: "nested traversal", base: "base", path: "dir/../../file.txt", wantErr: true},
		{name: "absolute traversal", base: "base", path: "/../file.txt", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SecureJoin(tt.base, tt.path)
			if tt.wantErr {
				if !errors.Is(err, ErrPathEscape) {
					t.Fatalf("SecureJoin(%q, %q) error = %v, want ErrPathEscape", tt.base, tt.path, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("SecureJoin(%q, %q) failed: %v", tt.base, tt.path, err)
			}
			if got != tt.want {
				t.Fatalf("SecureJoin(%q, %q) = %q, want %q", tt.base, tt.path, got, tt.want)
			}
		})
	}
}
