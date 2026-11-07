//go:build no_playwright && !no_jsparser

package js

import (
	"github.com/MoeForge/moejs"
	"github.com/charmbracelet/log"
)

var jsPlaywright = func(vm *moejs.Runtime, _ *log.Logger) map[string]any {
	return map[string]any{
		"get": moejs.NativeFunc(func(_ *moejs.Realm, _ moejs.Value, _ []moejs.Value) (moejs.Value, error) {
			return vm.FromGo(map[string]any{
				"error": "playwright is not supported in this build",
			})
		}),
	}
}
