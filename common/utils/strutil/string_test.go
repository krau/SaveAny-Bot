package strutil_test

import (
	"reflect"
	"testing"

	"github.com/krau/SaveAny-Bot/common/utils/strutil"
)

func TestExtractTagsFromText(t *testing.T) {
	tests := []struct {
		text     string
		expected []string
	}{
		{
			text: `初音ミクHappy 16th Birthday -Dear Creators-
			✨エンドイラスト公開！✨
			https://piapro.net/miku16thbd/
			#初音ミク #miku16th`,
			expected: []string{"初音ミク", "miku16th"},
		},
		{
			text: `ひっつきむし
			#創作百合`,
			expected: []string{"創作百合"},
		},
		{
			text:     `#創作百合 #原创`,
			expected: []string{"創作百合", "原创"},
		},
		{
			text:     `プラニャ　#ブルアカ`,
			expected: []string{"ブルアカ"},
		},
		{
			text:     `原神是一款#开放世界#冒险游戏，由中国著名游戏公司#miHoYo开发。`,
			expected: []string{},
		},
	}

	for _, test := range tests {
		result := strutil.ExtractTagsFromText(test.text)
		if !reflect.DeepEqual(result, test.expected) {
			t.Fatalf("ExtractTagsFromText(%s) = %v, expected %v", test.text, result, test.expected)
		}
	}
}

func TestParseIntStrRange(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		sep     string
		wantMin int64
		wantMax int64
		wantErr bool
	}{
		{
			name:    "normal range",
			input:   "10-20",
			sep:     "-",
			wantMin: 10,
			wantMax: 20,
		},
		{
			name:    "reverse order",
			input:   "30 - 10",
			sep:     "-",
			wantMin: 10,
			wantMax: 30,
		},
		{
			name:    "invalid format",
			input:   "10",
			sep:     "-",
			wantErr: true,
		},
		{
			name:    "invalid number",
			input:   "a-b",
			sep:     "-",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			min, max, err := strutil.ParseIntStrRange(tt.input, tt.sep)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseIntStrRange(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if min != tt.wantMin || max != tt.wantMax {
					t.Errorf("ParseIntStrRange(%q) = (%d, %d), want (%d, %d)", tt.input, min, max, tt.wantMin, tt.wantMax)
				}
			}
		})
	}
}

func TestParseArgsRespectQuotes(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "simple split",
			input: `/rule add FILENAME-REGEX (?i)\.(mp4|mkv)$ "我的 Alist" /视频`,
			want:  []string{"/rule", "add", "FILENAME-REGEX", "(?i)\\.(mp4|mkv)$", "我的 Alist", "/视频"},
		},
		{
			name:  "escaped quotes",
			input: `/rule add "My \"Awesome\" Folder"`,
			want:  []string{"/rule", "add", `My "Awesome" Folder`},
		},
		{
			name:  "escaped backslash",
			input: `/cmd "C:\\Users\\Admin" test`,
			want:  []string{"/cmd", `C:\Users\Admin`, "test"},
		},
		{
			name:  "multiple quoted parts",
			input: `"Hello World" "你好 世界"`,
			want:  []string{"Hello World", "你好 世界"},
		},
		{
			name:  "unquoted words",
			input: "a b c",
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "mixed quotes and plain",
			input: `cmd "quoted arg" plain`,
			want:  []string{"cmd", "quoted arg", "plain"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := strutil.ParseArgsRespectQuotes(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseArgsRespectQuotes(%q) = %#v, want %#v", tt.input, got, tt.want)
			}
		})
	}
}
