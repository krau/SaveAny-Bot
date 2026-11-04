package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/parsers"
)

type parserFileResponse struct {
	Name      string    `json:"name"`
	Directory string    `json:"directory"`
	Size      int64     `json:"size"`
	Modified  time.Time `json:"modified"`
}

type parserListResponse struct {
	Enabled bool                 `json:"enabled"`
	Dirs    []string             `json:"dirs"`
	Files   []parserFileResponse `json:"files"`
}

func parserDirs() []string {
	cfg := config.C()
	if len(cfg.Parser.PluginDirs) > 0 {
		return cfg.Parser.PluginDirs
	}
	return []string{"plugins"}
}

func (s *server) listParsers(w http.ResponseWriter, r *http.Request) {
	cfg := config.C()
	dirs := parserDirs()
	files := make([]parserFileResponse, 0)
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			WriteError(w, http.StatusInternalServerError, "read_failed", err.Error())
			return
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".js") {
				continue
			}
			info, err := entry.Info()
			if err != nil {
				continue
			}
			files = append(files, parserFileResponse{
				Name:      entry.Name(),
				Directory: dir,
				Size:      info.Size(),
				Modified:  info.ModTime(),
			})
		}
	}
	WriteJSON(w, http.StatusOK, parserListResponse{Enabled: cfg.Parser.PluginEnable, Dirs: dirs, Files: files})
}

func (s *server) createParser(w http.ResponseWriter, r *http.Request) {
	if !config.C().Parser.PluginEnable {
		WriteError(w, http.StatusConflict, "parser_disabled", "parser plugins are disabled")
		return
	}
	var body struct {
		Filename string `json:"filename"`
		Content  string `json:"content"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 10<<20)).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}
	filename := filepath.Base(strings.TrimSpace(body.Filename))
	if filename == "" || filename == "." || !strings.HasSuffix(strings.ToLower(filename), ".js") {
		WriteFieldErrors(w, http.StatusBadRequest, "filename must end with .js", map[string]string{"filename": "invalid filename"})
		return
	}
	if body.Content == "" {
		WriteFieldErrors(w, http.StatusBadRequest, "content is required", map[string]string{"content": "required"})
		return
	}
	if err := parsers.AddPlugin(r.Context(), body.Content, filename); err != nil {
		WriteError(w, http.StatusBadRequest, "install_failed", err.Error())
		return
	}
	WriteJSON(w, http.StatusCreated, map[string]any{"name": filename})
}
