package ytdlp

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/krau/SaveAny-Bot/config"
)

// rewriteOutputTemplates re-roots every -o/--output template in the custom flags
// below dir, the directory the bot owns. Templates keep their yt-dlp type prefix
// and their position, so repeated templates still override each other the way
// yt-dlp expects. A -o/--output without a template is rejected, since yt-dlp
// would consume the following argument (a URL or another flag) as the template.
// forbiddenYtdlpFlags can execute programs, load configuration, or read and
// write arbitrary files, so they are never accepted from user input.
var forbiddenYtdlpFlags = map[string]struct{}{
	"--exec":                 {},
	"--exec-before-download": {},
	"--config-location":      {},
	"--config-locations":     {},
	"--load-info-json":       {},
	"--batch-file":           {},
	"--downloader":           {},
	"--external-downloader":  {},
	"--paths":                {},
	"--cache-dir":            {},
	"--postprocessor-args":   {},
	"--ppa":                  {},
	"--netrc":                {},
	"--netrc-cmd":            {},
	"--proxy":                {},
	"--cookies":              {},
	"--cookies-from-browser": {},
	"--plugin-dirs":          {},
	"--update":               {},
	"--update-to":            {},
}

func validateCustomFlags(flags []string) error {
	for _, flag := range flags {
		name := flag
		if before, _, ok := strings.Cut(flag, "="); ok {
			name = before
		}
		if _, forbidden := forbiddenYtdlpFlags[name]; forbidden {
			return fmt.Errorf("yt-dlp flag %s is not allowed", name)
		}
		if name == "-P" || strings.HasPrefix(name, "-P") || name == "-U" {
			return fmt.Errorf("yt-dlp flag %s is not allowed", name)
		}
	}
	return nil
}

func rewriteOutputTemplates(flags []string, dir string) ([]string, error) {
	rewritten := make([]string, 0, len(flags))
	for i := 0; i < len(flags); i++ {
		flag := flags[i]
		if flag == "-o" || flag == "--output" {
			if i+1 == len(flags) || strings.HasPrefix(flags[i+1], "-") {
				return nil, fmt.Errorf("%s requires an output template", flag)
			}
			i++
			template, err := outputTemplatePath(dir, flags[i])
			if err != nil {
				return nil, err
			}
			rewritten = append(rewritten, flag, template)
			continue
		}
		switch {
		case strings.HasPrefix(flag, "--output="):
			template, err := outputTemplatePath(dir, strings.TrimPrefix(flag, "--output="))
			if err != nil {
				return nil, err
			}
			rewritten = append(rewritten, "--output="+template)
		case strings.HasPrefix(flag, "-o") && len(flag) > len("-o"):
			template, err := outputTemplatePath(dir, flag[len("-o"):])
			if err != nil {
				return nil, err
			}
			rewritten = append(rewritten, "-o"+template)
		default:
			rewritten = append(rewritten, flag)
		}
	}
	return rewritten, nil
}

// resolveFilenameTemplate returns the template outputs fall back to: the
// ytdlp.filename_template config, or the built-in default.
func resolveFilenameTemplate(cfg config.YtdlpConfig) string {
	if cfg.FilenameTemplate != "" {
		return cfg.FilenameTemplate
	}
	return config.DefaultYtdlpFilenameTemplate
}

// ytdlpOutputTypes are the type prefixes yt-dlp accepts in output templates
// ("[TYPES:]TEMPLATE", see OUTTMPL_TYPES in yt-dlp). They must stay in front of
// the resolved path instead of becoming part of it.
var ytdlpOutputTypes = map[string]struct{}{
	"annotation": {}, "chapter": {}, "description": {}, "infojson": {},
	"link": {}, "pl_description": {}, "pl_infojson": {}, "pl_thumbnail": {},
	"pl_video": {}, "subtitle": {}, "thumbnail": {},
}

// outputTemplatePath resolves an output template below dir. Templates that would
// escape dir through ".." are rejected.
func outputTemplatePath(dir, template string) (string, error) {
	if template == "" {
		return "", errors.New("empty output template")
	}
	prefix, tmpl := splitTypePrefix(template)
	output := filepath.Join(dir, tmpl)
	rel, err := filepath.Rel(dir, output)
	if err != nil {
		return "", fmt.Errorf("failed to resolve output template %q: %w", template, err)
	}
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("output template %q escapes the download directory", template)
	}
	return prefix + output, nil
}

// splitTypePrefix splits a leading yt-dlp template type prefix, e.g. "subtitle:"
// or "subtitle+thumbnail:".
func splitTypePrefix(template string) (prefix, rest string) {
	key, rest, ok := strings.Cut(template, ":")
	if !ok {
		return "", template
	}
	for k := range strings.SplitSeq(key, "+") {
		if _, known := ytdlpOutputTypes[k]; !known {
			return "", template
		}
	}
	return key + ":", rest
}
