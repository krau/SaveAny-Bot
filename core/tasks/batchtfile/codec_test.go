package batchtfile

import (
	"testing"
)

func TestDetectTaskKind(t *testing.T) {
	tests := []struct {
		name    string
		payload string
		want    string
		wantErr bool
	}{
		{"batch with kind", `{"kind":"batch","id":"1","elements":[]}`, "batch", false},
		{"file with kind", `{"kind":"file","id":"1","file":{}}`, "file", false},
		{"legacy batch by shape", `{"id":"1","elements":[]}`, "batch", false},
		{"legacy file by shape", `{"id":"1","file":{}}`, "file", false},
		{"legacy batch with element", `{"id":"1","elements":[{"id":"e"}]}`, "batch", false},
		{"no discriminator", `{"id":"1"}`, "", true},
		{"invalid json", `not json`, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := detectTaskKind([]byte(tt.payload))
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got kind %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("kind = %q, want %q", got, tt.want)
			}
		})
	}
}
