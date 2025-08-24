package fsutil_test

import (
	"testing"

	"github.com/krau/SaveAny-Bot/common/utils/fsutil"
)

func TestNormalizePathname(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "hello/world?.txt  ",
			expected: "hello_world_.txt",
		},
		{
			input:    "bad|name:\nfile\r.",
			expected: "bad_name__file",
		},
		{
			input:    "normal.txt",
			expected: "normal.txt",
		},
		{
			input:    "test....   ",
			expected: "test",
		},
		{
			input:    "abc<>def",
			expected: "abc__def",
		},
		{
			input:    "with\tcontrol",
			expected: "with_control",
		},
	}

	for _, tc := range tests {
		got := fsutil.NormalizePathname(tc.input)
		if got != tc.expected {
			t.Errorf("NormalizePathname(%q) = %q; want %q", tc.input, got, tc.expected)
		}
	}
}
