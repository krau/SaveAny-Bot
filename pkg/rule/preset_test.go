package rule

import (
	"regexp"
	"testing"
)

func TestPresetCategoriesCompile(t *testing.T) {
	for _, c := range PresetCategories("") {
		if _, err := regexp.Compile(c.Regex); err != nil {
			t.Errorf("preset %q has invalid regex %q: %v", c.Name, c.Regex, err)
		}
	}
}

func TestPresetCategoriesMatch(t *testing.T) {
	cases := map[string]string{
		"video":    "movie.MP4",
		"image":    "photo.jpg",
		"audio":    "song.flac",
		"document": "report.pdf",
		"archive":  "backup.zip",
	}

	byName := make(map[string]*regexp.Regexp)
	for _, c := range PresetCategories("") {
		byName[c.Name] = regexp.MustCompile(c.Regex)
	}

	for name, filename := range cases {
		re, ok := byName[name]
		if !ok {
			t.Errorf("missing preset category %q", name)
			continue
		}
		if !re.MatchString(filename) {
			t.Errorf("preset %q did not match %q", name, filename)
		}
	}
}

func TestPresetCategoriesBasePath(t *testing.T) {
	presets := PresetCategories("/media")
	for _, c := range presets {
		if c.Dir == "" || c.Dir[0] != '/' {
			t.Errorf("preset %q dir %q not joined under base path", c.Name, c.Dir)
		}
	}
	// Empty base path must not prefix a separator.
	for _, c := range PresetCategories("") {
		if c.Dir == "" || c.Dir[0] == '/' {
			t.Errorf("preset %q dir %q should be relative when base path empty", c.Name, c.Dir)
		}
	}
}
