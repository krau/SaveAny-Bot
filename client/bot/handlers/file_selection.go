package handlers

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/charmbracelet/log"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/msgelem"
	"github.com/krau/SaveAny-Bot/common/cache"
	"github.com/krau/SaveAny-Bot/common/i18n"
	"github.com/krau/SaveAny-Bot/common/i18n/i18nk"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/pkg/tcbdata"
	"github.com/krau/SaveAny-Bot/pkg/tfile"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/rs/xid"
)

const (
	fileSelectionPageSize          = 10
	fileSelectionButtonsPerRow     = 5
	fileSelectionMaxFilenameRunes  = 180
	fileSelectionActionToggle      = "toggle"
	fileSelectionActionAll         = "all"
	fileSelectionActionNone        = "none"
	fileSelectionActionInvert      = "invert"
	fileSelectionActionPage        = "page"
	fileSelectionActionConfirm     = "confirm"
	fileSelectionActionCancel      = "cancel"
	fileSelectionActionNoOperation = "noop"
)

type fileSelectionState struct {
	mu       sync.Mutex
	userID   int64
	files    []tfile.TGFileMessage
	selected []bool
	page     int
}

type fileSelectionSnapshot struct {
	files         []tfile.TGFileMessage
	selected      []bool
	page          int
	selectedCount int
}

func newFileSelectionState(userID int64, files []tfile.TGFileMessage) *fileSelectionState {
	selected := make([]bool, len(files))
	for index := range selected {
		selected[index] = true
	}
	return &fileSelectionState{
		userID:   userID,
		files:    files,
		selected: selected,
	}
}

func (s *fileSelectionState) snapshotLocked() fileSelectionSnapshot {
	selected := append([]bool(nil), s.selected...)
	selectedCount := 0
	for _, isSelected := range selected {
		if isSelected {
			selectedCount++
		}
	}
	return fileSelectionSnapshot{
		files:         append([]tfile.TGFileMessage(nil), s.files...),
		selected:      selected,
		page:          s.page,
		selectedCount: selectedCount,
	}
}

func (s *fileSelectionState) selectedFilesLocked() []tfile.TGFileMessage {
	files := make([]tfile.TGFileMessage, 0, len(s.files))
	for index, file := range s.files {
		if s.selected[index] {
			files = append(files, file)
		}
	}
	return files
}

func startFileSelection(userID int64, files []tfile.TGFileMessage) (string, *tg.ReplyInlineMarkup, error) {
	if len(files) < 2 {
		return "", nil, fmt.Errorf("file selection requires at least 2 files, got %d", len(files))
	}

	state := newFileSelectionState(userID, files)
	dataID := xid.New().String()
	ttl := time.Duration(config.C().Cache.FileSelectionTTL) * time.Second
	if err := cache.SetWithTTL(dataID, state, ttl); err != nil {
		return "", nil, fmt.Errorf("failed to cache file selection: %w", err)
	}

	state.mu.Lock()
	snapshot := state.snapshotLocked()
	state.mu.Unlock()
	text, markup := buildFileSelectionMessage(dataID, snapshot)
	return text, markup, nil
}

func buildFileSelectionMessage(dataID string, snapshot fileSelectionSnapshot) (string, *tg.ReplyInlineMarkup) {
	totalPages := max((len(snapshot.files)+fileSelectionPageSize-1)/fileSelectionPageSize, 1)
	page := min(max(snapshot.page, 0), totalPages-1)
	start := page * fileSelectionPageSize
	end := min(start+fileSelectionPageSize, len(snapshot.files))

	lines := make([]string, 0, end-start)
	indexButtons := make([]tg.KeyboardButtonClass, 0, end-start)
	for index := start; index < end; index++ {
		mark := "❌"
		if snapshot.selected[index] {
			mark = "✅"
		}
		fileName := normalizeFileSelectionName(snapshot.files[index].Name())
		lines = append(lines, fmt.Sprintf("%s %d. %s", mark, index+1, fileName))
		indexButtons = append(indexButtons, fileSelectionButton(
			fmt.Sprintf("%s %d", mark, index+1),
			dataID,
			fmt.Sprintf("%s %d", fileSelectionActionToggle, index),
		))
	}

	text := i18n.T(i18nk.BotMsgFileSelectionPrompt, map[string]any{
		"Selected":      snapshot.selectedCount,
		"Total":         len(snapshot.files),
		"Page":          page + 1,
		"Pages":         totalPages,
		"MultiplePages": totalPages > 1,
		"Files":         strings.Join(lines, "\n"),
	})

	markup := &tg.ReplyInlineMarkup{}
	for index := 0; index < len(indexButtons); index += fileSelectionButtonsPerRow {
		markup.Rows = append(markup.Rows, tg.KeyboardButtonRow{
			Buttons: indexButtons[index:min(index+fileSelectionButtonsPerRow, len(indexButtons))],
		})
	}
	markup.Rows = append(markup.Rows, tg.KeyboardButtonRow{Buttons: []tg.KeyboardButtonClass{
		fileSelectionButton(i18n.T(i18nk.BotMsgFileSelectionButtonSelectAll), dataID, fileSelectionActionAll),
		fileSelectionButton(i18n.T(i18nk.BotMsgFileSelectionButtonSelectNone), dataID, fileSelectionActionNone),
		fileSelectionButton(i18n.T(i18nk.BotMsgFileSelectionButtonInvert), dataID, fileSelectionActionInvert),
	}})
	if totalPages > 1 {
		navigation := make([]tg.KeyboardButtonClass, 0, 3)
		if page > 0 {
			navigation = append(navigation, fileSelectionButton("⬅️", dataID, fmt.Sprintf("%s %d", fileSelectionActionPage, page-1)))
		}
		navigation = append(navigation, fileSelectionButton(
			i18n.T(i18nk.BotMsgFileSelectionButtonPage, map[string]any{"Page": page + 1, "Pages": totalPages}),
			dataID,
			fileSelectionActionNoOperation,
		))
		if page+1 < totalPages {
			navigation = append(navigation, fileSelectionButton("➡️", dataID, fmt.Sprintf("%s %d", fileSelectionActionPage, page+1)))
		}
		markup.Rows = append(markup.Rows, tg.KeyboardButtonRow{Buttons: navigation})
	}
	markup.Rows = append(markup.Rows, tg.KeyboardButtonRow{Buttons: []tg.KeyboardButtonClass{
		fileSelectionButton(i18n.T(i18nk.BotMsgFileSelectionButtonConfirm, map[string]any{"Count": snapshot.selectedCount}), dataID, fileSelectionActionConfirm),
		fileSelectionButton(i18n.T(i18nk.BotMsgFileSelectionButtonCancel), dataID, fileSelectionActionCancel),
	}})

	return text, markup
}

func fileSelectionButton(text, dataID, action string) *tg.KeyboardButtonCallback {
	return &tg.KeyboardButtonCallback{
		Text: text,
		Data: fmt.Appendf(nil, "%s %s %s", tcbdata.TypeFileSelect, dataID, action),
	}
}

func normalizeFileSelectionName(fileName string) string {
	fileName = strings.Join(strings.Fields(fileName), " ")
	if fileName == "" {
		return "-"
	}
	runes := []rune(fileName)
	if len(runes) <= fileSelectionMaxFilenameRunes {
		return fileName
	}
	return string(runes[:fileSelectionMaxFilenameRunes-1]) + "…"
}

func handleFileSelectionCallback(ctx *ext.Context, update *ext.Update) error {
	logger := log.FromContext(ctx)
	args := strings.Fields(string(update.CallbackQuery.Data))
	if len(args) < 3 {
		return answerFileSelectionCallback(ctx, update, i18n.T(i18nk.BotMsgFileSelectionErrorInvalidAction), true)
	}

	dataID := args[1]
	action := args[2]
	state, ok := cache.Get[*fileSelectionState](dataID)
	if !ok {
		return answerFileSelectionCallback(ctx, update, i18n.T(i18nk.BotMsgCommonErrorDataExpired), true)
	}
	if state.userID != update.CallbackQuery.GetUserID() {
		return answerFileSelectionCallback(ctx, update, i18n.T(i18nk.BotMsgCommonErrorNoPermission), true)
	}

	state.mu.Lock()
	switch action {
	case fileSelectionActionToggle:
		index, valid := parseFileSelectionIndex(args, len(state.files))
		if !valid {
			state.mu.Unlock()
			return answerFileSelectionCallback(ctx, update, i18n.T(i18nk.BotMsgFileSelectionErrorInvalidAction), true)
		}
		state.selected[index] = !state.selected[index]
	case fileSelectionActionAll:
		for index := range state.selected {
			state.selected[index] = true
		}
	case fileSelectionActionNone:
		for index := range state.selected {
			state.selected[index] = false
		}
	case fileSelectionActionInvert:
		for index := range state.selected {
			state.selected[index] = !state.selected[index]
		}
	case fileSelectionActionPage:
		page, valid := parseFileSelectionIndex(args, max((len(state.files)+fileSelectionPageSize-1)/fileSelectionPageSize, 1))
		if !valid {
			state.mu.Unlock()
			return answerFileSelectionCallback(ctx, update, i18n.T(i18nk.BotMsgFileSelectionErrorInvalidAction), true)
		}
		state.page = page
	case fileSelectionActionConfirm:
		selectedFiles := state.selectedFilesLocked()
		state.mu.Unlock()
		if len(selectedFiles) == 0 {
			return answerFileSelectionCallback(ctx, update, i18n.T(i18nk.BotMsgFileSelectionErrorNoFilesSelected), true)
		}
		markup, err := msgelem.BuildAddSelectStorageKeyboard(storage.GetUserStorages(ctx, state.userID), tcbdata.Add{
			Files:   selectedFiles,
			AsBatch: len(selectedFiles) > 1,
		})
		if err != nil {
			logger.Errorf("Failed to build storage selection keyboard: %s", err)
			return answerFileSelectionCallback(ctx, update, i18n.T(i18nk.BotMsgCommonErrorBuildStorageSelectKeyboardFailed, map[string]any{"Error": err.Error()}), true)
		}
		if err := acknowledgeFileSelectionCallback(ctx, update); err != nil {
			logger.Errorf("Failed to answer file selection callback: %s", err)
		}
		if _, err := ctx.EditMessage(state.userID, &tg.MessagesEditMessageRequest{
			ID:          update.CallbackQuery.GetMsgID(),
			Message:     buildFoundFilesSelectStorageMessage(fileNamesFromTGFiles(selectedFiles)),
			ReplyMarkup: markup,
		}); err != nil {
			logger.Errorf("Failed to edit file selection message: %s", err)
		} else {
			cache.Delete(dataID)
		}
		return dispatcher.EndGroups
	case fileSelectionActionCancel:
		state.mu.Unlock()
		if err := acknowledgeFileSelectionCallback(ctx, update); err != nil {
			logger.Errorf("Failed to answer file selection callback: %s", err)
		}
		if _, err := ctx.EditMessage(state.userID, &tg.MessagesEditMessageRequest{
			ID:      update.CallbackQuery.GetMsgID(),
			Message: i18n.T(i18nk.BotMsgFileSelectionInfoCancelled),
		}); err != nil {
			logger.Errorf("Failed to cancel file selection: %s", err)
		} else {
			cache.Delete(dataID)
		}
		return dispatcher.EndGroups
	case fileSelectionActionNoOperation:
		state.mu.Unlock()
		if err := acknowledgeFileSelectionCallback(ctx, update); err != nil {
			logger.Errorf("Failed to answer file selection callback: %s", err)
		}
		return dispatcher.EndGroups
	default:
		state.mu.Unlock()
		return answerFileSelectionCallback(ctx, update, i18n.T(i18nk.BotMsgFileSelectionErrorInvalidAction), true)
	}

	snapshot := state.snapshotLocked()
	state.mu.Unlock()
	text, markup := buildFileSelectionMessage(dataID, snapshot)
	if err := acknowledgeFileSelectionCallback(ctx, update); err != nil {
		logger.Errorf("Failed to answer file selection callback: %s", err)
	}
	if _, err := ctx.EditMessage(state.userID, &tg.MessagesEditMessageRequest{
		ID:          update.CallbackQuery.GetMsgID(),
		Message:     text,
		ReplyMarkup: markup,
	}); err != nil {
		logger.Errorf("Failed to update file selection message: %s", err)
	}
	return dispatcher.EndGroups
}

func parseFileSelectionIndex(args []string, limit int) (int, bool) {
	if len(args) < 4 {
		return 0, false
	}
	index, err := strconv.Atoi(args[3])
	if err != nil || index < 0 || index >= limit {
		return 0, false
	}
	return index, true
}

func fileNamesFromTGFiles(files []tfile.TGFileMessage) []string {
	fileNames := make([]string, 0, len(files))
	for _, file := range files {
		fileNames = append(fileNames, file.Name())
	}
	return fileNames
}

func acknowledgeFileSelectionCallback(ctx *ext.Context, update *ext.Update) error {
	_, err := ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
		QueryID: update.CallbackQuery.GetQueryID(),
	})
	return err
}

func answerFileSelectionCallback(ctx *ext.Context, update *ext.Update, message string, alert bool) error {
	if _, err := ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
		QueryID: update.CallbackQuery.GetQueryID(),
		Alert:   alert,
		Message: message,
	}); err != nil {
		log.FromContext(ctx).Errorf("Failed to answer file selection callback: %s", err)
	}
	return dispatcher.EndGroups
}
