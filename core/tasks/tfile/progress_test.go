package tfile

import (
	"testing"
	"time"
)

func TestShouldUpdateUploadProgress(t *testing.T) {
	tests := []struct {
		name        string
		total       int64
		uploaded    int64
		lastPercent int
		elapsed     time.Duration
		want        bool
	}{
		{name: "invalid total", total: 0, uploaded: 1, want: false},
		{name: "no uploaded bytes", total: 100, uploaded: 0, want: false},
		{name: "percentage threshold", total: 100 << 20, uploaded: 10 << 20, elapsed: uploadProgressMinInterval, want: true},
		{name: "percentage threshold rate limited", total: 100 << 20, uploaded: 10 << 20, elapsed: uploadProgressMinInterval - time.Millisecond, want: false},
		{name: "maximum time threshold", total: 100 << 20, uploaded: 1 << 20, elapsed: uploadProgressMaxInterval, want: true},
		{name: "below thresholds", total: 100 << 20, uploaded: 1 << 20, elapsed: uploadProgressMaxInterval - time.Millisecond, want: false},
		{name: "completion", total: 100, uploaded: 100, lastPercent: 99, elapsed: uploadProgressMinInterval, want: true},
		{name: "completion rate limited", total: 100, uploaded: 100, lastPercent: 99, elapsed: uploadProgressMinInterval - time.Millisecond, want: false},
		{name: "completion already reported", total: 100, uploaded: 100, lastPercent: 100, elapsed: uploadProgressMinInterval, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldUpdateUploadProgress(tt.total, tt.uploaded, tt.lastPercent, tt.elapsed)
			if got != tt.want {
				t.Fatalf("shouldUpdateUploadProgress() = %v, want %v", got, tt.want)
			}
		})
	}
}
