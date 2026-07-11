package handlers

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/krau/SaveAny-Bot/common/i18n"
)

func TestBuildFoundFilesSelectStorageMessage(t *testing.T) {
	i18n.Init("zh-Hans")

	message := buildFoundFilesSelectStorageMessage([]string{
		"first.mp4",
		"second\nfile.jpg",
		"third.zip",
	})

	for _, expected := range []string{
		"找到 3 个文件:",
		"1. first.mp4",
		"2. second file.jpg",
		"3. third.zip",
		"请选择存储位置",
	} {
		if !strings.Contains(message, expected) {
			t.Errorf("expected message to contain %q, got %q", expected, message)
		}
	}
}

func TestBuildFoundFilesSelectStorageMessageTruncatesLongLists(t *testing.T) {
	i18n.Init("zh-Hans")

	fileNames := make([]string, 100)
	for index := range fileNames {
		fileNames[index] = fmt.Sprintf("%03d-%s.mp4", index, strings.Repeat("长文件名", 40))
	}

	message := buildFoundFilesSelectStorageMessage(fileNames)

	if utf8.RuneCountInString(message) > maxTelegramMessageRunes {
		t.Fatalf("message has %d runes, limit is %d", utf8.RuneCountInString(message), maxTelegramMessageRunes)
	}
	if !strings.Contains(message, "个文件未显示") {
		t.Fatalf("expected truncated file count in message, got %q", message)
	}
	if strings.Contains(message, fileNames[len(fileNames)-1]) {
		t.Fatalf("expected the last file to be omitted from the truncated message")
	}
}
