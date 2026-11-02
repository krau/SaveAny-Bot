package relay

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/pkg/tfile"
)

const resumeStateVersion = 1

type resumeState struct {
	Version          int    `json:"version"`
	RelayChatID      int64  `json:"relay_chat_id"`
	RelayMessageID   int    `json:"relay_message_id"`
	SourceKind       string `json:"source_kind"`
	SourceID         int64  `json:"source_id"`
	SourceAccessHash int64  `json:"source_access_hash"`
}

// ResumeStatePath returns the sidecar path used to keep a relay message alive
// across a crash. It deliberately lives next to the download cache.
func ResumeStatePath(cachePath string) string {
	return cachePath + ".relay"
}

func loadResumeState(path string) (*resumeState, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read relay resume state: %w", err)
	}
	var state resumeState
	if err := json.Unmarshal(data, &state); err != nil {
		_ = os.Remove(path)
		return nil, nil
	}
	if state.Version != resumeStateVersion || state.RelayMessageID == 0 || state.RelayChatID == 0 {
		_ = os.Remove(path)
		return nil, nil
	}
	return &state, nil
}

func saveResumeState(path string, state resumeState) error {
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshal relay resume state: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write relay resume state: %w", err)
	}
	return os.Rename(tmp, path)
}

func removeResumeState(path string) error {
	if path == "" {
		return nil
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.Remove(path + ".tmp"); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// CleanupResumeState releases a persisted relay message after the cache was
// already complete, for example if the process stopped between download and
// cleanup.
func CleanupResumeState(ctx context.Context, statePath string) {
	state, err := loadResumeState(statePath)
	if err != nil || state == nil {
		return
	}
	extCtx := tgutil.ExtFromContext(ctx)
	if extCtx == nil {
		_ = removeResumeState(statePath)
		return
	}
	relayed := &relayFile{
		primary:        extCtx,
		relayChatID:    state.RelayChatID,
		relayMessageID: state.RelayMessageID,
		keepMessages:   config.C().Telegram.RelayKeepMessages,
		statePath:      statePath,
	}
	relayed.cleanup(ctx)
}

func (s *resumeState) matches(identity tfile.FileIdentity) bool {
	if s == nil {
		return false
	}
	return s.SourceKind == identity.Kind && s.SourceID == identity.ID && s.SourceAccessHash == identity.AccessHash
}
