package handlers

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gotd/td/telegram/downloader"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/common/i18n"
	"github.com/krau/SaveAny-Bot/pkg/tcbdata"
	"github.com/krau/SaveAny-Bot/pkg/tfile"
)

type testFileSelectionFile struct {
	name string
}

func (f *testFileSelectionFile) Location() tg.InputFileLocationClass { return nil }
func (f *testFileSelectionFile) Dler() downloader.Client             { return nil }
func (f *testFileSelectionFile) Size() int64                         { return 0 }
func (f *testFileSelectionFile) Name() string                        { return f.name }
func (f *testFileSelectionFile) SetName(name string)                 { f.name = name }
func (f *testFileSelectionFile) Message() *tg.Message                { return nil }

func TestBuildFileSelectionMessage(t *testing.T) {
	i18n.Init("zh-Hans")

	files := makeFileSelectionTestFiles(12)
	state := newFileSelectionState(123, files)
	state.mu.Lock()
	state.selected[2] = false
	snapshot := state.snapshotLocked()
	state.mu.Unlock()

	text, markup := buildFileSelectionMessage("selection-id", snapshot)

	for _, expected := range []string{
		"已选择 11/12",
		"第 1/2 页",
		"✅ 1. file-01.mp4",
		"❌ 3. file-03.mp4",
	} {
		if !strings.Contains(text, expected) {
			t.Errorf("expected message to contain %q, got %q", expected, text)
		}
	}
	if strings.Contains(text, "file-11.mp4") {
		t.Fatalf("expected the first page to omit file 11, got %q", text)
	}
	if len(markup.Rows) != 5 {
		t.Fatalf("expected 5 keyboard rows, got %d", len(markup.Rows))
	}

	for _, row := range markup.Rows {
		for _, button := range row.Buttons {
			callback, ok := button.(*tg.KeyboardButtonCallback)
			if !ok {
				t.Fatalf("unexpected button type %T", button)
			}
			if len(callback.Data) > 64 {
				t.Errorf("callback data exceeds Telegram limit: %q", callback.Data)
			}
			if !strings.HasPrefix(string(callback.Data), tcbdata.TypeFileSelect+" selection-id ") {
				t.Errorf("unexpected callback data: %q", callback.Data)
			}
		}
	}
}

func TestNormalizeFileSelectionName(t *testing.T) {
	name := "first\nsecond\t" + strings.Repeat("长", fileSelectionMaxFilenameRunes)
	normalized := normalizeFileSelectionName(name)

	if strings.ContainsAny(normalized, "\n\t") {
		t.Fatalf("expected a single-line filename, got %q", normalized)
	}
	if !strings.HasSuffix(normalized, "…") {
		t.Fatalf("expected a truncated filename, got %q", normalized)
	}
}

func makeFileSelectionTestFiles(count int) []tfile.TGFileMessage {
	files := make([]tfile.TGFileMessage, count)
	for index := range files {
		files[index] = &testFileSelectionFile{name: fmt.Sprintf("file-%02d.mp4", index+1)}
	}
	return files
}
