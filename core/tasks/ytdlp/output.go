package ytdlp

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/krau/SaveAny-Bot/config"
)

// splitOutputTemplate pulls a user-supplied yt-dlp output template out of the
// custom flags: the last one wins, matching yt-dlp. A -o/--output without a
// template is rejected, since yt-dlp would consume the following argument (a URL
// or another flag) as the template.
func splitOutputTemplate(flags []string) (template string, rest []string, err error) {
	rest = make([]string, 0, len(flags))
	for i := 0; i < len(flags); i++ {
		flag := flags[i]
		if flag == "-o" || flag == "--output" {
			if i+1 == len(flags) || strings.HasPrefix(flags[i+1], "-") {
				return "", nil, fmt.Errorf("%s requires an output template", flag)
			}
			i++
			template = flags[i]
			continue
		}
		switch {
		case strings.HasPrefix(flag, "--output="):
			template = strings.TrimPrefix(flag, "--output=")
		case strings.HasPrefix(flag, "-o") && len(flag) > 2:
			template = flag[2:]
		default:
			rest = append(rest, flag)
		}
	}
	return template, rest, nil
}

// resolveFilenameTemplate picks the output template: a user-provided
// -o/--output, then ytdlp.filename_template, then the built-in default.
func resolveFilenameTemplate(cfg config.YtdlpConfig, userTemplate string) string {
	switch {
	case userTemplate != "":
		return userTemplate
	case cfg.FilenameTemplate != "":
		return cfg.FilenameTemplate
	default:
		return config.DefaultYtdlpFilenameTemplate
	}
}

// ytdlpOutputTypes are the type prefixes yt-dlp accepts in output templates
// ("[TYPES:]TEMPLATE", see OUTTMPL_TYPES in yt-dlp). They must stay in front of
// the resolved path instead of becoming part of it.
var ytdlpOutputTypes = map[string]struct{}{
	"annotation": {}, "chapter": {}, "description": {}, "infojson": {},
	"link": {}, "pl_description": {}, "pl_infojson": {}, "pl_thumbnail": {},
	"pl_video": {}, "subtitle": {}, "thumbnail": {},
}

// outputTemplatePath resolves an output template below dir, the directory the bot
// owns. Templates that would escape dir through ".." are rejected.
func outputTemplatePath(dir, template string) (string, error) {
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
	for _, k := range strings.Split(key, "+") {
		if _, known := ytdlpOutputTypes[k]; !known {
			return "", template
		}
	}
	return key + ":", rest
}
