package js_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/krau/SaveAny-Bot/parsers/js"
)

// Regression: a plugin with an invalid semver version must be rejected
// without panicking the process (previously semver.MustParse crashed the bot).
func TestLoadPluginsRejectsInvalidVersion(t *testing.T) {
	dir := t.TempDir()
	bad := `registerParser({
		metadata: { name: "probe", version: "not-a-version", description: "", author: "" },
		canHandle: function(url) { return true; },
		parse: async function(url) { return { resources: [] }; }
	});`
	if err := os.WriteFile(filepath.Join(dir, "bad.js"), []byte(bad), 0o644); err != nil {
		t.Fatal(err)
	}
	good := `registerParser({
		metadata: { name: "good", version: "1.0.0", description: "", author: "" },
		canHandle: function(url) { return false; },
		parse: async function(url) { return { resources: [] }; }
	});`
	if err := os.WriteFile(filepath.Join(dir, "good.js"), []byte(good), 0o644); err != nil {
		t.Fatal(err)
	}
	// Must not panic; the bad plugin is skipped and remaining plugins load.
	if err := js.LoadPlugins(t.Context(), dir); err != nil {
		t.Fatalf("LoadPlugins returned error: %v", err)
	}
}
