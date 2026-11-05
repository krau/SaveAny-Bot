package api

import (
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/blang/semver"
	"github.com/krau/SaveAny-Bot/common/selfupdate"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/unvgo/ghselfupdate"
)

var updateMu sync.Mutex

type updateInfoResponse struct {
	Current     string     `json:"current"`
	Latest      string     `json:"latest,omitempty"`
	Available   bool       `json:"available"`
	Updatable   bool       `json:"updatable"`
	Docker      bool       `json:"docker"`
	ReleaseNote string     `json:"release_notes,omitempty"`
	URL         string     `json:"url,omitempty"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
	Size        int64      `json:"size,omitempty"`
	Reason      string     `json:"reason,omitempty"`
	ReasonCode  string     `json:"reason_code,omitempty"`
}

func checkUpdate() (*updateInfoResponse, error) {
	info := &updateInfoResponse{
		Current: config.Version,
		Docker:  config.Docker == "true",
	}
	current, err := semver.Parse(config.Version)
	if err != nil {
		info.Reason = err.Error()
		info.ReasonCode = "invalid_version"
		return info, nil
	}
	info.Current = current.String()
	if err := selfupdate.Updatable(); err != nil {
		info.Reason = err.Error()
		switch {
		case errors.Is(err, selfupdate.ErrDocker):
			info.ReasonCode = "docker"
		case errors.Is(err, selfupdate.ErrDevBuild):
			info.ReasonCode = "dev_build"
		case errors.Is(err, selfupdate.ErrNotWritable):
			info.ReasonCode = "not_writable"
		default:
			info.ReasonCode = "not_updatable"
		}
		return info, nil
	}
	info.Updatable = true
	latest, ok, err := ghselfupdate.DetectLatest(config.GitRepo)
	if err != nil {
		return nil, err
	}
	if !ok {
		info.ReasonCode = "no_release"
		info.Reason = "no release found"
		return info, nil
	}
	info.Latest = latest.Version.String()
	info.ReleaseNote = latest.ReleaseNotes
	info.URL = latest.AssetURL
	info.Size = int64(latest.AssetByteSize)
	info.PublishedAt = latest.PublishedAt
	info.Available = latest.Version.Major == current.Major && latest.Version.GT(current)
	if latest.Version.Major != current.Major {
		info.Reason = "major upgrade required"
		info.ReasonCode = "major_upgrade_required"
	}
	return info, nil
}

func (s *server) getUpdateInfo(w http.ResponseWriter, r *http.Request) {
	info, err := checkUpdate()
	if err != nil {
		WriteError(w, http.StatusBadGateway, "update_check_failed", err.Error())
		return
	}
	WriteJSON(w, http.StatusOK, info)
}

func (s *server) applyUpdate(w http.ResponseWriter, r *http.Request) {
	updateMu.Lock()
	defer updateMu.Unlock()

	current, err := semver.Parse(config.Version)
	if err != nil {
		WriteError(w, http.StatusConflict, "not_updatable", err.Error())
		return
	}
	if err := selfupdate.Updatable(); err != nil {
		WriteError(w, http.StatusConflict, "not_updatable", err.Error())
		return
	}
	latest, ok, err := ghselfupdate.DetectLatest(config.GitRepo)
	if err != nil {
		WriteError(w, http.StatusBadGateway, "update_check_failed", err.Error())
		return
	}
	if !ok {
		WriteError(w, http.StatusNotFound, "no_release", "no release found")
		return
	}
	if latest.Version.Major != current.Major {
		WriteError(w, http.StatusConflict, "major_upgrade_required", "major upgrade required")
		return
	}
	if !latest.Version.GT(current) {
		WriteError(w, http.StatusConflict, "already_latest", "already on latest version")
		return
	}
	release, err := selfupdate.Update(current)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "update_failed", err.Error())
		return
	}
	WriteJSON(w, http.StatusAccepted, map[string]any{
		"version":    release.Version.String(),
		"restarting": true,
	})
	go func() {
		time.Sleep(200 * time.Millisecond)
		s.deps.Telegram.RestartProcess()
	}()
}
