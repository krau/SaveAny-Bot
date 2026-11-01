package api

import (
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const pathListLimit = 500

type filesystemEntry struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	IsDir bool   `json:"is_dir"`
	Size  int64  `json:"size,omitempty"`
}

type filesystemListing struct {
	Path      string            `json:"path"`
	Parent    string            `json:"parent,omitempty"`
	Root      string            `json:"root"`
	Cwd       string            `json:"cwd"`
	Selected  string            `json:"selected,omitempty"`
	Entries   []filesystemEntry `json:"entries"`
	Truncated bool              `json:"truncated,omitempty"`
}

// listFilesystem lists one directory on the deployment host for the server-side
// path picker. It returns names and metadata only, never file contents.
func (s *server) listFilesystem(w http.ResponseWriter, r *http.Request) {
	requested := strings.TrimSpace(r.URL.Query().Get("path"))
	if requested == "" {
		cwd, err := os.Getwd()
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		requested = cwd
	}
	absolute, err := filepath.Abs(requested)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_path", err.Error())
		return
	}
	dir := absolute
	selected := ""
	info, err := os.Stat(absolute)
	switch {
	case err == nil:
		if !info.IsDir() {
			selected = absolute
			dir = filepath.Dir(absolute)
		}
	case os.IsNotExist(err):
		// A target file or directory may not exist yet. Start the picker at the
		// nearest existing ancestor instead of failing outright.
		for {
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
			if parentInfo, statErr := os.Stat(dir); statErr == nil && parentInfo.IsDir() {
				break
			}
		}
		if _, statErr := os.Stat(dir); statErr != nil {
			WriteError(w, http.StatusNotFound, "path_not_found", statErr.Error())
			return
		}
	case os.IsPermission(err):
		WriteError(w, http.StatusForbidden, "path_forbidden", err.Error())
		return
	default:
		WriteError(w, http.StatusBadRequest, "path_unavailable", err.Error())
		return
	}

	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		status := http.StatusBadRequest
		code := "path_unavailable"
		if os.IsPermission(err) {
			status = http.StatusForbidden
			code = "path_forbidden"
		}
		WriteError(w, status, code, err.Error())
		return
	}

	entries := make([]filesystemEntry, 0, len(dirEntries))
	for _, entry := range dirEntries {
		entryInfo, err := entry.Info()
		if err != nil {
			continue
		}
		entries = append(entries, filesystemEntry{
			Name:  entry.Name(),
			Path:  filepath.Join(dir, entry.Name()),
			IsDir: entryInfo.IsDir(),
			Size:  entryInfo.Size(),
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})
	truncated := false
	if len(entries) > pathListLimit {
		entries = entries[:pathListLimit]
		truncated = true
	}

	parent := filepath.Dir(dir)
	if parent == dir {
		parent = ""
	}
	root := string(filepath.Separator)
	if volume := filepath.VolumeName(dir); volume != "" {
		root = volume + string(filepath.Separator)
	}
	cwd, _ := os.Getwd()
	w.Header().Set("Cache-Control", "no-store")
	WriteJSON(w, http.StatusOK, filesystemListing{
		Path:      dir,
		Parent:    parent,
		Root:      root,
		Cwd:       cwd,
		Selected:  selected,
		Entries:   entries,
		Truncated: truncated,
	})
}
